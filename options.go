package discovery

import (
	"github.com/leon-gopher/discovery/registry"
)

type RegistryOption func(o *registryOption)

type registryOption struct {
	registrators []registry.Registrator
	discoveries  []registry.Discovery
	bootstrap    map[registry.ServiceKey]int
	failType     FailType
}

func WithFailType(t FailType) RegistryOption {
	return func(o *registryOption) {
		o.failType = t
	}
}

func WithRegisters(regs ...registry.Registrator) RegistryOption {
	return func(o *registryOption) {
		o.registrators = append(o.registrators, regs...)
	}
}

func WithDiscoveries(disc ...registry.Discovery) RegistryOption {
	return func(o *registryOption) {
		o.discoveries = append(o.discoveries, disc...)
	}
}

func WithBootstrap(bootstrap map[registry.ServiceKey]int) RegistryOption {
	return func(o *registryOption) {
		for key, expected := range bootstrap {
			WithBootstrapByKey(key, expected)
		}
	}
}

func WithBootstrapByKey(key registry.ServiceKey, expected int) RegistryOption {
	return func(o *registryOption) {
		if o.bootstrap == nil {
			o.bootstrap = make(map[registry.ServiceKey]int)
		}

		o.bootstrap[key] = expected
	}
}

func WithBootstrapByName(name string, expected int) RegistryOption {
	return WithBootstrapByKey(registry.NewServiceKey(name, nil, ""), expected)
}
