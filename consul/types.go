package consul

import (
	"github.com/leon-gopher/discovery/registry"
	"github.com/hashicorp/consul/api"
)

type watchChan struct {
	dc      string
	name    string
	tags    []string
	index   uint64
	entries []*api.ServiceEntry
}

type actorChan struct {
	dc   string
	name string
	tags []string
	out  chan *actorServices
}

type actorServices struct {
	services []*registry.Service
	err      error
}
