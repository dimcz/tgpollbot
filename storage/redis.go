package storage

import (
	"context"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/dimcz/tgpollbot/lib/e"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

const RecordTTL = 7 * 24 * 60 * 60

const (
	RecordsList       = "records"
	SessionSet        = "session"
	PollPrefix        = "poll:"
	PollRequestPrefix = "request:"
	RecordPrefix      = "record:"
)

type Client struct {
	ctx context.Context
	db  *redis.Client

	index int64
	len   int64
	mu    sync.Mutex
}

// Records

func (cli *Client) Insert(r Record) error {
	defer cli.mu.Unlock()
	cli.mu.Lock()

	if err := cli.db.RPush(cli.ctx, RecordsList, r).Err(); err != nil {
		return err
	}

	cli.len += 1

	return nil
}

func (cli *Client) Next() (r Record, err error) {
	defer cli.mu.Unlock()
	cli.mu.Lock()

	if cli.len == 0 {
		return r, e.ErrNotFound
	}

	err = cli.db.LIndex(cli.ctx, RecordsList, cli.index).Scan(&r)
	if err != nil {
		if err == redis.Nil {
			return r, e.ErrNotFound
		}

		logrus.Error(err)
		return r, err
	}

	cli.index += 1
	if cli.index == cli.len {
		cli.index = 0
	}

	return
}

func (cli *Client) Drop(r Record) error {
	defer cli.mu.Unlock()
	cli.mu.Lock()

	if err := cli.db.LRem(cli.ctx, RecordsList, 1, r).Err(); err != nil {
		return err
	}

	cli.len -= 1
	if cli.index == cli.len {
		cli.index = 0
	}

	return nil
}

// Chats

func (cli *Client) AddChat(chatId int64) error {
	return cli.db.SAdd(cli.ctx, SessionSet, chatId).Err()
}

func (cli *Client) DropChat(chatId int64) error {
	return cli.db.SRem(cli.ctx, SessionSet, chatId).Err()
}

func (cli *Client) GetChats() (result []int64, err error) {
	err = cli.db.SMembers(cli.ctx, SessionSet).ScanSlice(&result)

	return
}

// Misc

func (cli *Client) Exists(key string) bool {
	r, err := cli.db.Exists(cli.ctx, key).Result()
	if err == nil && r == 1 {
		return true
	}

	return false
}

func (cli *Client) Set(key string, val interface{}) error {
	return cli.db.Set(cli.ctx, key, val, RecordTTL*time.Second).Err()
}

func (cli *Client) Get(key string, val interface{}) error {
	return cli.db.Get(cli.ctx, key).Scan(val)
}

func (cli *Client) Close() {
	if err := cli.db.Close(); err != nil {
		logrus.Error(err)
	}
}

func Connect(ctx context.Context, conn string) (*Client, error) {
	u, err := url.Parse(conn)
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
