package responses


type ErrorResponse struct {
	Message string `json:"message"`
}

type SuccessResponse struct {
	Message string `json:"message,omitempty"`
	Collection interface{} `json:"collection,omitempty"`
}

type SuccessFeatureResponse struct{
	Message string `json:"message,omitempty"`
	Feature interface{} `json:"feature,omitempty"`
}