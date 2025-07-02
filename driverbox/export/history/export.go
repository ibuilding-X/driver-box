package history

import (
	"database/sql"
	_ "github.com/glebarez/sqlite"
	"github.com/ibuilding-x/driver-box/driverbox/config"
	"github.com/ibuilding-x/driver-box/driverbox/helper"
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
)

type Export struct {
	writeNum int
	db       *sql.DB
	ready    bool
	//实时数据缓存
	realTimeDataQueue []deviceQueueData
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
	_, e = helper.Crontab.AddFunc(realTimeCycle, func() {
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
	_, e = helper.Crontab.AddFunc(snapshotDataCycle, func() {
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
	_, e = helper.Crontab.AddFunc("1h", func() {
		export0.clearExpiredData(defaultDuration)
	})
	if e != nil {
		helper.Logger.Error("register clear history data task error", zap.Error(e))
		return e
	}
	export0.ready = true
	return nil
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

	return export0.initHistorySchema()
}

func (export0 *Export) initHistorySchema() error {
	historyDataTable := "CREATE TABLE IF NOT EXISTS snapshot_data(id INTEGER PRIMARY KEY NOT NULL,device_id varchar(255) NOT null,mo_id varchar(255) ,point_data TEXT NOT null,meta TEXT DEFAULT '{}',create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);"
	_, err := export0.db.Exec(historyDataTable)
	if err != nil {
		helper.Logger.Error(err.Error())
		return err
	}
	historyIdxSql := "CREATE INDEX IF NOT EXISTS idx_devId_ts ON snapshot_data(device_id,create_time);"
	_, err = export0.db.Exec(historyIdxSql)
	if err != nil {
		helper.Logger.Warn(err.Error())
		return err
	}
	historyMoIdxSql := "CREATE INDEX IF NOT EXISTS idx_moId_ts ON snapshot_data(mo_id,create_time);"
	_, err = export0.db.Exec(historyMoIdxSql)
	if err != nil {
		helper.Logger.Warn(err.Error())
		return err
	}

	realTimeDataTable := "CREATE TABLE IF NOT EXISTS real_time_data(id INTEGER PRIMARY KEY NOT NULL,device_id varchar(255) NOT null,mo_id varchar(255) ,point_data TEXT NOT null,create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP);"
	_, er := export0.db.Exec(realTimeDataTable)
	if er != nil {
		helper.Logger.Error(er.Error())
		return err
	}
	realTimeIdxSql := "CREATE INDEX IF NOT EXISTS idx_devId_time ON real_time_data(device_id,create_time);"
	_, err = export0.db.Exec(realTimeIdxSql)
	if err != nil {
		helper.Logger.Warn(err.Error())
		return err
	}
	realTImeMoIdxSql := "CREATE INDEX IF NOT EXISTS idx_moId_time ON real_time_data(mo_id,create_time);"
	_, err = export0.db.Exec(realTImeMoIdxSql)
	if err != nil {
		helper.Logger.Warn(err.Error())
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
