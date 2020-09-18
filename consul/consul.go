package consul

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/consul/api"
	"github.com/leon-gopher/discovery/errors"
	"github.com/leon-gopher/discovery/logger"
	"github.com/leon-gopher/discovery/registry"
	"golang.org/x/sync/singleflight"
)

type adapter struct {
	client       *api.Client
	serviceList  *registry.ServiceList
	singleflight *singleflight.Group

	watchChans chan *watchChan
	actorChans chan *actorChan
	stopChan   chan bool

	uri     *url.URL
	opts    *option
	dump    *Dump
	watches sync.Map
	watcher registry.Watcher

	status int32
}

func New(addr string, opts ...ConsulOption) (*adapter, error) {
	uri, err := url.Parse(addr)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	//默认设置
	o := &option{
		stale:             true,
		passingOnly:       true,
		threshold:         DefaultDegradeThreshold,
		watchWaitTime:     DefaultWatchWaitTime,
		watchDumpInterval: DefaultWatchDumpInterval,
		calmInterval:      DefaultCalmInterval,
	}
	for _, opt := range opts {
		opt(o)
	}

	cfg := api.DefaultConfig()
	cfg.Address = uri.Host
	cfg.Scheme = uri.Scheme

	client, err := api.NewClient(cfg)
	if err != nil {
		return nil, errors.Wrap(err)
	}

	consul := &adapter{
		client:       client,
		serviceList:  registry.NewServiceList(),
		actorChans:   make(chan *actorChan, 10),
		watchChans:   make(chan *watchChan, 10),
		stopChan:     make(chan bool),
		singleflight: &singleflight.Group{},
		uri:          uri,
		opts:         o,
	}
	if o.dumper != nil {
		consul.dump = newDump(o.watchDumpInterval, o.dumper)

		go consul.dump.loop()
	}

	go consul.loop()

	return consul, nil
}

func (ca *adapter) Notify(event registry.Event) {
	status := int32(0)
	switch event {
	case registry.EventDegrade:
		status = 1

	case registry.EventRecover:
		status = 0
	}

	if ca.status == status {
		return
	}

	ok := atomic.CompareAndSwapInt32(&ca.status, ca.status, status)
	if ok {
		if ca.dump != nil {
			ca.dump.SetDisable(status == 1)
		}
	}
}

func (ca *adapter) Register(srv *registry.Service, opts ...registry.RegistratorOption) error {
	o := new(registry.CommonRegistratorOption)
	for _, opt := range opts {
		switch opt := opt.(type) {
		case registry.RegisterOpt:
			opt(o)
		default:
			return errors.Wrap(errors.ErrArgument)
		}
	}

	if srv.Meta == nil {
		srv.Meta = make(map[string]string)
	}

	//Default metadata
	for k, v := range DefaultServiceMeta {
		if _, ok := srv.Meta[k]; !ok {
			srv.Meta[k] = v
		}
	}

	//Option metadata
	for k, v := range o.Metadata {
		srv.Meta[k] = v
	}

	//metadata contains weight
	if _, ok := srv.Meta["weight"]; !ok {
		if srv.Weight <= 0 {
			srv.Weight = DefaultServiceWeight
		}

		srv.Meta["weight"] = strconv.FormatInt(int64(srv.Weight), 10)
	}

	service := &api.AgentServiceRegistration{
		ID:      srv.ServiceID(),
		Name:    srv.Name,
		Address: srv.ServiceIP(),
		Port:    srv.Port,
		Tags:    srv.Tags,
		Meta:    srv.Meta,
	}

	if srv.Weight > 0 {
		service.Weights = &api.AgentWeights{
			Passing: int(srv.Weight),
			Warning: int(srv.Weight),
		}
	}

	// build service check, default to tcp check
	for _, check := range o.Checks {
		ccheck, err := ca.BuildHealthCheck(srv.Name, srv.Addr(), check)
		if err != nil {
			return errors.Wrap(err)
		}

		service.Checks = append(service.Checks, ccheck)
	}

	if len(service.Checks) <= 0 {
		//没有注入check，给个默认的check
		service.Checks = append(service.Checks, &api.AgentServiceCheck{
			Name:                           srv.Name,
			TCP:                            srv.Addr(),
			Status:                         api.HealthPassing,
			Interval:                       DefaultServiceCheckInterval,
			DeregisterCriticalServiceAfter: DefaultServiceDeregisterCriticalAfter,
		})

	}

	var err error
	for i := 0; i < DefaultRetryTimes; i++ {
		err = ca.client.Agent().ServiceRegister(service)
		if err == nil {
			break
		}
		logger.Infof("%v times consul.Register(%s): %v", i+1, service.ID, err)

		time.Sleep(1 * time.Second)
	}

	if err != nil {
		return errors.Wrap(err)
	}

	logger.Infof("consul.Register(%s): OK!", service.ID)

	return nil
}

func (ca *adapter) Deregister(srv *registry.Service, opts ...registry.RegistratorOption) error {
	var err error
	for i := 0; i < DefaultRetryTimes; i++ {
		err = ca.client.Agent().ServiceDeregister(srv.ServiceID())
		if err == nil {
			break
		}

		logger.Infof("consul.Deregister(%s): %v", srv.ServiceID(), err)
		time.Sleep(1 * time.Second)
	}
	if err != nil {
		logger.Infof("consul.Deregister(%s): %v", srv.ServiceID(), err)
	} else {
		logger.Infof("consul.Deregister(%s): Done!", srv.ServiceID())
	}
	return err
}

