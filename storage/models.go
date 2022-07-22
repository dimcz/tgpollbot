package storage

const (
	PROCESS = "process"
	DONE    = "done"
)

type Task struct {
	Message string   `json:"message" validate:"required"`
	Buttons []string `json:"buttons" validate:"required"`
}

type Poll struct {
	MessageID int    `json:"message_id"`
	ChatID    int64  `json:"chat_id"`
	PollID    string `json:"poll_id"`
}

type Record struct {
	Status    string `json:"status"`
	UpdatedAt int64  `json:"updated_at"`
	Option    int    `json:"option,omitempty"`
	Text      string `json:"text,omitempty"`

	ID   string `json:"id"`
	Task Task   `json:"task"`
	Poll []Poll `json:"poll"`
}

type RecordDTO struct {
	Status string `json:"status"`
	Option int    `json:"option"`
	Text   string `json:"text"`
}

func (r *Record) DTO() RecordDTO {
	return RecordDTO{
		Status: r.Status,
		Option: r.Option,
		Text:   r.Text,
	}
}
