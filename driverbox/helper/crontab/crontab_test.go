package crontab

import (
	"fmt"
	"testing"
	"time"
)

func TestCrontab_AddFunc(t *testing.T) {
	cron := Instance()
	future, err := cron.AddFunc("3s", func() {
		fmt.Println("3s")
	})

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
