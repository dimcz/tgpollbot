package storage

type Storage interface {
	Set(key string, r Record) error
	Get(key string) (Record, error)
	Iterator(f func(k string, r Record)) error
	FindByPollID(id string) (Record, error)
	Close()
}
