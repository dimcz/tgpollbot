package storage

const RecordTTL = 7 * 24 * 60 * 60

type Storage interface {
	Set(r Record) error
	Get(key string) (Record, error)
	Iterator(f func(k string, r Record)) error
	Close()
}
