package consul

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/leon-gopher/discovery/errors"
	"github.com/leon-gopher/discovery/logger"
)

type Watch struct {
	adapter    *adapter
	dc         string
	name       string
	tags       []string
	plan       *watch.Plan
	watchChans chan *watchChan
	degrades   []Degrader
	inited     bool

	// internal
	lastIndex uint64
	rolling   *rollingWindow
	delay     time.Duration
}

func (w *Watch) option() *option {
	return w.adapter.opts
}

func (w *Watch) isDebug() bool {
	return w.option().debug
}

func (w *Watch) consul() *api.Client {
	return w.adapter.client
}

func (w *Watch) Watch() error {
	plan, err := watch.Parse(map[string]interface{}{
		"type":    "service",
		"service": w.name,
		"stale":   w.option().stale,
		"tag":     w.tags,
	})
	if err != nil {
		return errors.Wrap(err)
	}

	plan.Datacenter = w.dc
	plan.Handler = w.Handler
	plan.Watcher = w.ServiceWatch()

	w.plan = plan
	w.rolling = NewRollingWindow(DefaultWatchRollingWindowSize)

	err = plan.RunWithClientAndLogger(w.consul(), log.New(os.Stderr, "consul", 0))
	if err != nil {
		logger.Errorf("start(%v,%v,%v) watch failed:%v", w.name, w.dc, w.tags, err)
		return errors.Wrap(err)
	}
	return nil
}

func (w *Watch) Handler(idx uint64, services interface{}) {
	entries, ok := services.([]*api.ServiceEntry)
	if !ok {
		return
	}

	entries = ReduceRepeate(entries)

	nentries, err := w.CheckDegrade(entries)
	if err != nil {
		if !errors.Is(err, errors.ErrDegradePass) || w.inited {
			return
		}
	} else {
		entries = nentries
	}

	//如果是第一次同步数据，就算数据不满足阀值的要求, 也存放在内存中
	if !w.inited {
		w.inited = true
	}

	wc := &watchChan{
		dc:      w.dc,
		name:    w.name,
		tags:    w.tags,
		index:   idx,
		entries: entries,
	}
	if w.isDebug() {
		logger.Debugf("watch.Handler(%s, %d): services: %v", w.name, idx, len(entries))
	}

	w.watchChans <- wc
}

func (w *Watch) CheckDegrade(entries []*api.ServiceEntry) ([]*api.ServiceEntry, error) {
	var newEntries []*api.ServiceEntry
	var err error

	for _, degrade := range w.degrades {
		newEntries, err = degrade.CheckStatus(entries)
		if err != nil {
			if errors.Is(err, errors.ErrDegradePass) {
				continue
			}
			logger.Infof("watch.Handler(%s): degrader: %T, services: %v", w.name, degrade, len(entries))
		}
		return newEntries, nil
	}

	return newEntries, err
}

func (w *Watch) Stop() {
	w.plan.Stop()
}

func (w *Watch) ServiceWatch() watch.WatcherFunc {
	return func(p *watch.Plan) (watch.BlockingParamVal, interface{}, error) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		opts := &api.QueryOptions{
			Datacenter: w.dc,
			AllowStale: w.option().stale,
			WaitIndex:  w.lastIndex,
			UseCache:   w.option().agentCache,
			WaitTime:   w.option().watchWaitTime,
		}
		opts = opts.WithContext(ctx)

		nodes, meta, err := w.consul().Health().ServiceMultipleTags(w.name, w.tags, false, opts)
		if err != nil {
			return nil, nil, err
		}

		if w.isDebug() {
			logger.Debugf("watch.WatcherFunc(%s): prev index: %v, last index: %v, nodes: %d",
				w.name, w.lastIndex, meta.LastIndex, len(nodes))
		}

		if w.lastIndex == 0 {
			logger.Infof("watch.WatcherFunc(%s): prev index: %v, last index: %v, nodes: %d",
				w.name, w.lastIndex, meta.LastIndex, len(nodes))

			w.Handler(meta.LastIndex, nodes)
		}

		w.backoff(meta.LastIndex)

		//update lastIndex
		w.lastIndex = meta.LastIndex

		return watch.WaitIndexVal(w.lastIndex), nodes, err
	}
}

func (w *Watch) backoff(curIndex uint64) {
	if curIndex == w.lastIndex {
		w.delay = time.Duration(0)

		return
	}

	if w.rolling == nil {
		w.rolling = NewRollingWindow(DefaultWatchRollingWindowSize)
	}

	w.rolling.incr(time.Now())

	if w.rolling.isMatch(1, 2) {
		w.delay = 1 * time.Second
	}

	if w.rolling.isMatch(3, 3) {
		w.delay = 2 * time.Second
	}

	if w.rolling.isMatch(6, 4) {
		w.delay = 3 * time.Second
	}

	if w.rolling.isMatch(10, 5) {
		w.delay = 5 * time.Second
	}

	if w.delay.Seconds() > 0 {
		if w.isDebug() {
			logger.Debugf("watch.backoff(%s): triggered with delay: %v", w.name, w.delay)
		}

		// sleep with sliding duration
		time.Sleep(SlidingDuration(w.delay))
	}
}
