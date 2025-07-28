package crontab

import (
	"sync"
	"time"
)

var instance *crontab
var once = &sync.Once{}

type Crontab interface {
	Clear()
	// AddFunc s please refer to time.ParseDuration
	AddFunc(s string, f func()) (*Future, error)
}

func Instance() Crontab {
	once.Do(func() {
		instance = &crontab{}
	})
	return instance
}

type Future struct {
	//外部传入的定时任务函数
	function func()
	//定时器
	ticker *time.Ticker
	//是否启用
	enable bool
}
type crontab struct {
	tickerArray []*time.Ticker
}

func (c *crontab) Clear() {
	if len(c.tickerArray) > 0 {
		for i, _ := range c.tickerArray {
			c.tickerArray[i].Stop()
		}
		c.tickerArray = make([]*time.Ticker, 0)
	}
}

func (c *crontab) AddFunc(s string, f func()) (*Future, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return &Future{}, err
	}
	function := &Future{
		function: f,
		ticker:   time.NewTicker(d),
		enable:   true,
	}
	go function.run()
	return function, nil
}

func (f *Future) run() {
	for range f.ticker.C {
		if !f.enable {
			f.ticker.Stop()
			break
		}
		f.function()
	}
}
func (f *Future) Disable() {
	f.enable = false
}
