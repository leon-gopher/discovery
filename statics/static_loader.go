package statics

import (
	"github.com/leon-gopher/discovery/errors"
	"github.com/leon-gopher/discovery/registry"
)

type staticLoader struct {
	store map[registry.ServiceKey][]*registry.Service
}

func NewStaticLoader(list map[registry.ServiceKey][]*registry.Service) Loader {
	if list == nil {
		list = make(map[registry.ServiceKey][]*registry.Service)
	}

	return &staticLoader{
		store: list,
	}
}

func (sl *staticLoader) Load(key registry.ServiceKey) ([]*registry.Service, error) {
	services := sl.store[key]
	if len(services) <= 0 {
		return nil, errors.ErrNotFound
	}

	return services, nil
}
