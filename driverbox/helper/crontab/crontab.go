package crontab

import (
	"context"
	"time"
)

type Crontab interface {
	Start()
	Stop()
	// AddFunc s please refer to time.ParseDuration
	AddFunc(s string, f func()) error
}

func NewCrontab() Crontab {
	ctx, cancel := context.WithCancel(context.Background())
	return &crontab{
		signal: cancel,
		ctx:    ctx,
	}
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

func (c *crontab) AddFunc(s string, f func()) error {
	d, err := time.ParseDuration(s)
	if err != nil {
		return err
	}
	go c.handelFunc(d, f)
	return nil
}

func (c *crontab) handelFunc(d time.Duration, f func()) {
	select {
	case <-c.ctx.Done():
		ticker := time.NewTicker(d)
		c.tickerArray = append(c.tickerArray, ticker)
		for range ticker.C {
			f()
		}
	}
}
