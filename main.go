package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/dimcz/tgpollbot/config"
	"github.com/dimcz/tgpollbot/lib/validator"
	"github.com/dimcz/tgpollbot/service"
	"github.com/dimcz/tgpollbot/storage"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

const PREFIX = "1.0.0-"

var VERSION string

func main() {
	logrus.Info("Start TGPollBoot ", PREFIX, VERSION)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli, err := storage.Connect(ctx, config.Config.RedisDB)
	if err != nil {
		logrus.Fatal("could not connect to storage with error: ", err)

		return
	}

	defer cli.Close()

	tg, err := service.NewTGService(ctx, cli)
	if err != nil {
		logrus.Error("could not start TGService with error: ", err)

		return
	}
	defer tg.Close()

	tg.Run()

	srv := service.NewWebService(cli)

	if err := run(srv); err != nil {
		logrus.Error("failed to run web web service with error: ", err)
	}
}

func run(srv *service.WebService) error {
	e := echo.New()
	e.Validator = validator.NewValidator()

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	e.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup: "header:X-Api-Key",
		Validator: func(key string, ctx echo.Context) (bool, error) {
			return key == config.Config.XApiKey, nil
		},
	}))

	e.GET("/v1/:request_id", srv.Get)
	e.POST("/v1/", srv.Post)

	conn := fmt.Sprintf(":%d", config.Config.Port)

	go func() {
		if err := e.Start(conn); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	return e.Shutdown(ctx)
}
