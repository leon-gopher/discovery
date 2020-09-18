package file

import (
	"sync"

	"github.com/leon-gopher/discovery/dumper"
	"github.com/leon-gopher/discovery/errors"
	"github.com/leon-gopher/discovery/registry"
	"github.com/leon-gopher/discovery/statics"
)

// File implements registry.Discovery interface with local file.
type File struct {
	mux    sync.Mutex
	opts   *option
	loader Loader
	store  sync.Map
}

func New(dumper dumper.Dumper, opts ...Option) *File {
	return NewWithLoader(dumper, opts...)
}

func NewWithLoader(loader Loader, opts ...Option) *File {
	o := new(option)
	for _, opt := range opts {
		opt(o)
	}

	return &File{
		opts:   o,
		loader: loader,
	}
}

func (f *File) GetServices(name string, opts ...registry.DiscoveryOption) ([]*registry.Service, error) {
	o := registry.NewCommonDiscoveryOption(opts...)

	key := registry.NewServiceKey(name, o.Tags, o.DC)

	iface, ok := f.store.Load(key)
	if !ok {
		err := f.Load(key)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		iface, ok = f.store.Load(key)
		if !ok {
			return nil, errors.Wrap(errors.ErrNotFound)
		}
	}

	adapter, ok := iface.(registry.Discovery)
	if ok {
		return adapter.GetServices(name, opts...)
	}

	return nil, errors.Wrap(errors.ErrNotFound)
}

func (f *File) Watch(w registry.Watcher) {}

func (f *File) Notify(event registry.Event) {}

func (f *File) Load(key registry.ServiceKey) error {
	f.mux.Lock()
	defer f.mux.Unlock()

	f.store.Store(key, statics.NewWithLoader(f.loader))

	return nil
}
