package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dimcz/tgpollbot/config"
	"github.com/dimcz/tgpollbot/storage"
	"github.com/go-redis/redis/v8"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/slices"
)

const PauseBetweenPolls = 500 * time.Millisecond

type TGService struct {
	ctx    context.Context
	cancel func()
	group  sync.WaitGroup

	rc        *redis.Client
	cache     *Cache
	bot       *tgbotapi.BotAPI
	allowList []int64
}

func (tg *TGService) Close() {
	tg.bot.StopReceivingUpdates()

	tg.cancel()
	tg.group.Wait()
}

func (tg *TGService) Run() {
	//	tg.group.Add(2)
	//
	//	ch := tg.bot.GetUpdatesChan(tgbotapi.UpdateConfig{})
	//	go tg.updateService(ch)
	//
	//	go tg.sendService()
}

func (tg *TGService) sendService() {
	defer tg.group.Done()

	for {
		select {
		case <-tg.ctx.Done():
			return
		default:
			if err := tg.send(); err != nil {
				logrus.Error("failed to send service with error: ", err)
			}

			time.Sleep(PauseBetweenPolls)
		}
	}
}

func (tg *TGService) send() error {
	var sessions []int64
	if err := tg.rc.SMembers(tg.ctx, storage.SessionSet).ScanSlice(&sessions); err != nil {
		return errors.Wrap(err, "failed to get sessions")
	}

	if len(sessions) == 0 {
		return nil
	}

	requestID := tg.rc.LMove(
		tg.ctx,
		storage.RecordsList,
		storage.RecordsList,
		"LEFT",
		"RIGHT").Val()

	r, err := tg.cache.Get(tg.ctx, requestID)
	if err != nil {
		if err == redis.Nil {
			return nil
		}

		return errors.Wrap(err, "failed to get next request")
	}

	for _, chatId := range sessions {
		_, err = tg.searchKeys(storage.PollRequestsSet, fmt.Sprintf("%s:%d:*", requestID, chatId))
		if err == nil {
			continue
		}

		poll := tgbotapi.NewPoll(chatId, r.Task.Message, r.Task.Buttons...)
		poll.IsAnonymous = false
		poll.AllowsMultipleAnswers = false

		msg, err := tg.bot.Send(poll)
		if err != nil {
			return errors.Wrap(err, "failed send new poll")
		}

		logrus.Infof("send %s poll from %s request to %d chat",
			msg.Poll.ID, requestID, chatId)

		err = tg.rc.SAdd(tg.ctx, storage.PollRequestsSet,
			fmt.Sprintf("%s:%d:%s", requestID, chatId, msg.Poll.ID)).Err()
		if err != nil {
			return errors.Wrap(err, "failed adding poll request with err:")
		}
	}

	return nil
}

func (tg *TGService) remByPattern(set, pattern string) error {
	keys, err := tg.searchKeys(set, pattern)
	if err != nil {
		return err
	}

	members := make([]interface{}, 0, len(keys))
	for _, v := range keys {
		members = append(members, v)
	}

	return tg.rc.SRem(tg.ctx, set, members...).Err()
}

func (tg *TGService) updateService(ch tgbotapi.UpdatesChannel) {
	defer tg.group.Done()

	for update := range ch {
		switch {
		case update.Message != nil:
			message := tg.greetingUser(update.Message)
			tg.message(update.Message.Chat.ID, message)
		case update.PollAnswer != nil:
			keys, err := tg.searchKeys(storage.PollRequestsSet, "*:*:"+update.PollAnswer.PollID)
			if err != nil {
				logrus.Error("failed to search poll request with error: ", err)

				continue
			}

			requestId := strings.Split(keys[0], ":")[0]

			r, err := tg.cache.Get(tg.ctx, requestId)
			if err != nil {
				logrus.Error("failed to get request from cache with error: ", err)

				continue
			}

			r.Status = storage.RecordPollDone
			r.Option = &update.PollAnswer.OptionIDs[0]

			if err = tg.cache.Set(tg.ctx, requestId, r); err != nil {
				logrus.Error("could not set record to cache with error: ", err)

				continue
			}

			if err := tg.removeRequest(requestId); err != nil {
				logrus.Error("failed to delete request with error: ", err)
			}

			logrus.Infof("got answer to poll %s for request %s from %s/%d",
				update.PollAnswer.PollID, requestId,
				update.PollAnswer.User.UserName, update.PollAnswer.User.ID)
		}
	}
}

func (tg *TGService) removeRequest(requestId string) error {
	if err := tg.remByPattern(storage.PollRequestsSet, requestId+":*"); err != nil {
		logrus.Error("failed to deleting poll request with error: ", err)
	}

	if err := tg.rc.LRem(tg.ctx, storage.RecordsList, 1, requestId).Err(); err != nil {
		logrus.Error("failed to deleting request with error: ", err)
	}

	return nil
}

func (tg *TGService) greetingUser(message *tgbotapi.Message) string {
	if slices.Contains(tg.allowList, message.From.ID) {
		err := tg.rc.SAdd(tg.ctx, storage.SessionSet, message.Chat.ID).Err()
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

func (tg *TGService) searchKeys(set, pattern string) (result []string, err error) {
	var cursor uint64 = 0

	for {
		var keys []string
		keys, cursor, err = tg.rc.SScan(tg.ctx, set, cursor, pattern, 0).Result()

		if err != nil {
			return result, err
		}

		result = append(result, keys...)

		if cursor == 0 {
			break
		}
	}

	if len(result) == 0 {
		return result, redis.Nil
	}

	return
}

func NewTGService(rc *redis.Client, cache *Cache) (*TGService, error) {
	bot, err := tgbotapi.NewBotAPI(config.Config.Token)
	if err != nil {
		return nil, errors.Wrap(err, "could not init Telegram Bot API")
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

	ctx, cancel := context.WithCancel(context.Background())

	return &TGService{
		ctx:    ctx,
		cancel: cancel,

		rc:        rc,
		cache:     cache,
		bot:       bot,
		allowList: allowList,
	}, nil
}
