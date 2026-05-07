// server/internal/data/model/schema/user.go
package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

type User struct {
	ent.Schema
}

func (User) Fields() []ent.Field {
	return []ent.Field{
		field.String("username").
			NotEmpty().
			MaxLen(32),
		field.String("password_hash").
			NotEmpty().
			Sensitive(),
		field.String("oauth_provider").
			Optional().
			Nillable().
			MaxLen(32),
		field.String("oauth_subject").
			Optional().
			Nillable().
			MaxLen(255),
		field.String("oauth_email").
			Optional().
			Nillable().
			MaxLen(255),
		field.String("oauth_display_name").
			Optional().
			Nillable().
			MaxLen(128),
		field.Bool("disabled").
			Default(false),
		field.Time("last_login_at").
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

func (User) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("username").Unique(),
		index.Fields("oauth_provider", "oauth_subject").Unique(),
	}
}
