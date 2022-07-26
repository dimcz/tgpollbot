package db

import (
	"context"
	"net/url"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

func SSearch(ctx context.Context, client *redis.Client, set, pattern string) (result []string, err error) {
	var cursor uint64 = 0

	for {
		var keys []string
		keys, cursor, err = client.SScan(ctx, set, cursor, pattern, 0).Result()

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

func SPRem(ctx context.Context, client *redis.Client, set, pattern string) error {
	keys, err := SSearch(ctx, client, set, pattern)
	if err != nil {
		return err
	}

	members := make([]interface{}, 0, len(keys))
	for _, v := range keys {
		members = append(members, v)
	}

	return client.SRem(ctx, set, members...).Err()
}

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
