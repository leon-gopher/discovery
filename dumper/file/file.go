package file

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/leon-gopher/discovery/errors"
	"github.com/leon-gopher/discovery/logger"
	"github.com/leon-gopher/discovery/registry"
)

type Dumper struct {
	dir  string
	once sync.Once
}

func New(dir string) *Dumper {
	return &Dumper{
		dir: dir,
	}
}

// Filename returns filename of cached file for the key given.
func (dp *Dumper) Filename(key registry.ServiceKey) string {
	dp.once.Do(func() {
		err := os.MkdirAll(dp.dir, 0755)
		if err != nil {
			logger.Errorf("os.MkdirAll(%s): %+v", dp.dir, err)
		}
	})

	return filepath.Join(dp.dir, key.ToString())
}

// LastModify tries to resolve ctime of cached file for the key given.
func (dp *Dumper) LastModify(key registry.ServiceKey) (time.Time, error) {
	filename := dp.Filename(key)

	info, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return time.Time{}, errors.ErrNotFound
		}

		// avoid flush filename by returning now forever
		return time.Now(), err
	}

	return info.ModTime(), nil
}

// Store tries to persist services for the key within local cached file.
func (dp *Dumper) Store(key registry.ServiceKey, services interface{}) error {
	data, err := json.Marshal(services)
	if err != nil {
		return err
	}

	//may slow and safe write file
	filename := dp.Filename(key)
	return WriteAtomicWithPerms(filename, data, 0666)
}

// Load tries to parse services for the key from local cached file.
func (dp *Dumper) Load(key registry.ServiceKey) ([]*registry.Service, error) {
	filename := dp.Filename(key)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Wrap(errors.ErrNotFound)
		}

		return nil, err
	}

	var services []*registry.Service

	err = json.Unmarshal(data, &services)
	if err != nil {
		return nil, err
	}

	return services, nil
}
