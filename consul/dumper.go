package consul

import (
	"time"

	"github.com/leon-gopher/discovery/dumper"

	"github.com/leon-gopher/discovery/errors"
	"github.com/leon-gopher/discovery/logger"
	"github.com/leon-gopher/discovery/registry"
)

type Dump struct {
	dumper   dumper.Dumper
	dumpC    chan *dumpService
	disable  bool
	disableC chan bool
	interval time.Duration
	last     map[registry.ServiceKey]time.Time
}

func newDump(interval time.Duration, dumper dumper.Dumper) *Dump {
	return &Dump{
		dumper:   dumper,
		dumpC:    make(chan *dumpService, 1),
		disableC: make(chan bool, 1),
		last:     make(map[registry.ServiceKey]time.Time),
		interval: interval,
	}
}

type dumpService struct {
	key      registry.ServiceKey
	services []*registry.Service
}

func (d *Dump) dump(key registry.ServiceKey, services []*registry.Service) {
	d.dumpC <- &dumpService{
		key:      key,
		services: services,
	}
}

func (d *Dump) loop() {
	for {
		select {
		case job := <-d.dumpC:
			if d.disable {
				continue
			}

			if d.last[job.key].Add(d.interval).After(time.Now()) {
				continue
			}

			logger.Infof("%T.Store(%v): services: %v, last: %v, next: %v", d.dumper, job.key, len(job.services), d.last[job.key], d.last[job.key].Add(d.interval))

			lastModify, err := d.dumper.LastModify(job.key)
			if err != nil {
				switch {
				case !errors.Is(err, errors.ErrNotFound):
					logger.Errorf("%T.LastModify(%s): %v", d.dumper, job.key, err)
					continue

				default:
					logger.Infof("%T.LastModify(%s): %v", d.dumper, job.key, err)

					err = nil
				}
			}

			if lastModify.Add(d.interval).After(time.Now()) {
				d.last[job.key] = lastModify
				continue
			}

			err = d.dumper.Store(job.key, job.services)
			if err != nil {
				logger.Errorf("%T.Store(%s): services: %d, error: %v", d.dumper, job.key, len(job.services), err)
			} else {
				logger.Infof("%T.Store(%s): services: %v, OK!", d.dumper, job.key, len(job.services))

				d.last[job.key] = time.Now()
			}

		case disable := <-d.disableC:
			if disable {
				logger.Infof("dump.%T(): Enabled!", d.dumper)
			} else {
				logger.Infof("dump.%T(): Disabled!", d.dumper)
			}

			d.disable = disable
		}
	}
}

func (d *Dump) SetDisable(disable bool) {
	d.disableC <- disable
}
