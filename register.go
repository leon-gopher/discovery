package discovery

import (
	"github.com/leon-gopher/discovery/errors"
	"github.com/leon-gopher/discovery/registry"
)

type ServiceRegister interface {
	Deregister() error
}

type ServiceRegistrator struct {
	service      *registry.Service
	registrators []registry.Registrator
}

func (sr *ServiceRegistrator) Deregister() (err error) {
	for _, register := range sr.registrators {
		err = register.Deregister(sr.service)
		if err != nil {
			err = errors.Errorf("%T.Deregister(%#v): %+v", register, sr.service, err)
		}
	}

	return
}
