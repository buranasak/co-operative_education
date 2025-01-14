package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Feature struct {
	Id         primitive.ObjectID     `json:"id" bson:"_id"`
	Type       string                 `json:"type" bson:"type"`
	Geometry   Geometry               `json:"geometry" bson:"geometry"`
	Properties map[string]interface{} `json:"properties" bson:"properties"`
	CreatedAt  time.Time              `json:"created_at,omitempty" bson:"created_at,omitempty"`
	UpdatedAt  *time.Time             `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
	DeletedAt  *time.Time             `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
}

type Geometry struct {
	Type        string        `json:"type" bson:"type" validate:"required"`
	Coordinates []interface{} `json:"coordinates" bson:"coordinates" validate:"required"`
}
