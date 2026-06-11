package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// NtfyConfig holds the schema definition for the NtfyConfig entity.
type NtfyConfig struct {
	ent.Schema
}

// Fields of the NtfyConfig.
func (NtfyConfig) Fields() []ent.Field {
	return []ent.Field{
		field.Bool("enabled").
			Comment("是否启用ntfy通知").
			Default(false),
		field.String("server_url").
			Comment("ntfy服务器地址").
			Default("https://ntfy.sh"),
		field.String("topic").
			Comment("ntfy主题/topic").
			Default(""),
		field.String("token").
			Comment("ntfy访问令牌(可选)").
			Default(""),
		field.String("on_template").
			Comment("开灯通知模板").
			Default("{{.Name}} 任务执行成功，灯已开启"),
		field.String("off_template").
			Comment("关灯通知模板").
			Default("{{.Name}} 任务执行成功，灯已关闭"),
	}
}

// Edges of the NtfyConfig.
func (NtfyConfig) Edges() []ent.Edge {
	return nil
}
