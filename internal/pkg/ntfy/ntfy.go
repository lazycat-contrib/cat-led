package ntfy

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"text/template"
	"time"

	gotfy "github.com/AnthonyHewins/gotfy"
)

const (
	titleLedOn  = "设备开灯通知"
	titleLedOff = "设备关灯通知"
)

type LEDContext struct {
	Time   time.Time
	Status bool
	Name   string
}

type Config struct {
	Enabled     bool   `json:"enabled"`
	ServerURL   string `json:"server_url"`
	Topic       string `json:"topic"`
	Token       string `json:"token"`
	OnTemplate  string `json:"on_template"`
	OffTemplate string `json:"off_template"`
}

func DefaultConfig() *Config {
	return &Config{
		Enabled:     false,
		ServerURL:   "https://ntfy.sh",
		OnTemplate:  "{{.Name}} 任务执行成功，灯已开启",
		OffTemplate: "{{.Name}} 任务执行成功，灯已关闭",
	}
}

type Pusher struct {
	config   Config
	tplOpen  *template.Template
	tplClose *template.Template
}

func NewPusher(cfg Config) *Pusher {
	return &Pusher{
		config:   cfg,
		tplOpen:  template.Must(template.New("open").Parse(cfg.OnTemplate)),
		tplClose: template.Must(template.New("close").Parse(cfg.OffTemplate)),
	}
}

func (p *Pusher) PushLedOpenNotify(ctx LEDContext) error {
	return p.send(p.tplOpen, titleLedOn, ctx)
}

func (p *Pusher) PushLedCloseNotify(ctx LEDContext) error {
	return p.send(p.tplClose, titleLedOff, ctx)
}

func (p *Pusher) send(tpl *template.Template, title string, ctx LEDContext) error {
	content, err := executeTemplate(tpl, ctx)
	if err != nil {
		return fmt.Errorf("execute template: %w", err)
	}
	return p.publish(title, content)
}

func (p *Pusher) publish(title, message string) error {
	serverURL, err := url.Parse(p.config.ServerURL)
	if err != nil {
		return fmt.Errorf("parse server URL: %w", err)
	}

	pub, err := gotfy.NewPublisher(serverURL)
	if err != nil {
		return fmt.Errorf("create publisher: %w", err)
	}

	if p.config.Token != "" {
		pub.Headers.Set("Authorization", "Bearer "+p.config.Token)
	}

	_, err = pub.SendMessage(context.Background(), &gotfy.Message{
		Topic:    p.config.Topic,
		Message:  message,
		Title:    title,
		Tags:     []string{"bulb"},
		Priority: gotfy.Default,
	})
	return err
}

func executeTemplate(tpl *template.Template, data any) (string, error) {
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func TestConnection(cfg Config) error {
	p := NewPusher(cfg)
	return p.publish("懒猫小灯测试", "ntfy 通知通道已连接")
}
