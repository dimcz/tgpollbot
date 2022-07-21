package service

import (
	"context"
	"reflect"
	"strings"
	"time"

	"github.com/dimcz/tgpollbot/config"
	"github.com/dimcz/tgpollbot/lib/e"
	"github.com/dimcz/tgpollbot/lib/utils"
	"github.com/dimcz/tgpollbot/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

const SendTimeout = 3 * time.Second

type TGService struct {
	ctx context.Context
	db  storage.Storage
	bot *tgbotapi.BotAPI

	liveChats *utils.Set[int64]
}

func (tg *TGService) Close() {
	tg.bot.StopReceivingUpdates()
}

func (tg *TGService) Run() {
	ch := tg.bot.GetUpdatesChan(tgbotapi.UpdateConfig{})
	go tg.updateService(ch)

	go tg.sendService()
}

func (tg *TGService) sendService() {
	next := time.Now().Add(SendTimeout)

	timer := time.NewTimer(time.Until(next))
	defer timer.Stop()

	for {
		select {
		case <-tg.ctx.Done():
			return
		case <-timer.C:
			if err := tg.send(); err != nil {
				logrus.Error(err)

				return
			}

			next = time.Now().Add(SendTimeout)
			timer.Reset(time.Until(next))
		}
	}
}

func (tg *TGService) send() error {
	if tg.liveChats.Len() == 0 {
		return nil
	}

	r, err := tg.getReadyRecord()
	if err != nil {
		if err == e.ErrNotFound {
			return nil
		}

		return err
	}

	for _, id := range tg.liveChats.Range() {
		if chatPollExists(r.Poll, id) {
			continue
		}

		poll := tgbotapi.NewPoll(id, r.Task.Message, r.Task.Buttons...)
		poll.IsAnonymous = false
		poll.AllowsMultipleAnswers = false

		msg, err := tg.bot.Send(poll)
		if err != nil {
			logrus.Error(err)

			tg.liveChats.UnSet(id)
		}

		r.Poll = append(r.Poll, storage.Poll{
			MessageID: msg.MessageID,
			ChatID:    id,
			PollID:    msg.Poll.ID,
		})
	}

	r.UpdatedAt = time.Now().Unix()
	if err := tg.db.Set(r.ID, r); err != nil {
		return err
	}

	return nil
}

func (tg *TGService) getReadyRecord() (r storage.Record, err error) {
	records := make(map[string]storage.Record)
	err = tg.db.Iterator(func(k string, r storage.Record) {
		if r.Status == storage.PROCESS {
			records[k] = r
		}
	})
	if err != nil {
		return r, err
	}

	if len(records) == 0 {
		return r, e.ErrNotFound
	}

	keys := reflect.ValueOf(records).MapKeys()
	r = records[keys[0].String()]

	for _, v := range records {
		if r.UpdatedAt > v.UpdatedAt {
			r = v
		}
	}

	return
}

func (tg *TGService) updateService(ch tgbotapi.UpdatesChannel) {
	for update := range ch {
		switch {
		case update.Message != nil:
			u := strings.Split(config.Config.Users, ",")
			if slices.Contains(u, update.Message.From.UserName) {
				tg.liveChats.Set(update.Message.Chat.ID)
			}
		case update.PollAnswer != nil:
			r, err := tg.findByPollID(update.PollAnswer.PollID)
			if err != nil {
				logrus.Error(err)

				continue
			}

			r.Status = storage.DONE
			r.Option = update.PollAnswer.OptionIDs[0]
			r.Text = r.Task.Buttons[r.Option]

			if err := tg.db.Set(r.ID, r); err != nil {
				logrus.Error(err)
			}

			go tg.stopPolls(r.Poll)
		}
	}
}

func (tg *TGService) stopPolls(poll []storage.Poll) {
	for _, v := range poll {
		p := tgbotapi.NewStopPoll(v.ChatID, v.MessageID)

		_, err := tg.bot.Send(p)
		if err != nil {
			logrus.Error(err)
		}
	}
}

func NewTGService(ctx context.Context, db storage.Storage) (*TGService, error) {
	bot, err := tgbotapi.NewBotAPI(config.Config.Token)
	if err != nil {
		return nil, err
	}

	return &TGService{
		ctx:       ctx,
		db:        db,
		bot:       bot,
		liveChats: utils.SetInt64(),
	}, nil
}

func chatPollExists(p []storage.Poll, id int64) bool {
	for _, i := range p {
		if i.ChatID == id {
			return true
		}
	}

	return false
}

func (tg *TGService) findByPollID(id string) (record storage.Record, err error) {
	ok := false
	err = tg.db.Iterator(func(_ string, r storage.Record) {
		for _, p := range r.Poll {
			if p.PollID == id {
				record = r
				ok = true

				return
			}
		}
	})

	if err != nil {
		return record, err
	}

	if !ok {
		return record, e.ErrNotFound
	}

	return record, nil
}
