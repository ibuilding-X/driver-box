package crontab

import (
	"fmt"
	"testing"
	"time"
)

func TestCrontab_AddFunc(t *testing.T) {
	cron := NewCrontab()
	future, err := cron.AddFunc("3s", func() {
		fmt.Println("3s")
	})
	cron.Start()

	cron.AddFunc("2s", func() {
		fmt.Println("2s")
	})

	if err != nil {
		t.Error(err)
		return
	}
	time.Sleep(10 * time.Second)
	fmt.Println("disable cron")
	future.Disable()
	time.Sleep(5 * time.Second)
	fmt.Println("finish cron")
}
