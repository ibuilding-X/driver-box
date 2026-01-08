package linkedge

import (
	"github.com/ibuilding-x/driver-box/pkg/driverbox/export/linkedge"
	"github.com/robfig/cron/v3"
	"testing"
)

var scene = `
{
    "name": "test_scene",
    "enable": true,
    "description": "this is a test scene",
    "tags": [
        "test"
    ],
    "silentPeriod": 3,
    "trigger": [
        {
            "type": "schedule",
            "cron": "* * * * *"
        },
        {
            "type": "devicePoint",
            "devSn": "device_1",
            "point": "onOff",
            "contition": "=",
            "value": "1"
        }
    ],
    "condition": [
        {
            "type": "devicePoint",
            "devSn": "device_1",
            "point": "onOff",
            "contition": "=",
            "value": "1"
        },
        {
            "type": "executeTime",
            "begin": 1713943756000,
            "end": 1713943756000
        },
        {
            "type": "years",
            "years": [
                2020,
                2021
            ]
        },
        {
            "type": "months",
            "months": [
                1,
                2
            ]
        },
        {
            "type": "days",
            "days": [
                1,
                2,
                3
            ]
        },
        {
            "type": "weeks",
            "weeks": [
                0,
                1,
                2
            ]
        },
        {
            "type": "times",
            "begin_times": "09:00",
            "end_times": "18:00"
        }
    ],
    "action": [
        {
            "type": "devicePoint",
            "condition": [

            ],
            "attr": {

            },
            "sleep": "3s",
            "devSn": "device_1",
            "point": "onOff",
            "value": "1"
        },
        {
            "type": "linkEdge",
            "id": "xxxxxx"
        }
    ]
}
`

func TestCreate(t *testing.T) {
	s := &service{
		configs:           make(map[string]linkedge.Config),
		schedules:         make(map[string]*cron.Cron),
		triggerConditions: make(map[string][]linkedge.DevicePointCondition),
		envConfig:         EnvConfig{},
	}
	if err := s.Create([]byte(scene)); err != nil {
		t.Error(err)
	}
}
