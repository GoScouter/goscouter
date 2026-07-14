package records

import (
    "net/http"
)

type HTTPRecords struct {
	RequestURL string
	FinalURL   string
	StatusCode int
	Status     string
	Proto      string
	Headers    http.Header
}
