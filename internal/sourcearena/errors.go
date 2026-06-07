package sourcearena

import "fmt"

type APIError struct {
	StatusCode int
	Message    string
	Endpoint   string
}

func (e *APIError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("sourcearena %s: status %d: %s", e.Endpoint, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("sourcearena %s: %s", e.Endpoint, e.Message)
}

func NewAPIError(endpoint string, status int, message string) *APIError {
	return &APIError{Endpoint: endpoint, StatusCode: status, Message: message}
}
