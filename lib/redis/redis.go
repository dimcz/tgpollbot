package redis

import (
	"context"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/dimcz/tgpollbot/lib/e"
	"github.com/dimcz/tgpollbot/storage"
	"github.com/go-redis/redis/v8"
	"github.com/sirupsen/logrus"
)

type Client struct {
	db *redis.Client

	index int64
	len   int64
	mu    sync.Mutex
}

// List

func (cli *Client) RPush(ctx context.Context, list string, r storage.Request) error {
	defer cli.mu.Unlock()
	cli.mu.Lock()

	if err := cli.db.RPush(ctx, list, r).Err(); err != nil {
		return err
	}

	cli.len += 1

	return nil
}

func (cli *Client) Next(ctx context.Context, list string, value interface{}) error {
	defer cli.mu.Unlock()
	cli.mu.Lock()

	if cli.len == 0 {
		return e.ErrNotFound
	}

	err := cli.db.LIndex(ctx, list, cli.index).Scan(value)
	if err != nil {
		if err == redis.Nil {
			err = e.ErrNotFound
		}

		return err
	}

	cli.index += 1
	if cli.index == cli.len {
		cli.index = 0
	}

	return nil
}

func (cli *Client) LRange(ctx context.Context, list string, results interface{}) error {
	return cli.db.LRange(ctx, list, 0, -1).ScanSlice(results)
}

func (cli *Client) LRem(ctx context.Context, list string, member interface{}) error {
	defer cli.mu.Unlock()
	cli.mu.Lock()

	if err := cli.db.LRem(ctx, list, 1, member).Err(); err != nil {
		return err
	}

	cli.len -= 1
	if cli.index == cli.len {
		cli.index = 0
	}

	return nil
}

// Set

func (cli *Client) SRem(ctx context.Context, set string, members ...interface{}) error {
	return cli.db.SRem(ctx, set, members...).Err()
}

func (cli *Client) SMembers(ctx context.Context, set string, result interface{}) error {
	return cli.db.SMembers(ctx, set).ScanSlice(result)
}

func (cli *Client) SSearch(ctx context.Context, set, pattern string) (result []string, err error) {
	var cursor uint64 = 0

	for {
		var keys []string
		keys, cursor, err = cli.db.SScan(ctx, set, cursor, pattern, 0).Result()

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

func (cli *Client) SDelete(ctx context.Context, set, pattern string) error {
	keys, err := cli.SSearch(ctx, set, pattern)
	if err != nil {
		return err
	}

	members := make([]interface{}, 0, len(keys))
	for _, v := range keys {
		members = append(members, v)
	}

	return cli.SRem(ctx, set, members...)
}

func (cli *Client) SAdd(ctx context.Context, set string, member interface{}) error {
	return cli.db.SAdd(ctx, set, member).Err()
}

// Keys

func (cli *Client) Set(ctx context.Context, key string, val interface{}, ttl time.Duration) error {
	return cli.db.Set(ctx, key, val, ttl).Err()
}

func (cli *Client) Get(ctx context.Context, key string, val interface{}) error {
	return cli.db.Get(ctx, key).Scan(val)
}

// --

func (cli *Client) Close() {
	if err := cli.db.Close(); err != nil {
		logrus.Error(err)
	}
}

func (cli *Client) InitQueue(ctx context.Context, list string) (err error) {
	cli.len, err = cli.db.LLen(ctx, list).Result()

	return err
}

func Connect(conn string) (*Client, error) {
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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return &Client{
		db: client,
	}, nil
}
