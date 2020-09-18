package consul

import (
	"time"

	"sync/atomic"

	"github.com/hashicorp/consul/api"
	"github.com/leon-gopher/discovery/errors"
	"github.com/leon-gopher/discovery/logger"
)

//每个降级都在 watch 的 goroutine 里执行
type Degrader interface {
	//检查是否降级
	CheckStatus([]*api.ServiceEntry) ([]*api.ServiceEntry, error)
}

type passingOnlyDegrade struct {
	w *Watch

	//totalNodes 用于动态计算是否达到了预设的阀值(threshold),
	//1. 如果达到了阀值, 则不更新缓存中的值。
	//2. 如果在1个小时内，没有触发过降级，则更新值为当前节点数
	totalNodes     int32
	nextTotalNodes int32

	totalNodesTimer *time.Timer

	threshold   float32
	interval    time.Duration
	passingOnly bool
}

func newPassingOnlyDegrade(w *Watch) *passingOnlyDegrade {
	//初始的时候获取节点数
	p := &passingOnlyDegrade{
		w:           w,
		threshold:   w.option().threshold,
		interval:    w.option().calmInterval,
		passingOnly: w.option().passingOnly,
	}

	return p
}

func (p *passingOnlyDegrade) CheckStatus(entries []*api.ServiceEntry) ([]*api.ServiceEntry, error) {
	p.calcTotalNodes(len(entries))

	if p.shouldDegrade(len(entries)) {
		p.cancelTimer()
		return entries, errors.ErrDegradePass
	}

	if p.passingOnly {
		passingEntries := p.PassingService(entries)
		if !p.shouldDegrade(len(passingEntries)) {
			return passingEntries, nil
		}
	}

	return entries, nil
}

func (p *passingOnlyDegrade) PassingService(entries []*api.ServiceEntry) []*api.ServiceEntry {
	newEntries := make([]*api.ServiceEntry, 0, len(entries))
	for _, entry := range entries {
		if entry.Checks.AggregatedStatus() == api.HealthPassing ||
			entry.Checks.AggregatedStatus() == api.HealthWarning {
			newEntries = append(newEntries, entry)
		}
	}
	return newEntries
}

func (p *passingOnlyDegrade) shouldDegrade(current int) bool {
	totalNodes := atomic.LoadInt32(&p.totalNodes)
	return current < int(float32(totalNodes)*p.threshold)
}

// calctotalnodes 动态调整totalNodes的值
//1. 初始化时更新totalNodes的值
//2. 如果在interval时间内没有发生过降级行为,更新totalNodes的值
func (p *passingOnlyDegrade) calcTotalNodes(total int) {
	totalNodes := atomic.LoadInt32(&p.totalNodes)

	if totalNodes <= 0 {
		atomic.StoreInt32(&p.totalNodes, int32(total))
		logger.Infof("change total nodes form %v to %v", p.totalNodes, total)
		return
	}

	if totalNodes == int32(total) {
		return
	}

	p.nextTotalNodes = int32(total)

	//如果是加机器，立即更新
	if int32(total) > totalNodes {
		atomic.StoreInt32(&p.totalNodes, int32(total))
		logger.Infof("change total nodes form %v to %v", p.totalNodes, total)
		p.cancelTimer()
		return
	}

	if p.totalNodesTimer != nil {
		p.totalNodesTimer.Stop()
		p.totalNodesTimer.Reset(p.interval)
	} else {
		p.totalNodesTimer = time.AfterFunc(p.interval, func() {
			logger.Infof("change totalnodes form %v to %v", p.totalNodes, p.nextTotalNodes)
			atomic.StoreInt32(&p.totalNodes, p.nextTotalNodes)
		})
	}

}

func (p *passingOnlyDegrade) cancelTimer() {
	if p.totalNodesTimer != nil {
		p.totalNodesTimer.Stop()
	}
}
