package informer

import (
	"sync"
)

type DeltaFIFO struct {
	lock    sync.RWMutex
	items   []Delta
	keyFunc func(obj interface{}) (string, error)
	store   Store
}

func NewDeltaFIFO(keyFunc func(obj interface{}) (string, error), store Store) *DeltaFIFO {
	return &DeltaFIFO{
		items:   make([]Delta, 0),
		keyFunc: keyFunc,
		store:   store,
	}
}

func (f *DeltaFIFO) Add(obj interface{}) error {
	key, err := f.keyFunc(obj)
	if err != nil {
		return err
	}

	f.lock.Lock()
	defer f.lock.Unlock()

	_, exists := f.store.Get(key)
	if exists {
		f.items = append(f.items, Delta{Type: DeltaUpdated, Object: obj})
		f.store.Update(key, obj)
	} else {
		f.items = append(f.items, Delta{Type: DeltaAdded, Object: obj})
		f.store.Add(key, obj)
	}
	return nil
}

func (f *DeltaFIFO) Update(obj interface{}) error {
	return f.Add(obj)
}

func (f *DeltaFIFO) Delete(obj interface{}) error {
	key, err := f.keyFunc(obj)
	if err != nil {
		return err
	}

	f.lock.Lock()
	defer f.lock.Unlock()

	_, exists := f.store.Get(key)
	if exists {
		f.items = append(f.items, Delta{Type: DeltaDeleted, Object: obj})
		f.store.Delete(key)
	}
	return nil
}

func (f *DeltaFIFO) Pop(handler func(delta Delta) error) error {
	f.lock.Lock()
	if len(f.items) == 0 {
		f.lock.Unlock()
		return nil
	}

	delta := f.items[0]
	f.items = f.items[1:]
	f.lock.Unlock()

	err := handler(delta)
	return err
}

func (f *DeltaFIFO) Replace(items []interface{}) error {
	f.lock.Lock()
	defer f.lock.Unlock()

	// 创建新的 map 用于比较
	newItems := make(map[string]interface{})
	oldItems := make(map[string]interface{})

	for _, item := range f.store.List() {
		key, _ := f.keyFunc(item)
		oldItems[key] = item
	}

	// 添加新资源
	for _, item := range items {
		key, _ := f.keyFunc(item)
		newItems[key] = item

		if _, exists := oldItems[key]; !exists {
			f.items = append(f.items, Delta{Type: DeltaAdded, Object: item})
		} else {
			f.items = append(f.items, Delta{Type: DeltaUpdated, Object: item})
		}
		f.store.Add(key, item)
	}

	// 检测删除的资源
	for key, item := range oldItems {
		if _, exists := newItems[key]; !exists {
			f.items = append(f.items, Delta{Type: DeltaDeleted, Object: item})
			f.store.Delete(key)
		}
	}

	return nil
}

func (f *DeltaFIFO) HasSynced() bool {
	f.lock.RLock()
	defer f.lock.RUnlock()
	return len(f.items) == 0
}

