package model

import "go.mongodb.org/mongo-driver/bson/primitive"

// Exception model     //error response
type Exception struct {
	Code   string `json:"code,omitempty"`
	Type   string `json:"type,omitempty"`
	Title  string `json:"title,omitempty"`
	Status *int   `json:"status,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// Explore model 
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

//
type ExploreAggregate struct {
	Column    string `json:"column,omitempty"`
	Aggregate string `json:"aggregate,omitempty"`
	Alias     string `json:"alias,omitempty"`
}

//filter 
type ExploreFilter struct {
	Operator  string        `json:"op,omitempty"`
	Arguments []interface{} `json:"args,omitempty"`
}

//sort 
type ExploreSort struct {
	Column    string `json:"column,omitempty" bson:"column,omitempty"`
	Direction string `json:"direction,omitempty" bson:"direction,omitempty"`
}


// Results response 
type Explores struct {
	NumberMatched  *int          `json:"numberMatched,omitempty"`
	NumberReturned *int          `json:"numberReturned,omitempty"`
	Results        []interface{} `json:"results"`
}

// ServiceUsageResultExplore model
type ServiceUsageResultExplore struct {
	ServiceId primitive.ObjectID `json:"serviceId,omitempty"`
}
