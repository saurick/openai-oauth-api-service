package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GatewayUsageLog struct {
	ent.Schema
}

func (GatewayUsageLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("api_key_id").
			Optional().
			Nillable(),
		field.String("api_key_prefix").
			Default("").
			MaxLen(16),
		field.String("session_id").
			Default("").
			MaxLen(128),
		field.String("request_id").
			Default("").
			MaxLen(128),
		field.String("method").
			NotEmpty().
			MaxLen(16),
		field.String("path").
			NotEmpty().
			MaxLen(256),
		field.String("endpoint").
			Default("").
			MaxLen(64),
		field.String("model").
			Default("").
			MaxLen(128),
		field.Int("status_code").
			Default(0),
		field.Bool("success").
			Default(false),
		field.Bool("stream").
			Default(false),
		field.Int64("input_tokens").
			Default(0),
		field.Int64("output_tokens").
			Default(0),
		field.Int64("total_tokens").
			Default(0),
		field.Int64("cached_tokens").
			Default(0),
		field.Int64("reasoning_tokens").
			Default(0),
		field.Int64("request_bytes").
			Default(0),
		field.Int64("response_bytes").
			Default(0),
		field.Int64("duration_ms").
			Default(0),
		field.String("upstream_configured_mode").
			Default("").
			MaxLen(32),
		field.String("upstream_mode").
			Default("").
			MaxLen(32),
		field.Bool("upstream_fallback").
			Default(false),
		field.String("upstream_error_type").
			Default("").
			MaxLen(128),
		field.String("error_type").
			Default("").
			MaxLen(128),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (GatewayUsageLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
		index.Fields("api_key_id", "created_at"),
		index.Fields("session_id", "created_at"),
		index.Fields("model", "created_at"),
		index.Fields("endpoint", "created_at"),
		index.Fields("upstream_mode", "created_at"),
		index.Fields("upstream_fallback", "created_at"),
		index.Fields("success", "created_at"),
	}
}
