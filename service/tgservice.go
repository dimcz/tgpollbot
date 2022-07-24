package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dimcz/tgpollbot/config"
	"github.com/dimcz/tgpollbot/lib/e"
	"github.com/dimcz/tgpollbot/storage"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

const SendTimeout = 5 * time.Second

type TGService struct {
	ctx context.Context
	cli *storage.Client
	bot *tgbotapi.BotAPI

	allowList []int64
}

func (tg *TGService) Message(id int64, msg string) {
	message := tgbotapi.NewMessage(id, msg)
	if _, err := tg.bot.Send(message); err != nil {
		logrus.Error(err)
	}
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
	chats, err := tg.cli.GetChats()
	if err != nil {
		return err
	}

	if len(chats) == 0 {
		return nil
	}

	r, err := tg.cli.Next()
	if err != nil {
		if err == e.ErrNotFound {
			return nil
		}

		return err
	}

	for _, chatId := range chats {
		if tg.cli.Exists(fmt.Sprintf("%s%s:%d", storage.PollRequestPrefix, r.ID, chatId)) {
			continue
		}

		poll := tgbotapi.NewPoll(chatId, r.Task.Message, r.Task.Buttons...)
		poll.IsAnonymous = false
		poll.AllowsMultipleAnswers = false

		msg, err := tg.bot.Send(poll)
		if err != nil {
			logrus.Error(err)

			if err := tg.cli.DropChat(chatId); err != nil {
				logrus.Error(err)
			}
		}

		err = tg.cli.Set(fmt.Sprintf("%s%s:%d",
			storage.PollRequestPrefix, r.ID, chatId), msg.Poll.ID)
		if err != nil {
			logrus.Error(err)
		}

		err = tg.cli.Set(storage.PollPrefix+msg.Poll.ID, r)
		if err != nil {
			logrus.Error(err)
		}
	}

	return nil
}

func (tg *TGService) updateService(ch tgbotapi.UpdatesChannel) {
	for update := range ch {
		switch {
		case update.Message != nil:
			switch {
			case slices.Contains(tg.allowList, update.Message.From.ID):
				err := tg.cli.AddChat(update.Message.Chat.ID)
				if err != nil {
					logrus.Error(err)
				}
				tg.Message(update.Message.Chat.ID, "You have access granted")
			default:
				tg.Message(update.Message.Chat.ID,
					fmt.Sprintf("User %d does not have access", update.Message.From.ID))
			}

		case update.PollAnswer != nil:
			r := storage.Record{}
			if err := tg.cli.Get(storage.PollPrefix+update.PollAnswer.PollID, &r); err != nil {
				logrus.Error(err)

				continue
			}

			if err := tg.cli.Drop(r); err != nil {
				logrus.Error(err)
			}

			r.Status = storage.RecordPollDone
			r.Option = &update.PollAnswer.OptionIDs[0]
			r.Text = r.Task.Buttons[*r.Option]

			if err := tg.cli.Set(storage.RecordPrefix+r.ID, r); err != nil {
				logrus.Error(err)
			}
		}
	}
}

func NewTGService(ctx context.Context, cli *storage.Client) (*TGService, error) {
	bot, err := tgbotapi.NewBotAPI(config.Config.Token)
	if err != nil {
		return nil, err
	}

	u := strings.Split(config.Config.Users, ",")
	allowList := make([]int64, 0, len(u))
	for _, v := range u {
		id, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			continue
		}

		allowList = append(allowList, id)
	}

	return &TGService{
		ctx:       ctx,
		cli:       cli,
		bot:       bot,
		allowList: allowList,
	}, nil
}
