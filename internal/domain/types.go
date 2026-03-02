package domain

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// --- Shared Custom Types ---

// JSONB is a helper for handling JSONB columns in Postgres as a map.
type JSONB map[string]interface{}

func (j JSONB) Value() (driver.Value, error) {
	return json.Marshal(j)
}

func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, j)
}

// RawJSON is a helper for handling raw JSON bytes (like json.RawMessage)
// It is critical for ContentBlocks where the structure is dynamic.
type RawJSON json.RawMessage

func (j RawJSON) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

func (j *RawJSON) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	*j = append((*j)[0:0], bytes...)
	return nil
}

// MarshalJSON returns j as the JSON encoding of j.
// Required because 'type RawJSON json.RawMessage' strips the underlying MarshalJSON method.
func (j RawJSON) MarshalJSON() ([]byte, error) {
	if j == nil {
		return []byte("null"), nil
	}
	return j, nil
}

// UnmarshalJSON sets *j to a copy of data.
func (j *RawJSON) UnmarshalJSON(data []byte) error {
	if j == nil {
		return errors.New("RawJSON: UnmarshalJSON on nil pointer")
	}
	*j = append((*j)[0:0], data...)
	return nil
}

// Pagination
type Pagination struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalItems int64 `json:"totalItems"`
	TotalPages int   `json:"totalPages"`
}

// Response standardizes API responses.
type Response struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}
