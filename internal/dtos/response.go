package dtos

// APIResponse represents the standard API envelope response.
type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data,omitempty"`
}

// BaseResponse represents a response without data payload (e.g. errors or simple acknowledgments).
type BaseResponse struct {
	Success bool   `json:"success"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
