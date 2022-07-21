package badger

import (
	"bytes"
	"encoding/gob"
	"os"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/dimcz/tgpollbot/lib/e"
	"github.com/dimcz/tgpollbot/storage"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const RecordTTL = 7 * 24 * 60 * 60

type Client struct {
	db *badger.DB
}

func (cli *Client) Close() {
	if err := cli.db.Close(); err != nil {
		logrus.Error(err)
	}
}

func (cli *Client) Set(key string, r storage.Record) error {
	var buffer bytes.Buffer

	if err := gob.NewEncoder(&buffer).Encode(r); err != nil {
		return errors.Wrap(err, "failed to encode struct")
	}

	wb := cli.db.NewWriteBatch()
	defer wb.Cancel()

	entry := badger.NewEntry([]byte(key), buffer.Bytes()).
		WithMeta(0).
		WithTTL(time.Duration(RecordTTL * time.Second.Nanoseconds()))
	if err := wb.SetEntry(entry); err != nil {
		return errors.Wrap(err, "failed to write data to cache")
	}

	if err := wb.Flush(); err != nil {
		return errors.Wrap(err, "failed to flush data to cache")
	}

	return nil
}

func (cli *Client) Get(key string) (r storage.Record, err error) {
	var buffer []byte
	err = cli.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		buffer, err = item.ValueCopy(nil)

		return err
	})

	if err != nil {
		if err == badger.ErrKeyNotFound {
			return r, e.ErrNotFound
		}

		return r, e.NewInternal(err.Error())
	}

	if err := gob.NewDecoder(bytes.NewBuffer(buffer)).Decode(&r); err != nil {
		return r, e.NewInternal(err.Error())
	}

	return r, nil
}

func (cli *Client) Iterator(f func(k string, r storage.Record)) error {
	err := cli.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10

		it := txn.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()

			err := item.Value(func(val []byte) error {
				var r storage.Record
				if err := gob.NewDecoder(bytes.NewBuffer(val)).Decode(&r); err != nil {
					return err
				}

				f(string(key), r)

				return nil
			})
			if err != nil {
				return err
			}
		}

		return nil
	})

	return err
}

func (cli *Client) FindByPollID(id string) (record storage.Record, err error) {
	ok := false
	err = cli.Iterator(func(_ string, r storage.Record) {
		for _, p := range r.Poll {
			if p.PollID == id {
				record = r
				ok = true

				return
			}
		}
	})

	if err != nil {
		return record, err
	}

	if !ok {
		return record, e.ErrNotFound
	}

	return record, nil
}

func Open(path string) (*Client, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, os.FileMode(0700)); err != nil {
			return nil, errors.Wrap(err, "failed to create directory")
		}
	}

	opts := badger.DefaultOptions(path).
		WithDir(path).
		WithValueDir(path).
		WithSyncWrites(false).
		WithValueThreshold(256).
		WithCompactL0OnClose(true)

	db, err := badger.Open(opts)
	if err != nil {
		return nil, errors.Wrap(err, "badger open failed")
	}

	go cleanup(db)

	return &Client{db}, nil
}

func cleanup(db *badger.DB) {
	timer := time.NewTicker(5 * time.Minute)
	defer timer.Stop()

	for range timer.C {
	loop:
		if err := db.RunValueLogGC(0.7); err == nil {
			goto loop
		}
	}
}
