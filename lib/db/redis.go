package db

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

func RedisConnect(conn string) (*redis.Client, error) {
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

	if err = client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}
