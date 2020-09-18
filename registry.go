package discovery

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"math/rand"
	"time"

	"github.com/leon-gopher/discovery/consul"
	"github.com/leon-gopher/discovery/dumper"
	"github.com/leon-gopher/discovery/errors"
	"github.com/leon-gopher/discovery/file"
	"github.com/leon-gopher/discovery/logger"
	"github.com/leon-gopher/discovery/registry"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// Registry wraps both register and discovery interfaces.
type Registry struct {
	lock     sync.Mutex
	opts     *registryOption
	watchers []registry.Watcher
}

// NewRegistry creates a new *Registry with given register or resolver implementation.
func NewRegistry(opts ...RegistryOption) (*Registry, error) {
	o := new(registryOption)
	for _, opt := range opts {
		opt(o)
	}

	r := &Registry{
		opts: o,
	}

	// apply watchers
	for _, adapter := range r.opts.discoveries {
		adapter.Watch(registry.WatchFunc(r.watchServices))
	}

	return r, nil
}

// NewRegistryWithConsul creates a new *Registry with consul adapter as default discovery and register. It
// creates a dump with filepath.Join(os.TempDir(), "discovery-local".
func NewRegistryWithConsul(addr string, opts ...consul.ConsulOption) (*Registry, error) {
	localDir := filepath.Join(os.TempDir(), DefaultTempDir)

	err := os.MkdirAll(localDir, 0755)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	return NewRegistryWithConsulAndFile(addr, localDir, opts...)
}

// NewRegistryWithConsulAndFile creates a new *Registry with consul adapter as default discovery and register, and uses
// fileDir as dump for local discovery.
// NOTE: It the customer' ability to ensure the fileDir is existed and can write!
func NewRegistryWithConsulAndFile(consulAddr, localDir string, opts ...consul.ConsulOption) (*Registry, error) {
	dp, err := dumper.New(dumper.WithLocalDir(localDir), dumper.WithFormat(dumper.FormatDiscovery))
	if err != nil {
		return nil, err
	}

	opts = append(opts, consul.WithDumper(dp))

	adapter, err := consul.New(consulAddr, opts...)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	fallbackAdapter := file.New(dp)

	return NewRegistry(WithDiscoveries(adapter, fallbackAdapter), WithRegisters(adapter))
}

// LookupServices tries to resolve services of the name from registered discovery. It will retries among all discoveries
// when the result is unexpected.
func (r *Registry) LookupServices(name string, opts ...registry.DiscoveryOption) ([]*registry.Service, error) {
	o := registry.NewCommonDiscoveryOption(opts...)
	key := registry.NewServiceKey(name, o.Tags, o.DC)

	var currentServices []*registry.Service
	var currentErr error
	for _, disc := range r.opts.discoveries {
		newServices, err := disc.GetServices(name, opts...)
		currentErr = err

		if len(newServices) > len(currentServices) {
			currentServices = newServices
		}

		if err != nil {
			logger.Errorf("%T.LookupServices(%s, %v): %+v", disc, name, opts, err)
			if r.opts.failType == FailFast {
				return nil, errors.Wrap(err)
			}
			continue
		}

		if r.isFallback(key, newServices, err) {
			disc.Notify(registry.EventDegrade)
			continue
		}

		disc.Notify(registry.EventRecover)
		return currentServices, nil
	}
	if len(currentServices) > 0 {
		return currentServices, nil
	}

	return currentServices, currentErr
}

// Register tries to register service with all registrators and returns wrapped service registrator which use for deregister service
// by one call.
func (r *Registry) Register(service *registry.Service, opts ...registry.RegistratorOption) (registrator ServiceRegister, err error) {
	for _, register := range r.opts.registrators {
		err = register.Register(service, opts...)
		if err != nil {
			logger.Errorf("%T.Register(%#v, %v): %+v", register, service, opts, err)

			if r.opts.failType == FailFast {
				return nil, errors.Wrap(err)
			}

			continue
		}
	}

	return &ServiceRegistrator{
		service:      service,
		registrators: r.opts.registrators,
	}, err
}

func (r *Registry) WithWatcher(w registry.Watcher) {
	r.lock.Lock()
	defer r.lock.Unlock()

	r.watchers = append(r.watchers, w)
}

func (r *Registry) WithWatcherFunc(service registry.ServiceKey, w registry.Watcher) {
	r.WithWatcher(registry.WatchFunc(func(key registry.ServiceKey, services []*registry.Service) {
		if key != service {
			return
		}

		w.Watch(key, services)
	}))
}

func (r *Registry) watchServices(key registry.ServiceKey, services []*registry.Service) {
	r.lock.Lock()
	defer r.lock.Unlock()

	var err error

	if r.isFallback(key, services, nil) {
		logger.Errorf("Registry fallback triggered, service: %s, total: %v", key.ToString(), len(services))

		services, err = r.LookupServices(key.Name, registry.WithDC(key.DC), registry.WithTags(strings.Split(key.Tags, ":")))
		if err != nil {
			logger.Errorf("registry.LookupServices(%s): fallback with %+v", key.ToString(), err)
			return
		}
	}

	for _, w := range r.watchers {
		w.Watch(key, services)
	}
}

func (r *Registry) isFallback(key registry.ServiceKey, services []*registry.Service, err error) bool {
	// fallback with not found error
	if err != nil && errors.Is(err, errors.ErrNotFound) {
		return true
	}

	// fallback with empty service
	if len(services) <= 0 {
		return true
	}

	// fallback with bootstrap strategy
	if r.opts.bootstrap[key] > len(services) {
		return true
	}

	return false
}
