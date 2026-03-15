package repository

type KeyValueStorage interface {
	HasKey(key string) bool
	WriteValue(key string, value string)
	GetValue(key string) string
}

type MapKeyValueStorage struct {
	dict map[string]string
}

func NewMapKeyValueStorage() KeyValueStorage {
	return &MapKeyValueStorage{dict: make(map[string]string)}
}

func (a *MapKeyValueStorage) HasKey(key string) bool {
	_, ok := a.dict[key]
	return ok

}

func (a *MapKeyValueStorage) WriteValue(key string, value string) {
	a.dict[key] = value
}

func (a *MapKeyValueStorage) GetValue(key string) string {
	return a.dict[key]
}
