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
	MessageID int    `json:"message_id" bson:"message_id"`
	ChatID    int64  `json:"chat_id" bson:"chat_id"`
	PollID    string `json:"poll_id" bson:"poll_id"`
}

type Record struct {
	Status    string `json:"status" bson:"status"`
	UpdatedAt int64  `json:"updated_at" bson:"updated_at"`
	Option    int    `json:"option" bson:"option"`
	Text      string `json:"text" bson:"text"`

	ID   string `json:"id" bson:"id"`
	Task Task   `json:"task" bson:"task"`
	Poll []Poll `json:"poll" bson:"poll"`
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

type Opts map[string]interface{}
