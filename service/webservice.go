package service

import (
	"net/http"
	"time"

	"github.com/dimcz/tgpollbot/lib/e"
	"github.com/dimcz/tgpollbot/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

type WebService struct {
	storage storage.Storage
}

func (srv *WebService) Post(ctx echo.Context) error {
	var task storage.Task
	if err := ctx.Bind(&task); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.Wrap(err, "could not decode user data"))
	}
	if err := ctx.Validate(&task); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err)
	}

	requestID := uuid.New().String()
	r := storage.Record{
		ID:        requestID,
		Status:    storage.PROCESS,
		UpdatedAt: time.Now().Unix(),
		Task:      task,
	}

	if err := srv.storage.Set(requestID, r); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return ctx.JSON(http.StatusCreated, map[string]string{"request_id": requestID})
}

func (srv *WebService) Get(ctx echo.Context) error {
	id := ctx.Param("id")
	r, err := srv.storage.Get(id)
	if err != nil {
		return e.HTTPError(err)
	}

	var code int
	switch r.Status {
	case storage.DONE:
		code = http.StatusOK
	case storage.PROCESS:
		code = http.StatusCreated
	default:
		return e.HTTPError(e.NewInternal("unexpected status"))
	}

	return ctx.JSON(code, r.DTO())
}

func NewWebService(s storage.Storage) *WebService {
	return &WebService{s}
}