//-----------------------------discovery---------------------------
func (ca *adapter) BuildHealthCheck(name, addr string, check *registry.HealthCheck) (*api.AgentServiceCheck, error) {
	if check == nil {
		return nil, nil
	}

	healthStatus := api.HealthPassing
	if check.Status == registry.HealthCritical {
		healthStatus = api.HealthCritical
	}

	interval := check.Interval
	if check.Interval.Seconds() < 1 {
		interval = time.Second
	}

	health := &api.AgentServiceCheck{
		Name:                           check.Name,
		Status:                         healthStatus,
		Interval:                       interval.String(),
		DeregisterCriticalServiceAfter: DefaultServiceDeregisterCriticalAfter,
	}
	switch check.Type {
	case registry.HealthTypeHTTP:
		health.HTTP = check.URI
		health.Method = check.Method
		health.Header = check.Header

	case registry.HealthTypeTCP:
		health.TCP = check.URI

	default:
		return nil, errors.Wrap(errors.ErrArgument)
	}

	return health, nil
}

func (ca *adapter) GetServices(name string, opts ...registry.DiscoveryOption) ([]*registry.Service, error) {
	o := registry.NewCommonDiscoveryOption(opts...)
	key := registry.NewServiceKey(name, o.Tags, o.DC)

	services, err := ca.serviceList.GetServices(key)
	if err == nil {
		return services, nil
	}

	entries, err, _ := ca.singleflight.Do(key.ToString(), func() (interface{}, error) {
		var services []*registry.Service
		var err error
		if ca.opts.firstFetchUseCatalog {
			services, err = ca.CatalogServices(name, o.DC, o.Tags)
		} else {
			services, err = ca.ServiceMultipleTags(name, o.DC, o.Tags)
		}

		if err != nil {
			return nil, err
		}

		ca.serviceList.Set(key, services)

		//不存在,执行一个启动流程
		ca.actorChans <- &actorChan{
			dc:   o.DC,
			name: name,
			tags: o.Tags,
		}

		return services, nil
	})

	if services, ok := entries.([]*registry.Service); ok {
		return services, err
	}
	return nil, err
}

func (ca *adapter) Watch(w registry.Watcher) {
	ca.watcher = w
}

func (ca *adapter) loop() {
	for {
		select {
		case action := <-ca.actorChans:
			ca.startWatch(action.name, action.tags, action.dc)
		case service := <-ca.watchChans:
			ca.addService(service, false)

		case <-ca.stopChan:
			ca.watches.Range(func(_, value interface{}) bool {
				if w, ok := value.(*Watch); ok {
					logger.Infof("%T.Stop(): OK!", w)

					w.Stop()
				}

				return true
			})
		}
	}
}

func (ca *adapter) addService(service *watchChan, overwrite bool) {
	key := registry.NewServiceKey(service.name, service.tags, service.dc)
	if len(service.entries) <= 0 {
		if !overwrite {
			return
		}
		ca.serviceList.Set(key, make([]*registry.Service, 0))
		return
	}

	entries := ServicesCovert(service.entries)

	ca.serviceList.Set(key, entries)
	if ca.watcher != nil {
		ca.watcher.Watch(key, entries)
	}

	if ca.dump != nil {
		ca.dump.dump(key, entries)
	}
}

func (ca *adapter) CatalogServices(name string, dc string, tags []string) ([]*registry.Service, error) {
	apiOpts := &api.QueryOptions{
		Datacenter: dc,
		AllowStale: ca.opts.stale,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	apiOpts = apiOpts.WithContext(ctx)

	services, _, err := ca.client.Catalog().ServiceMultipleTags(name, tags, apiOpts)
	if err != nil {
		return nil, errors.Wrap(fmt.Errorf("consul.Catalog().ServiceMultipleTags(%s, %v, %v, %+v): %v", name, tags, ca.opts.passingOnly, apiOpts, err))
	}

	services = CatalogReduceRepeate(services, ca.opts.passingOnly)

	return CatalogServiceCovert(services), nil
}

func (ca *adapter) ServiceMultipleTags(name string, dc string, tags []string) ([]*registry.Service, error) {
	apiOpts := &api.QueryOptions{
		Datacenter: dc,
		AllowStale: ca.opts.stale,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	apiOpts = apiOpts.WithContext(ctx)

	services, _, err := ca.client.Health().ServiceMultipleTags(name, tags, ca.opts.passingOnly, apiOpts)
	if err != nil {
		return nil, errors.Wrap(fmt.Errorf("consul.Health().ServiceMultipleTags(%s, %v, %v, %+v): %v", name, tags, ca.opts.passingOnly, apiOpts, err))
	}
	services = ReduceRepeate(services)
	return ServicesCovert(services), nil
}

func (ca *adapter) Stop() {
	ca.stopChan <- true
}

func (ca *adapter) startWatch(name string, tags []string, dc string) {
	key := registry.NewServiceKey(name, tags, dc)
	if _, ok := ca.watches.Load(key); ok {
		return
	}

	watch := &Watch{
		adapter:    ca,
		dc:         dc,
		name:       name,
		tags:       tags,
		watchChans: ca.watchChans,
	}

	//使用降级策略
	if ca.opts.enableDegrade() {
		watch.degrades = []Degrader{newPassingOnlyDegrade(watch)}
	}

	ca.watches.Store(key, watch)

	go watch.Watch()
}
