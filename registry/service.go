package registry

import (
	"os"
	"strconv"
	"sync"

	"github.com/leon-gopher/discovery/logger"
	"github.com/hashicorp/go-sockaddr"
	"github.com/hashicorp/go-sockaddr/template"
)

const (
	ServiceDefaultHostname = "default"
)

type Service struct {
	ID         string            `discovery:"可选,服务id"`
	Name       string            `discovery:"必填,服务名"`
	IP         string            `discovery:"可选,默认拿en0的地址"`
	IPTemplate string            `discovery:"可选,可以用于指定特殊网卡" json:"IPTemplate,omitempty"`
	Port       int               `discovery:"必填,端口"`
	Weight     int32             `discovery:"可选,权重"`
	Tags       []string          `discovery:"可选,标签"`
	Meta       map[string]string `discovery:"可选,自定义元数据"`

	once sync.Once
}

func (s *Service) FillWithDefaults() {
	s.once.Do(func() {
		// adjust service ip if empty
		if len(s.IP) == 0 {
			// try ip template if provided
			if len(s.IPTemplate) > 0 {
				ipv4, err := template.Parse(s.IPTemplate)
				if err == nil {
					s.IP = ipv4
				} else {
					logger.Errorf("template.Parse(%s): %v", s.IPTemplate, err)
				}
			}

			if len(s.IP) == 0 {
				ipv4, err := sockaddr.GetPrivateIP()
				if err == nil {
					s.IP = ipv4
				} else {
					logger.Errorf("sockaddr.GetPrivateIP(): %v", err)
				}
			}
		}

		// adjust service id
		if len(s.ID) == 0 {
			hostname, err := os.Hostname()
			if err != nil {
				hostname = ServiceDefaultHostname
			}
			s.ID = s.Name + "~" + s.IP + "~" + hostname
		}

		// avoid panic with write to nil map
		if s.Meta == nil {
			s.Meta = make(map[string]string)
		}
	})
}

func (s *Service) Addr() string {
	s.FillWithDefaults()

	return s.IP + ":" + strconv.FormatInt(int64(s.Port), 10)
}

func (s *Service) ServiceID() string {
	s.FillWithDefaults()

	return s.ID
}

func (s *Service) ServiceIP() string {
	s.FillWithDefaults()

	return s.IP
}
