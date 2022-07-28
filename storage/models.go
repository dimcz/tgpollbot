package storage

import (
	"encoding/json"
	"time"
)

const RecordTTL = 7 * 24 * 60 * 60 * time.Second

const (
	PollRequestsSet = "pollRequestsSet"
	RecordsList     = "requestsList"
	RecordPrefix    = "record:"
	SessionSet      = "sessionSet"
)

const (
	RecordProcessing = "processing"
	RecordPollDone   = "done"
)

type Task struct {
	Message string   `json:"message" validate:"required,max=4096"`
	Buttons []string `json:"buttons" validate:"required,checkOption"`
}

type Request struct {
	Status string `json:"status"`
	Option *int   `json:"option"`
	Task   `json:"task"`
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

func (r *Request) DTO() DTO {
	d := DTO{
		Status: r.Status,
		Option: r.Option,
	}

	if r.Option != nil {
		d.Text = r.Buttons[*r.Option]
	}

	return d
}

/*
Poll #75909
-----------------------------------------
*/
