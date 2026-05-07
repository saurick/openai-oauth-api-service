package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GatewayModelPrice struct {
	ent.Schema
}

func (GatewayModelPrice) Fields() []ent.Field {
	return []ent.Field{
		field.String("model_id").
			NotEmpty().
			MaxLen(128),
		field.Float("input_usd_per_million").
			Default(0),
		field.Float("cached_input_usd_per_million").
			Default(0),
		field.Float("output_usd_per_million").
			Default(0),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (GatewayModelPrice) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("model_id").Unique(),
	}
}
