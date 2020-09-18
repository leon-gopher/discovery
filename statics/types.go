package statics

import "github.com/leon-gopher/discovery/registry"

type Loader interface {
	Load(registry.ServiceKey) ([]*registry.Service, error)
}
