package dumper

import (
	"github.com/leon-gopher/discovery/dumper/consul"
	"github.com/leon-gopher/discovery/dumper/file"
	"github.com/leon-gopher/discovery/errors"
)

func New(opts ...Option) (Dumper, error) {
	o := new(options)
	for _, opt := range opts {
		opt(o)
	}

	if !o.format.IsValid() {
		return nil, errors.ErrInvalidDumper
	}

	var dp Dumper
	switch o.format {
	case FormatConsul:
		dp = consul.New(o.root)

	case FormatDiscovery:
		dp = file.New(o.root)
	}

	return dp, nil
}
