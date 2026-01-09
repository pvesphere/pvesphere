package informer

import (
	"context"
	"sync"
	"time"

	"pvesphere/pkg/log"
	"go.uber.org/zap"
)

type Reflector struct {
	name         string
	listWatcher  ListWatcher
	deltaFIFO    *DeltaFIFO
	version      string
	logger       *log.Logger
	stopCh       chan struct{}
	wg           sync.WaitGroup
	resyncPeriod time.Duration
}

func NewReflector(
	name string,
	listWatcher ListWatcher,
	deltaFIFO *DeltaFIFO,
	logger *log.Logger,
	resyncPeriod time.Duration,
) *Reflector {
	return &Reflector{
		name:         name,
		listWatcher:  listWatcher,
		deltaFIFO:    deltaFIFO,
		logger:       logger,
		stopCh:       make(chan struct{}),
		resyncPeriod: resyncPeriod,
	}
}

func (r *Reflector) Run(ctx context.Context) {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.listAndWatch(ctx)
	}()
}

func (r *Reflector) Stop() {
	close(r.stopCh)
	r.wg.Wait()
}

func (r *Reflector) listAndWatch(ctx context.Context) {
	// 首次列出所有资源
	items, err := r.listWatcher.List(ctx)
	if err != nil {
		r.logger.Error("reflector list failed", zap.String("name", r.name), zap.Error(err))
		return
	}

	if len(items) > 0 {
		r.version = r.listWatcher.GetResourceVersion(items[0])
	} else {
		r.version = ""
	}
	r.logger.Info("reflector initial list completed", zap.String("name", r.name), zap.Int("count", len(items)))

	// 替换 FIFO 中的所有项
	if err := r.deltaFIFO.Replace(items); err != nil {
		r.logger.Error("reflector replace failed", zap.String("name", r.name), zap.Error(err))
	}

	resyncTicker := time.NewTicker(r.resyncPeriod)
	defer resyncTicker.Stop()

	watchTicker := time.NewTicker(5 * time.Second)
	defer watchTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-resyncTicker.C:
			// 定期重新同步
			r.resync(ctx)
		case <-watchTicker.C:
			// 监听变化
			newVersion, items, err := r.listWatcher.Watch(ctx, r.version)
			if err != nil {
				r.logger.Error("reflector watch failed", zap.String("name", r.name), zap.Error(err))
				continue
			}

			if newVersion != r.version && newVersion != "" {
				r.version = newVersion
				if len(items) > 0 {
					if err := r.deltaFIFO.Replace(items); err != nil {
						r.logger.Error("reflector replace failed", zap.String("name", r.name), zap.Error(err))
					}
				}
			}
		}
	}
}

func (r *Reflector) resync(ctx context.Context) {
	items, err := r.listWatcher.List(ctx)
	if err != nil {
		r.logger.Error("reflector resync list failed", zap.String("name", r.name), zap.Error(err))
		return
	}

	if err := r.deltaFIFO.Replace(items); err != nil {
		r.logger.Error("reflector resync replace failed", zap.String("name", r.name), zap.Error(err))
		return
	}

	r.logger.Debug("reflector resync completed", zap.String("name", r.name), zap.Int("count", len(items)))
}

