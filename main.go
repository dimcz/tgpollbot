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
	"github.com/dimcz/tgpollbot/storage/badger"
	"github.com/dimcz/tgpollbot/storage/redis"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"
)

var VERSION string

func main() {
	logrus.Info("Start TGPollBoot ", VERSION)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var (
		db  storage.Storage
		err error
	)

	if len(config.Config.RedisDB) == 0 {
		db, err = badger.Open("/tmp/database")
		if err != nil {
			logrus.Fatal(err)
		}

		defer db.Close()
	} else {
		db, err = redis.Connect(ctx)
		if err != nil {
			logrus.Fatal(err)
		}

		defer db.Close()
	}

	tg, err := service.NewTGService(ctx, db)
	if err != nil {
		logrus.Error(err)

		return
	}
	defer tg.Close()

	tg.Run()

	srv := service.NewWebService(db)

	if err := run(srv); err != nil {
		logrus.Error(err)
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

	e.GET("/v1/:id", srv.Get)
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
