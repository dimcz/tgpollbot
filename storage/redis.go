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
	PollRequestsSet = "pollRequestsSet"
	RecordsList     = "requestsList"
	RecordPrefix    = "record:"
	SessionSet      = "sessionSet"
)

type Client struct {
	ctx context.Context
	db  *redis.Client

	index int64
	len   int64
	mu    sync.Mutex
}

// List

func (cli *Client) RPush(list string, r Request) error {
	defer cli.mu.Unlock()
	cli.mu.Lock()

	if err := cli.db.RPush(cli.ctx, list, r).Err(); err != nil {
		return err
	}

	cli.len += 1

	return nil
}

func (cli *Client) Next() (r Request, err error) {
	defer cli.mu.Unlock()
	cli.mu.Lock()

	if cli.len == 0 {
		err = e.ErrNotFound

		return
	}

	err = cli.db.LIndex(cli.ctx, RecordsList, cli.index).Scan(&r)
	if err != nil {
		if err == redis.Nil {
			err = e.ErrNotFound
		}

		return
	}

	cli.index += 1
	if cli.index == cli.len {
		cli.index = 0
	}

	return
}

func (cli *Client) LRange(list string, results interface{}) error {
	return cli.db.LRange(cli.ctx, list, 0, -1).ScanSlice(results)
}

func (cli *Client) LRem(list string, member interface{}) error {
	defer cli.mu.Unlock()
	cli.mu.Lock()

	if err := cli.db.LRem(cli.ctx, list, 1, member).Err(); err != nil {
		return err
	}

	cli.len -= 1
	if cli.index == cli.len {
		cli.index = 0
	}

	return nil
}

// Set

func (cli *Client) SRem(set string, members ...interface{}) error {
	return cli.db.SRem(cli.ctx, set, members...).Err()
}

func (cli *Client) SMembers(set string, result []interface{}) error {
	return cli.db.SMembers(cli.ctx, set).ScanSlice(&result)
}

func (cli *Client) Sessions() (result []int64, err error) {
	if err = cli.db.SMembers(cli.ctx, SessionSet).ScanSlice(&result); err != nil {
		return
	}

	if len(result) == 0 {
		err = e.ErrNotFound
	}

	return
}

func (cli *Client) SSearch(set, pattern string) (result []string, err error) {
	var cursor uint64 = 0

	for {
		var keys []string
		keys, cursor, err = cli.db.SScan(cli.ctx, set, cursor, pattern, 0).Result()

		if err != nil {
			return result, err
		}

		result = append(result, keys...)

		if cursor == 0 {
			break
		}
	}

	return
}

func (cli *Client) SDelete(set, pattern string) error {
	keys, err := cli.SSearch(set, pattern)
	if err != nil {
		return err
	}

	members := make([]interface{}, 0, len(keys))
	for _, v := range keys {
		members = append(members, v)
	}

	return cli.SRem(set, members...)
}

func (cli *Client) SAdd(set string, member interface{}) error {
	return cli.db.SAdd(cli.ctx, set, member).Err()
}

// Keys

func (cli *Client) Set(key string, val interface{}) error {
	return cli.db.Set(cli.ctx, key, val, RecordTTL*time.Second).Err()
}

func (cli *Client) Get(key string, val interface{}) error {
	return cli.db.Get(cli.ctx, key).Scan(val)
}

// --

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

	return &Client{
		ctx: ctx,
		db:  client,
		len: client.LLen(ctx, RecordsList).Val(),
	}, nil
}
