package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GatewayAlertRule struct {
	ent.Schema
}

func (GatewayAlertRule) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").
			NotEmpty().
			MaxLen(128),
		field.String("metric").
			NotEmpty().
			MaxLen(64),
		field.String("operator").
			Default(">=").
			MaxLen(8),
		field.Float("threshold").
			Default(0),
		field.Int64("window_seconds").
			Default(300),
		field.Bool("enabled").
			Default(true),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (GatewayAlertRule) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("enabled"),
		index.Fields("metric"),
	}
}
