package informer

import (
	"sync"
)

// threadSafeStore 线程安全的本地缓存
type threadSafeStore struct {
	lock  sync.RWMutex
	items map[string]interface{}
}

func NewThreadSafeStore() Store {
	return &threadSafeStore{
		items: make(map[string]interface{}),
	}
}

func (s *threadSafeStore) Add(key string, obj interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.items[key] = obj
	return nil
}

func (s *threadSafeStore) Update(key string, obj interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.items[key] = obj
	return nil
}

func (s *threadSafeStore) Delete(key string) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	delete(s.items, key)
	return nil
}

func (s *threadSafeStore) Get(key string) (interface{}, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()
	item, exists := s.items[key]
	return item, exists
}

func (s *threadSafeStore) List() []interface{} {
	s.lock.RLock()
	defer s.lock.RUnlock()
	list := make([]interface{}, 0, len(s.items))
	for _, item := range s.items {
		list = append(list, item)
	}
	return list
}

func (s *threadSafeStore) GetKeys() []string {
	s.lock.RLock()
	defer s.lock.RUnlock()
	keys := make([]string, 0, len(s.items))
	for k := range s.items {
		keys = append(keys, k)
	}
	return keys
}

func (s *threadSafeStore) Replace(items map[string]interface{}) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.items = items
	return nil
}

