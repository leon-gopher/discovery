package dumper

import (
	"time"

	"github.com/leon-gopher/discovery/registry"
)

// Dumper interface.
type Dumper interface {
	LastModify(registry.ServiceKey) (time.Time, error)
	Store(registry.ServiceKey, interface{}) error
	Load(registry.ServiceKey) ([]*registry.Service, error)
}
