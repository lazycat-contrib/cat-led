package serverchan

import (
	"bytes"
	"cat-led/internal/ent"
	"cat-led/internal/pkg/zlog"
	"io"
	"net/http"
	"text/template"
	"time"

	svrchan "github.com/easychen/serverchan-sdk-golang"
)

var (
	buf = make([]byte, 0, 512)
)

type LEDContext struct {
	Time   time.Time
	Status bool
	Name   string
}

type Pusher struct {
	enabled      bool
	emailEnabled bool
	emailURL     string
	sendKey      string
	tplOpen      *template.Template
	tplClose     *template.Template
	logger       *zlog.Logger
}

func NewPusher(config *ent.ServerChanConfig, logger *zlog.Logger) *Pusher {
	return &Pusher{
		enabled:      config.Enabled,
		sendKey:      config.SendKey,
		emailEnabled: config.EmailEnabled,
		emailURL:     config.EmailURL,
		tplOpen:      template.Must(template.New("open").Parse(config.OnTemplate)),
		tplClose:     template.Must(template.New("close").Parse(config.OffTemplate)),
		logger:       logger,
	}
}

func (p *Pusher) PushLedOpenNotify(ctx LEDContext) {
	notify, err := executeTpl(p.tplOpen, ctx)
	if err != nil {
		p.logger.Error().Err(err).Msg("执行开灯通知模板失败")
		return
	}

	p.sendNotifications("设备开灯通知", notify)
}

func (p *Pusher) PushLedCloseNotify(ctx LEDContext) {
	notify, err := executeTpl(p.tplClose, ctx)
	if err != nil {
		p.logger.Error().Err(err).Msg("执行关灯通知模板失败")
		return
	}
	p.sendNotifications("设备关灯通知", notify)
}

func executeTpl(tpl *template.Template, data any) (string, error) {
	buffer := bytes.NewBuffer(buf)
	defer buffer.Reset()
	err := tpl.Execute(buffer, data)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func (p *Pusher) sendNotifications(title string, body string) {
	if p.enabled && p.sendKey != "" {
		p.sendServerchainNotify(title, body)
	}
	if p.emailEnabled && p.emailURL != "" {
		p.sendEmailNotify(title, body)
	}
}

func (p *Pusher) sendServerchainNotify(title string, body string) {
	_, err := svrchan.ScSend(p.sendKey, title, body, nil)
	if err != nil {
		p.logger.Error().Err(err).Msg("发送Server酱通知失败")
		return
	}
	p.logger.Info().Msg("发送Server酱通知成功")
}

func (p *Pusher) sendEmailNotify(_ string, body string) {
	resp, err := http.Post(p.emailURL, "text/plain", bytes.NewBufferString(body))
	if err != nil {
		p.logger.Error().Err(err).Msg("发送邮件通知失败")
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	text := string(respBody)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		p.logger.Error().
			Int("status_code", resp.StatusCode).
			Str("Response", text).
			Msg("发送邮件通知失败")
		return
	}
	p.logger.Info().Msg("发送邮件通知成功")
}
