package history

import (
	"database/sql"
	_ "github.com/glebarez/sqlite"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/config"
	"github.com/ibuilding-x/driver-box/driverbox/pkg/crontab"
	"github.com/ibuilding-x/driver-box/driverbox/plugin"
	"go.uber.org/zap"
	"os"
	"strconv"
	"sync"
	"time"
)

var once = &sync.Once{}
var instance *Export

const (
	TableNameSnapshotData string = "snapshot_data"
	TableNameRealTimeData string = "real_time_data"

	SchemaSQL = `
CREATE TABLE IF NOT EXISTS snapshot_data( -- 设备影子快照数据。周期性生成，因此point_data包含当前设备完整的点位数据
    id INTEGER PRIMARY KEY NOT NULL, -- 自增主键ID
    device_id varchar(255) NOT null, -- 设备ID
    mo_id varchar(255) ,  -- 模型ID
    point_data TEXT NOT null, -- 设备影子数据，即物模型点位定义的数据，json格式,例如：{"pointName1":"pointValue2","pointName2":"pointValue2"}
    meta TEXT DEFAULT '{}', -- 设备扩展信息
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP --创建时间
    );
CREATE INDEX IF NOT EXISTS idx_devId_ts ON snapshot_data(device_id,create_time);

CREATE INDEX IF NOT EXISTS idx_moId_ts ON snapshot_data(mo_id,create_time);

CREATE TABLE IF NOT EXISTS real_time_data( -- 设备影子实时数据，实时记录设备点位变化值。因此point_data仅包含本次发生变更的内容，若需完整信息可从snapshot_data获取
    id INTEGER PRIMARY KEY NOT NULL, -- 自增主键ID
    device_id varchar(255) NOT null, -- 设备ID
    mo_id varchar(255) ,-- 模型ID
    point_data TEXT NOT null, -- 设备影子数据，即物模型点位定义的数据，json格式,例如：{"pointName1":"pointValue2","pointName2":"pointValue2"}
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP --创建时间
    ); 

CREATE INDEX IF NOT EXISTS idx_devId_time ON real_time_data(device_id,create_time);

CREATE INDEX IF NOT EXISTS idx_moId_time ON real_time_data(mo_id,create_time);
`
)

type Export struct {
	writeNum int
	db       *sql.DB
	ready    bool
	//实时数据缓存
	realTimeDataQueue []deviceQueueData
	realTask          *crontab.Future
	snapshotTask      *crontab.Future
	clearTask         *crontab.Future
}

func NewExport() *Export {
	once.Do(func() {
		instance = &Export{
			writeNum:          20,
			realTimeDataQueue: make([]deviceQueueData, 0),
		}
	})
	return instance
}

// Init 初始化
func (export0 *Export) Init() error {
	e := export0.initHistoryDataDB()
	if e != nil {
		helper.Logger.Error("init history data db error", zap.Error(e))
		return e
	}

	//周期性写入缓冲区中的实时数据
	realTimeCycle := os.Getenv(config.EXPORT_HISTORY_REAL_TIME_FLUSH_INTERVAL)
	if realTimeCycle == "" {
		realTimeCycle = "5s"
	}
	export0.realTask, e = helper.Crontab.AddFunc(realTimeCycle, func() {
		if len(export0.realTimeDataQueue) == 0 {
			return
		}
		queue := export0.realTimeDataQueue
		export0.realTimeDataQueue = make([]deviceQueueData, 0)
		export0.writeRealTimeData(queue)
	})
	if e != nil {
		helper.Logger.Error("register realTime data task error", zap.Error(e))
		return e
	}

	//周期性产生设备剖面数据
	snapshotDataCycle := os.Getenv(config.EXPORT_HISTORY_SNAPSHOT_FLUSH_INTERVAL)
	if snapshotDataCycle == "" {
		snapshotDataCycle = "60s"
	}
	export0.snapshotTask, e = helper.Crontab.AddFunc(snapshotDataCycle, func() {
		export0.writeDeviceSnapshotData()
	})
	if e != nil {
		helper.Logger.Error("register snapshot data task error", zap.Error(e))
		return e
	}

	//周期性清理过期数据
	defaultDuration := 14
	duration := os.Getenv(config.EXPORT_HISTORY_RESERVED_DAYS)
	if duration != "" {
		value, _ := strconv.ParseInt(duration, 10, 64)
		defaultDuration = int(value)
	}
	export0.clearTask, e = helper.Crontab.AddFunc("1h", func() {
		export0.clearExpiredData(defaultDuration)
	})
	if e != nil {
		helper.Logger.Error("register clear history data task error", zap.Error(e))
		return e
	}
	export0.ready = true
	return nil
}

func (export0 *Export) Destroy() error {
	export0.ready = false
	export0.realTask.Disable()
	export0.snapshotTask.Disable()
	export0.clearTask.Disable()
	export0.realTimeDataQueue = make([]deviceQueueData, 0)
	return export0.db.Close()
}

// ExportTo 接收驱动数据
func (export0 *Export) ExportTo(deviceData plugin.DeviceData) {
	if plugin.RealTimeExport == deviceData.ExportType {
		export0.realTimeDataQueue = append(export0.realTimeDataQueue, deviceQueueData{
			deviceData: deviceData,
			addTime:    time.Now(),
		})
		if len(export0.realTimeDataQueue) >= export0.writeNum {
			old := export0.realTimeDataQueue
			export0.realTimeDataQueue = make([]deviceQueueData, 0)
			export0.writeRealTimeData(old)
		}
	}
}

// OnEvent 接收事件数据
func (export0 *Export) OnEvent(eventCode string, key string, eventValue interface{}) error {
	// 暂时不处理任何事件
	return nil
}

func (export0 *Export) IsReady() bool {
	return export0.ready
}

func (export0 *Export) initHistoryDataDB() error {
	dir := os.Getenv(config.EXPORT_HISTORY_DATA_PATH)
	if dir == "" {
		dir = "./res/history_data"
	}
	var err error
	if _, err = os.Stat(dir); os.IsNotExist(err) {
		//目录不存在创建目录
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			helper.Logger.Error("create history data directory error", zap.Error(err))
			return err
		}
	}
	dataSourceName := dir + "/history.db"
	export0.db, err = sql.Open("sqlite", dataSourceName)
	if err != nil {
		helper.Logger.Error("open history db error", zap.Error(err))
		return err
	}

	//设置最大连接数
	export0.db.SetMaxOpenConns(10)
	//设置连接池中最大空闲连接数
	export0.db.SetMaxIdleConns(5)
	export0.db.SetConnMaxLifetime(time.Hour)

	_, err = export0.db.Exec(SchemaSQL)
	if err != nil {
		helper.Logger.Error(err.Error())
		return err
	}
	return nil
}

func (export0 *Export) clearExpiredData(day int) {
	historyDataSql := "DELETE FROM snapshot_data where create_time < ?"
	realTimeDataSql := "DELETE FROM real_time_data where create_time < ?"
	currentTime := time.Now()
	durationDaysAgo := currentTime.Add(-24 * time.Duration(day) * time.Hour)
	_, err := export0.db.Exec(historyDataSql, durationDaysAgo)
	if err != nil {
		helper.Logger.Error(err.Error())
	}
	_, er := export0.db.Exec(realTimeDataSql, durationDaysAgo)
	if er != nil {
		helper.Logger.Error(er.Error())
	}
}
