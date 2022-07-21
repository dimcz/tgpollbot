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
	MessageID int
	ChaID     int64
	PollID    string
}

type Record struct {
	Status    string `json:"status"`
	UpdatedAt int64  `json:"-"`
	Option    int    `json:"option,omitempty"`
	Text      string `json:"text,omitempty"`

	ID   string `json:"-"`
	Task Task   `json:"-"`
	Poll []Poll `json:"-"`
}
