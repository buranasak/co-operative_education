package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Product struct {
	ID           primitive.ObjectID `json:"id" bson:"_id"`
	ProductName  string             `json:"productName" bson:"productName"`
	Category     string             `json:"category" bson:"category"`
	ValueTHB     int                `json:"valueTHB" bson:"valueTHB"`
	ValueUSD     int                `json:"valueUSD" bson:"valueUSD"`
	BusinessSize string             `json:"businessSize" bson:"businessSize"`
	CreatedAt    *time.Time         `json:"createdAt" bson:"createdAt"`
	UpdatedAt    *time.Time         `json:"updatedAt" bson:"updatedAt"`
	//deletedAt
}

type Export struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ProductId primitive.ObjectID `json:"productId" bson:"productId"`
	Country   string             `json:"country" bson:"country"`
	Month     int                `json:"month" bson:"month"`
	Year      int                `json:"year" bson:"year"`
	CreatedAt *time.Time         `json:"createdAt" bson:"createdAt"`
	UpdatedAt *time.Time         `json:"updatedAt" bson:"updatedAt"`
	//deletedAt
}

type ExportData struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	ProductName  string             `json:"productName" bson:"productName"`
	Category     string             `json:"category" bson:"category"`
	ValueTHB     int                `json:"valueTHB" bson:"valueTHB"`
	ValueUSD     int                `json:"valueUSD" bson:"valueUSD"`
	BusinessSize string             `json:"businessSize" bson:"businessSize"`
	Country      string             `json:"country" bson:"country"`
	Month        int                `json:"month" bson:"month"`
	Year         int                `json:"year" bson:"year"`
	CreatedAt    *time.Time         `json:"createdAt" bson:"createdAt"`
	UpdatedAt    *time.Time         `json:"updatedAt" bson:"updatedAt"`
}

type ExportWithProduct struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Product   Product            `json:"product" bson:"product"`
	Country   string             `json:"country" bson:"country"`
	Month     int                `json:"month" bson:"month"`
	Year      int                `json:"year" bson:"year"`
	CreatedAt *time.Time         `json:"createdAt" bson:"createdAt"`
	UpdatedAt *time.Time         `json:"updatedAt" bson:"updatedAt"`
	//deletedAt
}

// explore
type ExploreRequest struct {
	Columns   []*ExploreColumn    `json:"columns,omitempty"`
	Aggregate []*ExploreAggregate `json:"aggregate,omitempty"`
	Filter    *ExploreFilter      `json:"filter,omitempty"`
	Sorts     []*ExploreSort      `json:"sorts,omitempty"`
	Offset    *int                `json:"offset,omitempty"`
	Limit     *int                `json:"limit,omitempty"`
}

type ExploreColumn struct {
	Name  string `json:"name,omitempty"`
	Alias string `json:"alias,omitempty"`
}

// aggregate
type ExploreAggregate struct {
	Column    string `json:"column,omitempty"`
	Aggregate string `json:"aggregate,omitempty"`
	Alias     string `json:"alias,omitempty"`
}

// filter and group
type ExploreFilter struct {
	Operator  string        `json:"op,omitempty"`
	Arguments []interface{} `json:"args,omitempty"`
}

// sort
type ExploreSort struct {
	Column    string `json:"column,omitempty" bson:"column,omitempty"`
	Direction string `json:"direction,omitempty" bson:"direction,omitempty"`
}

// results response
type Explores struct {
	NumberMatched  *int          `json:"numberMatched,omitempty"`
	NumberReturned *int          `json:"numberReturned,omitempty"`
	Results        []interface{} `json:"results"`
}

// Exception model or error response
type Exception struct {
	Code   string `json:"code,omitempty"`
	Type   string `json:"type,omitempty"`
	Title  string `json:"title,omitempty"`
	Status *int   `json:"status,omitempty"`
	Detail string `json:"detail,omitempty"`
}
