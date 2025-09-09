package serverchan

import (
	"bytes"
	svrchan "github.com/easychen/serverchan-sdk-golang"
	"text/template"
	"time"
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
	sendKey  string
	tplOpen  *template.Template
	tplClose *template.Template
}

func NewPusher(sendKey string, tplOpen string, tplClose string) *Pusher {
	return &Pusher{
		sendKey:  sendKey,
		tplOpen:  template.Must(template.New("open").Parse(tplOpen)),
		tplClose: template.Must(template.New("close").Parse(tplClose))}
}

func (p *Pusher) PushLedOpenNotify(ctx LEDContext) error {
	notify, err := executeTpl(p.tplOpen, ctx)
	if err != nil {
		return err
	}
	_, err = svrchan.ScSend(p.sendKey, "设备开灯通知", notify, nil)
	if err != nil {
		return err
	}
	return nil
}

func (p *Pusher) PushLedCloseNotify(ctx LEDContext) error {
	notify, err := executeTpl(p.tplClose, ctx)
	if err != nil {
		return err
	}
	_, err = svrchan.ScSend(p.sendKey, "设备关灯通知", notify, nil)
	if err != nil {
		return err
	}
	return nil
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
