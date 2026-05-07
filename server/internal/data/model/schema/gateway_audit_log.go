package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type GatewayAuditLog struct {
	ent.Schema
}

func (GatewayAuditLog) Fields() []ent.Field {
	return []ent.Field{
		field.Int("actor_id").
			Default(0),
		field.String("actor_name").
			Default("").
			MaxLen(128),
		field.String("actor_role").
			Default("").
			MaxLen(32),
		field.String("action").
			NotEmpty().
			MaxLen(128),
		field.String("target_type").
			Default("").
			MaxLen(64),
		field.String("target_id").
			Default("").
			MaxLen(128),
		field.JSON("metadata", map[string]any{}).
			Optional(),
		field.Time("created_at").
			Default(time.Now).
			Immutable(),
	}
}

func (GatewayAuditLog) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("created_at"),
		index.Fields("actor_id", "created_at"),
		index.Fields("action", "created_at"),
		index.Fields("target_type", "target_id"),
	}
}
