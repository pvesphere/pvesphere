package informer

import (
	"context"
)

// DeltaType 表示资源的变化类型
type DeltaType string

const (
	DeltaAdded   DeltaType = "Added"
	DeltaUpdated DeltaType = "Updated"
	DeltaDeleted DeltaType = "Deleted"
)

// Delta 表示资源的一个变化
type Delta struct {
	Type   DeltaType
	Object interface{}
}

// EventHandler 处理资源变化事件
type EventHandler interface {
	OnAdd(obj interface{}) error
	OnUpdate(oldObj, newObj interface{}) error
	OnDelete(obj interface{}) error
}

// Store 接口定义本地缓存
type Store interface {
	Add(key string, obj interface{}) error
	Update(key string, obj interface{}) error
	Delete(key string) error
	Get(key string) (interface{}, bool)
	List() []interface{}
	GetKeys() []string
	Replace(items map[string]interface{}) error
}

// ListFunc 列出所有资源
type ListFunc func(ctx context.Context) ([]interface{}, error)

// WatchFunc 监听资源变化（使用轮询模拟）
type WatchFunc func(ctx context.Context, version string) (string, []interface{}, error)

// ListWatcher 接口
type ListWatcher interface {
	List(ctx context.Context) ([]interface{}, error)
	Watch(ctx context.Context, version string) (string, []interface{}, error)
	GetResourceVersion(obj interface{}) string
}

