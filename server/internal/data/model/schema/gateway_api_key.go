package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GatewayAPIKey struct {
	ent.Schema
}

func (GatewayAPIKey) Fields() []ent.Field {
	return []ent.Field{
		field.Int("owner_user_id").
			Optional().
			Nillable(),
		field.String("name").
			NotEmpty().
			MaxLen(80),
		field.String("key_hash").
			NotEmpty().
			Sensitive().
			MaxLen(64),
		field.String("plain_key").
			Default("").
			Sensitive().
			MaxLen(128),
		field.String("key_prefix").
			NotEmpty().
			MaxLen(16),
		field.String("key_last4").
			NotEmpty().
			MaxLen(8),
		field.Bool("disabled").
			Default(false),
		field.Int64("quota_requests").
			Default(0),
		field.Int64("quota_total_tokens").
			Default(0),
		field.Int64("quota_daily_tokens").
			Default(0),
		field.Int64("quota_weekly_tokens").
			Default(0),
		field.Int64("quota_daily_input_tokens").
			Default(0),
		field.Int64("quota_weekly_input_tokens").
			Default(0),
		field.Int64("quota_daily_output_tokens").
			Default(0),
		field.Int64("quota_weekly_output_tokens").
			Default(0),
		field.Int64("quota_daily_billable_input_tokens").
			Default(0),
		field.Int64("quota_weekly_billable_input_tokens").
			Default(0),
		field.JSON("allowed_models", []string{}).
			Optional(),
		field.Time("last_used_at").
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

func (GatewayAPIKey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("owner_user_id"),
		index.Fields("key_hash").Unique(),
		index.Fields("disabled"),
	}
}
