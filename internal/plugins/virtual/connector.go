package virtual

//
//import (
//	"github.com/ibuilding-x/driver-box/driverbox/helper"
//	"github.com/ibuilding-x/driver-box/driverbox/plugin"
//	"github.com/ibuilding-x/driver-box/driverbox/plugin/callback"
//)
//
//type connector struct {
//	plugin *Plugin
//}
//
//func (c *connector) Send(data interface{}) (err error) {
//	v, _ := data.(transportationData)
//	switch v.Mode {
//	case plugin.ReadMode: // 读操作（需触发点位上报）
//		if len(v.Points) > 0 {
//			for i, _ := range v.Points {
//				var value any
//				if value, err = helper.DeviceShadow.GetDevicePoint(v.SN, v.Points[i].PointName); err != nil {
//					return err
//				}
//				v.Points[i].Value = value
//			}
//		}
//
//		if _, err = callback.OnReceiveHandler(c.plugin, v); err != nil {
//			return err
//		}
//	case plugin.WriteMode: // 写操作
//		if len(v.Points) > 0 {
//			for _, point := range v.Points {
//				if err = helper.DeviceShadow.SetDevicePoint(v.SN, point.PointName, point.Value); err != nil {
//					return err
//				}
//			}
//		}
//	}
//	return
//}
//
//func (c *connector) Release() (err error) {
//	return nil
//}
