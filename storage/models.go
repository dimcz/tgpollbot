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

type Record struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Option *int   `json:"option"`
	Text   string `json:"text"`

	Task Task `json:"task"`
}

func (r Record) MarshalBinary() ([]byte, error) {
	return json.Marshal(r)
}

func (r *Record) UnmarshalBinary(data []byte) error {
	return json.Unmarshal(data, r)
}

type RecordDTO struct {
	Status string `json:"status"`
	Option *int   `json:"option,omitempty"`
	Text   string `json:"text,omitempty"`
}

func (r *Record) DTO() RecordDTO {
	return RecordDTO{
		Status: r.Status,
		Option: r.Option,
		Text:   r.Text,
	}
}

type Opts map[string]interface{}
