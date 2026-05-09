package lzcnotify

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	gohelper "gitee.com/linakesi/lzc-sdk/lang/go"
	"gitee.com/linakesi/lzc-sdk/lang/go/common"
	"gitee.com/linakesi/lzc-sdk/lang/go/localdevice"
	"google.golang.org/grpc"
)

const deviceAPITokenKey = "lzc_dapi_auth_token"

// NotifyUser sends a LazyCat built-in notification to all online clients of a user.
func NotifyUser(ctx context.Context, uid, title, body, deeplinkURL string) (int, error) {
	if uid == "" {
		return 0, errors.New("uid is required")
	}

	gw, err := gohelper.NewAPIGateway(ctx)
	if err != nil {
		return 0, fmt.Errorf("create api gateway: %w", err)
	}
	defer gw.Close()

	devices, err := gw.Devices.ListEndDevices(ctx, &common.ListEndDeviceRequest{Uid: uid})
	if err != nil {
		return 0, fmt.Errorf("list end devices: %w", err)
	}

	transportCred, err := gohelper.BuildClientCredOption(gohelper.CAPath, gohelper.APPKeyPath, gohelper.APPCertPath)
	if err != nil {
		return 0, fmt.Errorf("build device api credentials: %w", err)
	}

	request := &localdevice.NotifyRequest{
		Title: title,
		Body:  body,
	}
	if deeplinkURL != "" {
		request.DeeplinkUrl = &deeplinkURL
	}

	var sent int
	var sendErrs []error
	for _, device := range devices.GetDevices() {
		if !device.GetIsOnline() || device.GetDeviceApiUrl() == "" {
			continue
		}
		if err := notifyDevice(ctx, transportCred, device.GetDeviceApiUrl(), request); err != nil {
			sendErrs = append(sendErrs, fmt.Errorf("notify device %q: %w", device.GetName(), err))
			continue
		}
		sent++
	}

	return sent, errors.Join(sendErrs...)
}

func notifyDevice(ctx context.Context, transportCred grpc.DialOption, deviceAPIURL string, request *localdevice.NotifyRequest) error {
	parsedURL, err := url.Parse(deviceAPIURL)
	if err != nil {
		return fmt.Errorf("parse device api url: %w", err)
	}
	if parsedURL.Host == "" {
		return fmt.Errorf("device api url %q has no host", deviceAPIURL)
	}

	unauthConn, err := grpc.DialContext(ctx, parsedURL.Host, transportCred, grpc.WithBlock())
	if err != nil {
		return fmt.Errorf("dial device api for auth: %w", err)
	}
	authToken, err := gohelper.RequestAuthToken(ctx, unauthConn)
	closeErr := unauthConn.Close()
	if err != nil {
		return fmt.Errorf("request auth token: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("close auth connection: %w", closeErr)
	}

	conn, err := grpc.DialContext(
		ctx,
		parsedURL.Host,
		transportCred,
		grpc.WithPerRPCCredentials(staticTokenCredentials{token: authToken.Token}),
		grpc.WithBlock(),
	)
	if err != nil {
		return fmt.Errorf("dial device api: %w", err)
	}
	defer conn.Close()

	_, err = localdevice.NewNotificationServiceClient(conn).Notify(ctx, request)
	if err != nil {
		return fmt.Errorf("notify: %w", err)
	}
	return nil
}

type staticTokenCredentials struct {
	token string
}

func (c staticTokenCredentials) GetRequestMetadata(context.Context, ...string) (map[string]string, error) {
	return map[string]string{
		deviceAPITokenKey: c.token,
	}, nil
}

func (staticTokenCredentials) RequireTransportSecurity() bool {
	return true
}
