package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Collection struct {
	ID        primitive.ObjectID `json:"id" bson:"_id"`
	Name      string             `json:"name" bson:"name" validate:"required"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt *time.Time         `json:"updated_at,omitempty" bson:"updated_at,omitempty"`
	DeletedAt *time.Time         `json:"deleted_at,omitempty" bson:"deleted_at,omitempty"`
}