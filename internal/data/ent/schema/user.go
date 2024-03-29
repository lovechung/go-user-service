package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema"
	"entgo.io/ent/schema/field"
)

// User holds the schema definition for the User entity.
type User struct {
	ent.Schema
}

func (User) Annotations() []schema.Annotation {
	return []schema.Annotation{
		entsql.Annotation{Table: "user"},
	}
}

// Fields of the User.
func (User) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("username").
			Optional(),
		field.String("password").
			Optional(),
		field.Time("created_at").
			Optional(),
		//Default(time.Now().Local).
		//SchemaType(map[string]string{dialect.MySQL: "datetime"}),
		field.Time("updated_at").
			Optional(),
		//Default(time.Now().Local).
		//SchemaType(map[string]string{dialect.MySQL: "datetime"}),
	}
}

// Edges of the User.
func (User) Edges() []ent.Edge {
	return nil
}
