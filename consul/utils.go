package consul

import (
	"math/rand"
	"strconv"
	"time"

	"github.com/leon-gopher/discovery/logger"
	"github.com/leon-gopher/discovery/registry"
	"github.com/hashicorp/consul/api"
)

func SlidingDuration(d time.Duration) time.Duration {
	sliding := rand.Int63() % (int64(d) / 16)

	return d + time.Duration(sliding)
}

func ServicesCovert(src []*api.ServiceEntry) []*registry.Service {
	entries := make([]*registry.Service, 0, len(src))

	for _, entry := range src {
		weight := int32(DefaultServiceWeight)
		if weightStr, ok := entry.Service.Meta["weight"]; ok {
			weightInt64, err := strconv.ParseInt(weightStr, 10, 64)
			if err != nil {
				logger.Errorf("service(%v) weight parse err: %v", entry.Service.Service, err)
			} else {
				weight = int32(weightInt64)
			}
		}
		if entry.Service == nil {
			continue
		}

		entries = append(entries, &registry.Service{
			ID:     entry.Service.ID,
			Name:   entry.Service.Service,
			IP:     entry.Service.Address,
			Port:   entry.Service.Port,
			Tags:   entry.Service.Tags,
			Meta:   entry.Service.Meta,
			Weight: weight,
		})
	}
	return entries
}

func CatalogServiceCovert(src []*api.CatalogService) []*registry.Service {
	entries := make([]*registry.Service, 0, len(src))
	for _, entry := range src {
		weight := int32(DefaultServiceWeight)
		if weightStr, ok := entry.ServiceMeta["weight"]; ok {
			weightInt64, err := strconv.ParseInt(weightStr, 10, 64)
			if err != nil {
				logger.Errorf("service(%v) weight parse err:%v", entry.ServiceName, err)
			} else {
				weight = int32(weightInt64)
			}
		}

		entries = append(entries, &registry.Service{
			ID:     entry.ServiceID,
			Name:   entry.ServiceName,
			IP:     entry.ServiceAddress,
			Port:   entry.ServicePort,
			Tags:   entry.ServiceTags,
			Meta:   entry.ServiceMeta,
			Weight: weight,
		})
	}
	return entries
}

func CatalogReduceRepeate(entries []*api.CatalogService, passingOnly bool) []*api.CatalogService {
	reduceMap := make(map[string]int)
	newEntries := make([]*api.CatalogService, 0, len(entries))
	for _, entry := range entries {
		if passingOnly && (entry.Checks.AggregatedStatus() != api.HealthPassing) {
			continue
		}

		if idx, ok := reduceMap[entry.ServiceID]; ok {
			if newEntries[idx].ModifyIndex < entry.ModifyIndex {
				newEntries[idx] = entry
			}
			continue
		}
		newEntries = append(newEntries, entry)
		reduceMap[entry.ServiceID] = len(newEntries) - 1
	}

	return newEntries
}

func ReduceRepeate(entries []*api.ServiceEntry) []*api.ServiceEntry {
	//entries 去重
	reduceMap := make(map[string]int)
	newEntries := make([]*api.ServiceEntry, 0, len(entries))

	for _, entry := range entries {
		if idx, ok := reduceMap[entry.Service.ID]; ok {
			//如果已经存在，则判断谁的index更大
			if newEntries[idx].Service.ModifyIndex < entry.Service.ModifyIndex {
				newEntries[idx] = entry
			}
			continue
		}
		newEntries = append(newEntries, entry)
		reduceMap[entry.Service.ID] = len(newEntries) - 1
	}

	return newEntries
}
