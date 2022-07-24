package main

import (
	"context"
	"os"
	"time"

	"github.com/dimcz/tgpollbot/config"
	"github.com/dimcz/tgpollbot/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := storage.Connect(ctx, config.Config.RedisDB)
	if err != nil {
		logrus.Error(err)

		return
	}

	recordID := uuid.New().String()

	r := storage.Record{
		ID: recordID,
		Task: storage.Task{
			Message: "Poll 1",
			Buttons: []string{
				"Option 1",
				"Option 2",
				"Option 3",
			},
		},
	}

	if err := db.Insert(r); err != nil {
		logrus.Error(err)

		return
	}

	r.Task.Message = "Poll 2"
	if err := db.Insert(r); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	r.Task.Message = "Poll 3"
	if err := db.Insert(r); err != nil {
		logrus.Error(err)
		os.Exit(1)
	}

	count := 1
	for {
		r, err = db.Next()
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		logrus.Info(r)

		count += 1

		if count%4 == 0 {
			if err := db.Drop(r); err != nil {
				logrus.Error(err)
				os.Exit(1)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}
