package registry

//使用方保证线程安全
type Watcher interface {
	Watch(ServiceKey, []*Service)
}

type WatchFunc func(ServiceKey, []*Service)

func (f WatchFunc) Watch(serviceKey ServiceKey, services []*Service) {
	f(serviceKey, services)
}
