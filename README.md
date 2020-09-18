# discovery 服务注册与发现 SDK.v2


主要用于标准方式部署的 consul，即每个服务机器都需要部署 consul agent 服务！

# 使用方式

## 使用 SDK 进行注册

```go
package main

import (
	"log"
	"sync"

	"github.com/leon-gopher/discovery"
	"github.com/leon-gopher/discovery/registry"
)

var (
	// 建议全局初始化一个 registry 对象
	singleRegistry     *discovery.Registry
	singleRegistryOnce sync.Once
)

func main() {
	singleRegistryOnce.Do(func() {
		var err error

		singleRegistry, err = discovery.NewRegistryWithConsul("http://localhost:4500")
		if err != nil {
			panic(err)
		}
	})

	//注册服务
	service, err := singleRegistry.Register(&registry.Service{
		//服务名: 建议ops项目名，不能使用下换线且任何非url safe的字符
		Name: "test-redis",
		//服务注册ip地址
		IP: "10.104.32.79",
		//服务端口
		Port: 9999,
		//tag
		Tags: []string{"test"},
		//元数据
		Meta: map[string]string{
			"hostname": "ffffff",
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	//注销服务
	service.Deregister()
}
```


## 使用 `*http.Client` 进行服务发现

```go
package main

import (
	"log"
	gohttp "net/http"
	"sync"
	"time"

	"github.com/leon-gopher/discovery/http"
)

var (
	// 建议全局初始化一个 discovery http client 对象
	HTTPClient         *gohttp.Client
	singleRegistryOnce sync.Once
)

func main() {
	singleRegistryOnce.Do(func() {
		var err error

		cfg := &http.Config{
			ConsulAddr:  "http://localhost:8500",
			Timeout:     5 * time.Second,
			LoadBalance: http.LBRoundRobin,
		}

		HTTPClient, err = http.NewClient(cfg)
		if err != nil {
			panic(err)
		}
	})

	req, err := gohttp.NewRequest("GET", "http://backend-http-service/ping", nil)
	if err != nil {
		log.Fatalln(err)
	}

	resp, err := HTTPClient.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	// deal with resp
	_ = resp
}
```


## 使用 gRPC 进行服务发现

```go
package main

import (
	"fmt"
	"sync"

	"github.com/leon-gopher/discovery"
	resolver "github.com/leon-gopher/discovery/grpc"

	"google.golang.org/grpc"
	"google.golang.org/grpc/balancer/roundrobin"
)

var (
	// 建议全局初始化一个 registry 对象
	singleRegistry     *discovery.Registry
	singleRegistryOnce sync.Once
)

func init() {
	singleRegistryOnce.Do(func() {
		var err error

		singleRegistry, err = discovery.NewRegistryWithConsul("http://localhost:8500")
		if err != nil {
			panic(err)
		}
	})

	// 注册 grpc reslover
	resolver.Register(singleRegistry)

}

func main() {
	cc, err := grpc.Dial("dis:///backend-grpc-service", grpc.WithBalancerName(roundrobin.Name), grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return
	}

	fmt.Println(cc.GetState())
}
```

