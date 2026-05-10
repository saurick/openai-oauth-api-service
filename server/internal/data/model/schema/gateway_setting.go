package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GatewaySetting struct {
	ent.Schema
}

func (GatewaySetting) Fields() []ent.Field {
	return []ent.Field{
		field.String("key").
			NotEmpty().
			MaxLen(128),
		field.String("value").
			Default("").
			MaxLen(512),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (GatewaySetting) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("key").Unique(),
	}
}
