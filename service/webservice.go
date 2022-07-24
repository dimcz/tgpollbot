package service

import (
	"net/http"

	"github.com/dimcz/tgpollbot/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

type WebService struct {
	cli *storage.Client
}

type JSON map[string]string

func (srv *WebService) Post(ctx echo.Context) error {
	var task storage.Task
	if err := ctx.Bind(&task); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.Wrap(err, "could not decode user data"))
	}

	if err := ctx.Validate(&task); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, err)
	}

	id := uuid.New().String()
	r := storage.Request{
		ID:   id,
		Task: task,
	}

	if err := srv.cli.RPush(storage.RecordsList, r); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	err := srv.cli.Set(storage.RecordPrefix+id,
		storage.DTO{
			Status: storage.RecordProcessing,
			Option: nil,
		})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, err)
	}

	return ctx.JSON(http.StatusCreated, JSON{
		"request_id": id,
	})
}

func (srv *WebService) Get(ctx echo.Context) error {
	id := ctx.Param("request_id")

	dto := storage.DTO{}
	if err := srv.cli.Get(storage.RecordPrefix+id, &dto); err == nil {
		return ctx.JSON(http.StatusOK, dto)
	}

	return echo.NewHTTPError(http.StatusNotFound, "request not found")
}

func NewWebService(cli *storage.Client) *WebService {
	return &WebService{cli}
}
