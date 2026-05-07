package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GatewayModel struct {
	ent.Schema
}

func (GatewayModel) Fields() []ent.Field {
	return []ent.Field{
		field.String("model_id").
			NotEmpty().
			MaxLen(128),
		field.String("owned_by").
			Default("").
			MaxLen(128),
		field.Int64("created_unix").
			Default(0),
		field.Bool("enabled").
			Default(true),
		field.String("source").
			Default("manual").
			MaxLen(32),
		field.Time("last_seen_at").
			Optional().
			Nillable(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (GatewayModel) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("model_id").Unique(),
		index.Fields("enabled"),
		index.Fields("source"),
	}
}
