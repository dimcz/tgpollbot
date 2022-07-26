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
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

const SendTimeout = 1 * time.Second

type TGService struct {
	ctx context.Context
	cli *storage.Client
	bot *tgbotapi.BotAPI

	allowList []int64
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
				if !strings.Contains(err.Error(), "not_found") {
					logrus.Error("failed to send service with error: ", err)
				}
			}

			next = time.Now().Add(SendTimeout)
			timer.Reset(time.Until(next))
		}
	}
}

func (tg *TGService) send() error {
	sessions, err := tg.cli.Sessions()
	if err != nil {
		return errors.Wrap(err, "failed to get sessions")
	}

	r, err := tg.cli.Next()
	if err != nil {
		return errors.Wrap(err, "failed to get next request")
	}

	for _, chatId := range sessions {
		k, _ := tg.cli.SSearch(storage.PollRequestsSet,
			fmt.Sprintf("%s:%d:*", r.ID, chatId))
		if len(k) > 0 {
			continue
		}

		poll := tgbotapi.NewPoll(chatId, r.Task.Message, r.Task.Buttons...)
		poll.IsAnonymous = false
		poll.AllowsMultipleAnswers = false

		msg, err := tg.bot.Send(poll)
		if err != nil {
			logrus.Error("failed send new poll with err: ", err)

			if err := tg.cli.SRem(storage.SessionSet, chatId); err != nil {
				logrus.Error("failed removing error session with err: ", err)
			}
		}

		logrus.Infof("send %s poll from %s request to %d chat",
			msg.Poll.ID, r.ID, chatId)

		err = tg.cli.SAdd(storage.PollRequestsSet,
			fmt.Sprintf("%s:%d:%s", r.ID, chatId, msg.Poll.ID))
		if err != nil {
			logrus.Error("failed adding poll request with err: ", err)
		}
	}

	return nil
}

func (tg *TGService) updateService(ch tgbotapi.UpdatesChannel) {
	for update := range ch {
		switch {
		case update.Message != nil:
			message := tg.greetingUser(update.Message)
			tg.message(update.Message.Chat.ID, message)
		case update.PollAnswer != nil:
			keys, err := tg.cli.SSearch(storage.PollRequestsSet, "*:*:"+update.PollAnswer.PollID)
			if err != nil {
				if err != e.ErrNotFound {
					logrus.Error("failed to search poll request with error: ", err)
				}

				continue
			}

			array := strings.Split(keys[0], ":")
			reqId := array[0]

			text, err := tg.deleteRequest(reqId, update.PollAnswer.OptionIDs[0])
			if err != nil {
				logrus.Error("failed to delete request with error: ", err)
			}

			dto := storage.DTO{
				Status: storage.RecordPollDone,
				Text:   text,
				Option: &update.PollAnswer.OptionIDs[0],
			}

			if err := tg.cli.Set(storage.RecordPrefix+reqId, dto); err != nil {
				logrus.Error("failed to set record with error: ", err)
			}

			logrus.Infof("got answer to poll %s for request %s from %s/%d",
				update.PollAnswer.PollID, reqId,
				update.PollAnswer.User.UserName, update.PollAnswer.User.ID)
		}
	}
}

func (tg *TGService) deleteRequest(reqId string, index int) (text string, err error) {
	var requests []storage.Request

	if err := tg.cli.SDelete(storage.PollRequestsSet, reqId+":*"); err != nil {
		logrus.Error("failed to deleting poll request with error: ", err)
	}

	if err = tg.cli.LRange(storage.RecordsList, &requests); err != nil {
		return
	}

	for _, v := range requests {
		if v.ID == reqId {
			return v.Buttons[index], tg.cli.LRem(storage.RecordsList, v)
		}
	}

	return text, e.ErrNotFound
}

func (tg *TGService) greetingUser(message *tgbotapi.Message) string {
	if slices.Contains(tg.allowList, message.From.ID) {
		err := tg.cli.SAdd(storage.SessionSet, message.Chat.ID)
		if err == nil {
			return "You have access granted"
		}

		logrus.Error("could not set user session with error: ", err)
	}

	return fmt.Sprintf("User %d does not have access", message.From.ID)
}

func (tg *TGService) message(id int64, msg string) {
	message := tgbotapi.NewMessage(id, msg)
	if _, err := tg.bot.Send(message); err != nil {
		logrus.Error("could not send message to Telegram with error: ", err)
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
