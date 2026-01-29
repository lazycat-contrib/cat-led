package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserPreference holds the schema definition for the UserPreference entity.
type UserPreference struct {
	ent.Schema
}

// Fields of the UserPreference.
func (UserPreference) Fields() []ent.Field {
	return []ent.Field{
		field.String("user_id").
			Immutable().
			Comment("用户ID"),
		field.String("bulb_style").
			Default("classic").
			Comment("灯泡样式: classic(经典灯泡), lava(熔岩灯), vintage(老式台灯), liquid(液态光效), lightbulb(灯泡开关), analog(点阵时钟)"),
		field.Time("created_at").
			Immutable().
			Comment("创建时间"),
		field.Time("updated_at").
			Comment("更新时间"),
	}
}

// Edges of the UserPreference.
func (UserPreference) Edges() []ent.Edge {
	return nil
}

// Indexes of the UserPreference.
func (UserPreference) Indexes() []ent.Index {
	return []ent.Index{
		// 确保每个用户只有一条偏好设置记录
		index.Fields("user_id").Unique(),
	}
}
