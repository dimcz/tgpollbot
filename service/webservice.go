package service

import (
	"net/http"

	"github.com/dimcz/tgpollbot/lib/redis"
	"github.com/dimcz/tgpollbot/storage"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

type WebService struct {
	cli *redis.Client
}

func (srv *WebService) Post(ctx echo.Context) error {
	var task storage.Task
	if err := ctx.Bind(&task); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, errors.Wrap(err, "could not decode user data"))
	}

	if err := ctx.Validate(&task); err != nil {
		return echo.NewHTTPError(http.StatusUnprocessableEntity, errors.Wrap(err, "could not validate user data"))
	}

	id := uuid.New().String()
	r := storage.Request{
		ID:   id,
		Task: task,
	}

	if err := srv.cli.RPush(ctx.Request().Context(), storage.RecordsList, r); err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, errors.Wrap(err, "failed to push new request"))
	}

	err := srv.cli.Set(ctx.Request().Context(), storage.RecordPrefix+id,
		storage.DTO{
			Status: storage.RecordProcessing,
			Option: nil,
		})
	if err != nil {
		return echo.NewHTTPError(http.StatusInternalServerError, errors.Wrap(err, "failed to set new request"))
	}

	return ctx.JSON(http.StatusCreated, map[string]string{
		"request_id": id,
	})
}

func (srv *WebService) Get(ctx echo.Context) error {
	id := ctx.Param("request_id")

	dto := storage.DTO{}
	if err := srv.cli.Get(ctx.Request().Context(), storage.RecordPrefix+id, &dto); err == nil {
		return ctx.JSON(http.StatusOK, dto)
	}

	return echo.NewHTTPError(http.StatusNotFound, "request not found")
}

func NewWebService(cli *redis.Client) *WebService {
	return &WebService{cli}
}
