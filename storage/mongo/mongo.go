package mongo

import (
	"context"
	"net/url"
	"time"

	"github.com/dimcz/tgpollbot/lib/e"
	"github.com/dimcz/tgpollbot/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const RecordsCollection = "records"

var _ storage.Storage = &Client{}

type Client struct {
	ctx    context.Context
	client *mongo.Client
	db     *mongo.Database
}

func (cli *Client) Set(key string, r storage.Record) (err error) {
	_, err = cli.db.Collection(RecordsCollection).
		ReplaceOne(cli.ctx, bson.M{"id": key}, r)

	return
}

func (cli *Client) Get(key string) (r storage.Record, err error) {
	err = cli.db.Collection(RecordsCollection).
		FindOne(cli.ctx, bson.M{"id": key}).Decode(&r)

	return
}

func (cli *Client) GetAndUpdate(key string, sets storage.Opts) (r storage.Record, err error) {
	set := make(bson.M, len(sets))
	for k, v := range sets {
		set[k] = v
	}

	err = cli.db.Collection(RecordsCollection).
		FindOneAndUpdate(cli.ctx, bson.M{"id": key}, set).Decode(&r)

	return
}

func (cli *Client) Find(clause storage.Opts, opts storage.Opts) (r []storage.Record, err error) {
	filter := make(bson.M, len(clause))
	for k, v := range clause {
		filter[k] = v
	}

	o := options.Find()
	if v, ok := opts["sort"]; ok {
		o.SetSort(v)
	}

	cursor, err := cli.db.Collection(RecordsCollection).
		Find(cli.ctx, filter, o)
	if err != nil {
		return nil, err
	}

	defer func(cursor *mongo.Cursor, ctx context.Context) {
		_ = cursor.Close(ctx)
	}(cursor, cli.ctx)

	err = cursor.All(cli.ctx, &r)
	if err != nil {
		return nil, err
	}

	if len(r) == 0 {
		return nil, e.ErrNotFound
	}

	return
}

func (cli *Client) Iterator(f func(k string, r storage.Record)) error {
	records, err := cli.Find(storage.Opts{}, storage.Opts{})
	if err != nil {
		return err
	}

	for _, r := range records {
		f(r.ID, r)
	}

	return nil
}

func (cli *Client) Close() {
	if err := cli.client.Disconnect(context.Background()); err != nil {
		logrus.Error(err)
	}
}

func createIndex(ctx context.Context, db *mongo.Database) (err error) {
	index := mongo.IndexModel{
		Keys: bson.D{
			{Key: "id", Value: 1},
		},
		Options: options.Index().SetUnique(true).SetExpireAfterSeconds(storage.RecordTTL),
	}

	_, err = db.Collection(RecordsCollection).Indexes().CreateOne(ctx, index)

	return
}

func Connect(ctx context.Context, conn string) (*Client, error) {
	u, err := url.Parse(conn)
	if err != nil {
		return nil, err
	}

	opts := options.Client().ApplyURI(conn)

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		return nil, errors.Wrap(err, "failed connect to mongodb")
	}

	if err = client.Ping(ctx, nil); err != nil {
		return nil, errors.Wrap(err, "failed to mongo.Client.Ping")
	}

	db := client.Database(u.Path[1:])

	if err := createIndex(ctx, db); err != nil {
		return nil, err
	}

	return &Client{
		ctx:    ctx,
		client: client,
		db:     db,
	}, nil
}
