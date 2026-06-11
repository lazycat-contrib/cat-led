package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"github.com/google/uuid"
)

// Schedule holds the schema definition for the Schedule entity.
type Schedule struct {
	ent.Schema
}

// Fields of the Schedule.
func (Schedule) Fields() []ent.Field {
	return []ent.Field{
		field.UUID("id", uuid.UUID{}).Default(uuid.New).StorageKey("uuid"),
		field.String("name"),
		field.String("creator").Immutable(),
		field.Ints("week_days"),
		field.Int("hour"),
		field.Int("minute"),
		field.Enum("operation").Values("on", "off", "shutdown", "reboot").Default("on"),
		field.Bool("enabled").Default(true),
		field.Bool("allow_edit_by_others"),
		field.Bool("notify_via_server_chan").Default(false).Comment("是否通过Server酱通知"),
		field.Bool("notify_via_lzc").Default(false).Comment("是否通过懒猫内置通知"),
		field.Bool("notify_via_ntfy").Default(false).Comment("是否通过ntfy通知"),
	}
}

// Edges of the Schedule.
func (Schedule) Edges() []ent.Edge {
	return nil
}
