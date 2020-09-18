package consul

import (
	"time"
)

// 滑动窗口, 线程不安全
type rollingWindow struct {
	size int
	data []int
	cur  int
	now  time.Time
}

func NewRollingWindow(size int) *rollingWindow {
	return &rollingWindow{
		size: size,
		data: make([]int, size),
		cur:  -1,
		now:  time.Now(),
	}
}

func (rw *rollingWindow) incr(now time.Time) {
	since := int(now.Unix() - rw.now.Unix())
	if since == 0 {
		//当前时间窗口
		if rw.cur < 0 {
			rw.cur = (rw.cur + rw.size) % rw.size
		}
		rw.data[rw.cur]++
		return
	}

	//时间已经滚动了很多秒
	if since > rw.size {
		since = rw.size
	}

	//窗口滚动
	for i := 0; i < since; i++ {
		rw.cur = (rw.cur + 1) % rw.size
		rw.data[rw.cur] = 0
	}
	rw.data[rw.cur]++
	rw.now = now
}

//连续n秒的请求数达到m次
func (rw *rollingWindow) isMatch(n int, m int) bool {
	if n > rw.size {
		n = rw.size
	}

	last := (rw.cur - n + rw.size + 1) % rw.size
	count := 0
	for {
		count += rw.data[last]
		if count >= m {
			return true
		}
		if last == rw.cur {
			break
		}
		last = (last + 1) % rw.size
	}
	return false
}

func (rw *rollingWindow) sum(n int) int {
	if n > rw.size {
		n = rw.size
	}

	last := (rw.cur - n + rw.size + 1) % rw.size
	count := 0
	for {
		count += rw.data[last]
		if last == rw.cur {
			break
		}
		last = (last + 1) % rw.size
	}
	return count
}
