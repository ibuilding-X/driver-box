package crontab

import (
	"context"
	"time"
)

type Crontab interface {
	Start()
	Stop()
	// AddFunc s please refer to time.ParseDuration
	AddFunc(s string, f func()) (*Future, error)
}

func NewCrontab() Crontab {
	ctx, cancel := context.WithCancel(context.Background())
	return &crontab{
		signal: cancel,
		ctx:    ctx,
	}
}

type Future struct {
	//外部传入的定时任务函数
	function func()
	//定时器
	ticker *time.Ticker
	//是否启用
	enable bool
	ctx    context.Context
}
type crontab struct {
	signal      context.CancelFunc
	ctx         context.Context
	tickerArray []*time.Ticker
}

func (c *crontab) Start() {
	c.signal()
}

func (c *crontab) Stop() {
	if len(c.tickerArray) > 0 {
		for i, _ := range c.tickerArray {
			c.tickerArray[i].Stop()
		}
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
		ctx:      c.ctx,
	}
	go function.run()
	return function, nil
}

func (f *Future) run() {
	select {
	case <-f.ctx.Done():
		for range f.ticker.C {
			if !f.enable {
				f.ticker.Stop()
				break
			}
			f.function()
		}
	}
	//fmt.Println("stop run")
}
func (f *Future) Disable() {
	f.enable = false
}
