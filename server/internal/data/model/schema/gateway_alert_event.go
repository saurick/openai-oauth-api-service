package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GatewayAlertEvent struct {
	ent.Schema
}

func (GatewayAlertEvent) Fields() []ent.Field {
	return []ent.Field{
		field.Int("rule_id").
			Optional().
			Nillable(),
		field.String("rule_name").
			Default("").
			MaxLen(128),
		field.String("metric").
			NotEmpty().
			MaxLen(64),
		field.Float("value").
			Default(0),
		field.Float("threshold").
			Default(0),
		field.String("status").
			Default("open").
			MaxLen(32),
		field.Int("ack_by").
			Optional().
			Nillable(),
		field.Time("ack_at").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (GatewayAlertEvent) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
		index.Fields("status", "created_at"),
		index.Fields("rule_id", "created_at"),
	}
}
