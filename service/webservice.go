package service

import (
	"net/http"

	"github.com/dimcz/tgpollbot/storage"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type WebService struct {
	cache *Cache
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
		Status: storage.RecordProcessing,
		Task:   task,
	}

	if err := srv.cache.InitRequest(ctx.Request().Context(), id, r); err != nil {
		logrus.Error("could not set record to cache with error: ", err)

		return echo.NewHTTPError(http.StatusInternalServerError, errors.Wrap(err, "failed to set new request"))
	}

	return ctx.JSON(http.StatusCreated, map[string]string{
		"request_id": id,
	})
}

func (srv *WebService) Get(ctx echo.Context) error {
	id := ctx.Param("request_id")

	r, err := srv.cache.Get(ctx.Request().Context(), id)
	if err != nil {
		if err == redis.Nil {
			return echo.NewHTTPError(http.StatusNotFound, "request not found")
		}

		logrus.Error("failed to get request from cache with error: ", err)

		return echo.NewHTTPError(http.StatusInternalServerError, errors.Wrap(err, "failed to get request"))
	}

	return ctx.JSON(http.StatusOK, r.DTO())
}

func NewWebService(cache *Cache) *WebService {
	return &WebService{cache}
}
