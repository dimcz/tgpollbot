package storage

import (
	"encoding/json"
)

const (
	RecordProcessing = "processing"
	RecordPollDone   = "done"
)

type Task struct {
	Message string   `json:"message" validate:"required"`
	Buttons []string `json:"buttons" validate:"required"`
}

type Request struct {
	ID   string `json:"id"`
	Task `json:"task"`
}

func (r Request) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

func (r *Request) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

type DTO struct {
	Status string `json:"status"`
	Option *int   `json:"option,omitempty"`
	Text   string `json:"text,omitempty"`
}

func (d DTO) MarshalBinary() ([]byte, error) {
	return json.Marshal(d)
}

func (d *DTO) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, d)
}
