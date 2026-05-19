package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GatewayContextSummary struct {
	ent.Schema
}

func (GatewayContextSummary) Fields() []ent.Field {
	return []ent.Field{
		field.String("session_id").
			NotEmpty().
			Unique().
			MaxLen(128),
		field.Int("api_key_id").
			Optional().
			Nillable(),
		field.String("api_key_prefix").
			Default("").
			MaxLen(16),
		field.Text("summary").
			Default(""),
		field.Int64("summary_tokens").
			Default(0),
		field.Int("compaction_count").
			Default(0),
		field.String("last_request_id").
			Default("").
			MaxLen(128),
		field.String("last_reason").
			Default("").
			MaxLen(64),
		field.Int64("last_original_bytes").
			Default(0),
		field.Int64("last_compacted_bytes").
			Default(0),
		field.Int64("last_original_tokens").
			Default(0),
		field.Int64("last_compacted_tokens").
			Default(0),
		field.String("last_error").
			Default("").
			MaxLen(256),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
		field.Time("updated_at").
			Default(time.Now).
			UpdateDefault(time.Now),
	}
}

func (GatewayContextSummary) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("api_key_id", "updated_at"),
		index.Fields("updated_at"),
	}
}
