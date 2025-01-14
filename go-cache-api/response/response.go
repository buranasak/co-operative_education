package response

import "go-cache-api/models"

type ProductsCacheResponse struct {
	Message  string           `json:"message"`
	Count    int64            `json:"count,omitempty"`
	Products []models.Product `json:"products"`
}

type ProductCacheResponse struct {
	TotalProduct int            `json:"totalProduct"`
	Products     models.Product `json:"products"`
}
