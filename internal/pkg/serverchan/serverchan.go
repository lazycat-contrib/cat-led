package serverchan

import (
	"bytes"
	"text/template"
	"time"

	svrchan "github.com/easychen/serverchan-sdk-golang"
)

// Notification titles for LED operations.
const (
	titleLedOn  = "设备开灯通知"
	titleLedOff = "设备关灯通知"
)

// LEDContext contains the data passed to notification templates.
type LEDContext struct {
	Time   time.Time
	Status bool
	Name   string
}

// Pusher handles sending notifications via ServerChan.
type Pusher struct {
	sendKey  string
	tplOpen  *template.Template
	tplClose *template.Template
}

// NewPusher creates a new Pusher with the given templates.
func NewPusher(sendKey, tplOpen, tplClose string) *Pusher {
	return &Pusher{
		sendKey:  sendKey,
		tplOpen:  template.Must(template.New("open").Parse(tplOpen)),
		tplClose: template.Must(template.New("close").Parse(tplClose)),
	}
}

// PushLedOpenNotify sends a notification when LED is turned on.
func (p *Pusher) PushLedOpenNotify(ctx LEDContext) error {
	return p.sendNotification(p.tplOpen, titleLedOn, ctx)
}

// PushLedCloseNotify sends a notification when LED is turned off.
func (p *Pusher) PushLedCloseNotify(ctx LEDContext) error {
	return p.sendNotification(p.tplClose, titleLedOff, ctx)
}

// sendNotification executes the template and sends the notification.
func (p *Pusher) sendNotification(tpl *template.Template, title string, ctx LEDContext) error {
	content, err := executeTemplate(tpl, ctx)
	if err != nil {
		return err
	}
	_, err = svrchan.ScSend(p.sendKey, title, content, nil)
	return err
}

// executeTemplate renders a template with the given data.
func executeTemplate(tpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
