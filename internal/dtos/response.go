package dtos

import "time"

// APIResponse represents the standard API envelope response.
type APIResponse[T any] struct {
	Status    string    `json:"status"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Data      T         `json:"data,omitempty"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}

// BaseResponse represents a response without data payload (e.g. errors or simple acknowledgments).
type BaseResponse struct {
	Status    string    `json:"status"`
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp,omitempty"`
}
