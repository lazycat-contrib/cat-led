package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// ServerChanConfig holds the schema definition for the ServerChanConfig entity.
type ServerChanConfig struct {
	ent.Schema
}

// Fields of the ServerChanConfig.
func (ServerChanConfig) Fields() []ent.Field {
	return []ent.Field{
		field.String("send_key").
			Comment("Server酱的SendKey"),
		field.String("on_template").
			Comment("开灯通知模板").
			Default("{{.Name}} 任务执行成功，灯已开启"),
		field.String("off_template").
			Comment("关灯通知模板").
			Default("{{.Name}} 任务执行成功，灯已关闭"),
		field.Bool("enabled").
			Comment("是否启用Server酱通知").
			Default(false),
	}
}

// Edges of the ServerChanConfig.
func (ServerChanConfig) Edges() []ent.Edge {
	return nil
}
