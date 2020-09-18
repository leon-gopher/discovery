package registry

import (
	"sync"

	"github.com/leon-gopher/discovery/errors"
)

type ServiceList struct {
	services sync.Map
}

func NewServiceList() *ServiceList {
	return &ServiceList{}
}

func (s *ServiceList) Set(key ServiceKey, services []*Service) {
	s.services.Store(key, services)
}

func (s *ServiceList) GetServices(key ServiceKey) ([]*Service, error) {
	val, ok := s.services.Load(key)
	if !ok {
		return nil, errors.Wrap(errors.ErrNotFound)
	}
	if services, ok := val.([]*Service); ok {
		return services, nil
	}

	return nil, errors.Wrap(errors.ErrNotFound)
}
