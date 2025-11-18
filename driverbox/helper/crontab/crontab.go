package crontab

import (
	"sync"
	"time"

	"github.com/robfig/cron/v3"
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
		instance = &crontab{
			c: cron.New(cron.WithSeconds()),
		}
		instance.c.Start()
	})
	return instance
}

type Future struct {
	//外部传入的定时任务函数
	function func()
	//定时器
	ticker *time.Ticker
	cronId cron.EntryID
	//是否启用
	enable bool
}
type crontab struct {
	futures []*Future
	c       *cron.Cron
}

func (c *crontab) Clear() {
	if len(c.futures) > 0 {
		for i, _ := range c.futures {
			c.futures[i].Disable()
		}
		c.futures = make([]*Future, 0)
	}
}

func (c *crontab) AddFunc(s string, f func()) (*Future, error) {
	d, err := time.ParseDuration(s)
	if err == nil {
		function := &Future{
			function: f,
			ticker:   time.NewTicker(d),
			enable:   true,
		}
		c.futures = append(c.futures, function)
		go function.run()
		return function, nil
	}
	//尝试按照crontab格式添加
	cronId, err := c.c.AddFunc(s, f)
	if err != nil {
		return &Future{}, err
	}
	function := &Future{
		function: f,
		cronId:   cronId,
		enable:   true,
	}
	c.futures = append(c.futures, function)
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
	if f.ticker != nil {
		f.ticker.Reset(1)
	}
	if f.cronId != 0 {
		instance.c.Remove(f.cronId)
	}
}
