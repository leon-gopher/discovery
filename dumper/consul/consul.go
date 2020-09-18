package consul

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/hashicorp/consul/api"

	"github.com/leon-gopher/discovery/dumper/file"
	"github.com/leon-gopher/discovery/errors"
	"github.com/leon-gopher/discovery/registry"
)

type Dumper struct {
	*file.Dumper
}

func New(root string) *Dumper {
	fd := file.New(root)

	return &Dumper{
		Dumper: fd,
	}
}

// Load tries to parse services for the key from local cached file.
// NOTE: it parses data dumped from http://consul/v1/health/service/<service> api by overwriting file.Dumper.Load implementation.
func (dp *Dumper) Load(key registry.ServiceKey) ([]*registry.Service, error) {
	filename := dp.Filename(key)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrap(errors.ErrNotFound)
		}

		return nil, err
	}

	// try to parse consul entries
	var entries []*api.ServiceEntry

	err = json.Unmarshal(data, &entries)
	if err != nil {
		return nil, err
	}

	// build services for registry
	services := make([]*registry.Service, len(entries))
	for i, entry := range entries {
		if entry.Service == nil {
			continue
		}

		services[i] = &registry.Service{
			ID:   entry.Service.ID,
			Name: key.Name,
			IP:   entry.Service.Address,
			Port: entry.Service.Port,
			Tags: entry.Service.Tags,
			Meta: entry.Service.Meta,
		}
		if entry.Service.Weights.Passing > 0 {
			services[i].Weight = int32(entry.Service.Weights.Passing)
		}
	}

	if len(services) == 0 {
		return nil, errors.Wrap(errors.ErrNotFound)
	}

	return services, nil
}
