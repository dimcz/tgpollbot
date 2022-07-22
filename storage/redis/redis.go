package redis

import (
	"context"
	"encoding/json"
	"net/url"
	"strconv"
	"time"

	"github.com/dimcz/tgpollbot/config"
	"github.com/dimcz/tgpollbot/lib/e"
	"github.com/dimcz/tgpollbot/storage"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

var _ storage.Storage = &Client{}

type Client struct {
	ctx context.Context
	db  *redis.Client
}

func (cli *Client) Set(key string, r storage.Record) error {
	data, err := json.Marshal(r)
	if err != nil {
		return err
	}

	return cli.db.Set(cli.ctx, key, data, storage.RecordTTL*time.Second).Err()
}

func (cli *Client) Get(key string) (r storage.Record, err error) {
	data, err := cli.db.Get(cli.ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return r, e.ErrNotFound
		}
	}

	err = json.Unmarshal(data, &r)

	return
}

func (cli *Client) Iterator(f func(k string, r storage.Record)) error {
	var keys []string

	iter := cli.db.Scan(cli.ctx, 0, "*", 0).Iterator()

	for iter.Next(cli.ctx) {
		keys = append(keys, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return err
	}

	if len(keys) == 0 {
		return e.ErrNotFound
	}

	data, err := cli.db.MGet(cli.ctx, keys...).Result()
	if err != nil {
		return err
	}

	for _, d := range data {
		if v, ok := d.(string); ok {
			var r storage.Record

			if err := json.Unmarshal([]byte(v), &r); err != nil {
				continue
			}

			f(r.ID, r)
		}
	}

	return nil
}

func (cli *Client) Close() {
	if err := cli.db.Close(); err != nil {
		logrus.Error(err)
	}
}

func Connect(ctx context.Context) (*Client, error) {
	u, err := url.Parse(config.Config.RedisDB)
	if err != nil {
		return nil, err
	}

	num, err := strconv.Atoi(u.Path[1:])
	if err != nil {
		return nil, err
	}

	username, password := "", ""
	if u.User != nil {
		username = u.User.Username()

		if p, ok := u.User.Password(); ok {
			password = p
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:     u.Host,
		Username: username,
		Password: password,
		DB:       num,
	})

	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(c).Err(); err != nil {
		return nil, err
	}

	return &Client{ctx: ctx, db: client}, nil
}
