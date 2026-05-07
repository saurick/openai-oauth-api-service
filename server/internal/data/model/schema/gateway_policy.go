package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GatewayPolicy struct {
	ent.Schema
}

func (GatewayPolicy) Fields() []ent.Field {
	return []ent.Field{
		field.Int("api_key_id").
			Positive(),
		field.String("model_id").
			NotEmpty().
			MaxLen(128),
		field.Int64("rpm").
			Default(0),
		field.Int64("tpm").
			Default(0),
		field.Int64("daily_requests").
			Default(0),
		field.Int64("monthly_requests").
			Default(0),
		field.Int64("daily_tokens").
			Default(0),
		field.Int64("monthly_tokens").
			Default(0),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (GatewayPolicy) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("api_key_id", "model_id").Unique(),
		index.Fields("model_id"),
	}
}
