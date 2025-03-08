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
		field.Enum("operation").Values("on", "off").Default("on"),
		field.Bool("enabled").Default(true),
		field.Bool("allow_edit_by_others"),
	}
}

// Edges of the Schedule.
func (Schedule) Edges() []ent.Edge {
	return nil
}
