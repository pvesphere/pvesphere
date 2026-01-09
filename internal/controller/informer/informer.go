package informer

import (
	"context"
	"sync"
	"time"

	"pvesphere/pkg/log"
	"go.uber.org/zap"
)

type Informer interface {
	Run(ctx context.Context)
	Stop()
	AddEventHandler(handler EventHandler)
	HasSynced() bool
}

type informer struct {
	name      string
	reflector *Reflector
	deltaFIFO *DeltaFIFO
	handlers  []EventHandler
	logger    *log.Logger
	stopCh    chan struct{}
	wg        sync.WaitGroup
	keyFunc   func(obj interface{}) (string, error)
	store     Store
}

func NewInformer(
	name string,
	listWatcher ListWatcher,
	keyFunc func(obj interface{}) (string, error),
	logger *log.Logger,
	resyncPeriod time.Duration,
) Informer {
	store := NewThreadSafeStore()
	deltaFIFO := NewDeltaFIFO(keyFunc, store)
	reflector := NewReflector(name, listWatcher, deltaFIFO, logger, resyncPeriod)

	return &informer{
		name:      name,
		reflector: reflector,
		deltaFIFO: deltaFIFO,
		handlers:  make([]EventHandler, 0),
		logger:    logger,
		stopCh:    make(chan struct{}),
		keyFunc:   keyFunc,
		store:     store,
	}
}

func (i *informer) Run(ctx context.Context) {
	i.reflector.Run(ctx)

	i.wg.Add(1)
	go func() {
		defer i.wg.Done()
		i.processLoop(ctx)
	}()
}

func (i *informer) Stop() {
	close(i.stopCh)
	i.reflector.Stop()
	i.wg.Wait()
}

func (i *informer) AddEventHandler(handler EventHandler) {
	i.handlers = append(i.handlers, handler)
}

func (i *informer) HasSynced() bool {
	return i.deltaFIFO.HasSynced()
}

func (i *informer) processLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-i.stopCh:
			return
		default:
			err := i.deltaFIFO.Pop(func(delta Delta) error {
				return i.processDelta(delta)
			})
			if err != nil {
				i.logger.Error("process delta failed", zap.String("name", i.name), zap.Error(err))
			}
			// 避免 CPU 占用过高
			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (i *informer) processDelta(delta Delta) error {
	for _, handler := range i.handlers {
		var err error
		switch delta.Type {
		case DeltaAdded:
			err = handler.OnAdd(delta.Object)
		case DeltaUpdated:
			// 对于更新，我们需要获取旧对象
			key, _ := i.keyFunc(delta.Object)
			oldObj, exists := i.store.Get(key)
			if exists {
				err = handler.OnUpdate(oldObj, delta.Object)
			} else {
				err = handler.OnAdd(delta.Object)
			}
		case DeltaDeleted:
			err = handler.OnDelete(delta.Object)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

