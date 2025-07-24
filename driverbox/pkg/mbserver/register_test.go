package mbserver

//
//import (
//	"fmt"
//	"testing"
//)
//
//func TestRegister(t *testing.T) {
//	register := NewRegister()
//
//	deviceNumber := 10
//
//	// 初始化
//	t.Run("Initialize", func(t *testing.T) {
//		models := []Model{
//			{
//				Id: "20f35e630daf44dbfa4c3f68f5399d8c",
//				Properties: []Property{
//					{
//						Name:      "on",
//						ValueType: ValueTypeUint16,
//						Access:    AccessReadWrite,
//					},
//					{
//						Name:      "temp",
//						ValueType: ValueTypeFloat32,
//						Access:    AccessReadWrite,
//					},
//					{
//						Name:      "mode",
//						ValueType: ValueTypeUint16,
//						Access:    AccessReadWrite,
//					},
//					{
//						Name:      "fanSpeed",
//						ValueType: ValueTypeUint16,
//						Access:    AccessReadWrite,
//					},
//					{
//						Name:      "errorCode",
//						ValueType: ValueTypeUint16,
//						Access:    AccessReadWrite,
//					},
//				},
//			},
//		}
//
//		var devices []Device
//		for i := 0; i < deviceNumber; i++ {
//			devices = append(devices, Device{
//				Mid: "20f35e630daf44dbfa4c3f68f5399d8c",
//				Id:  fmt.Sprintf("device_%d", i),
//			})
//		}
//
//		err := register.Initialize(models, devices)
//		if err != nil {
//			t.Fatal(err)
//		}
//	})
//
//	// 设置属性
//	t.Run("SetProperty", func(t *testing.T) {
//		for i := 0; i < deviceNumber; i++ {
//			did := fmt.Sprintf("device_%d", i)
//
//			if err := register.SetProperty(did, "on", 1); err != nil {
//				t.Fatalf("set property error: %v", err)
//			}
//			if err := register.SetProperty(did, "temp", 26.5); err != nil {
//				t.Fatalf("set property error: %v", err)
//			}
//			if err := register.SetProperty(did, "mode", 2); err != nil {
//				t.Fatalf("set property error: %v", err)
//			}
//			if err := register.SetProperty(did, "fanSpeed", 3); err != nil {
//				t.Fatalf("set property error: %v", err)
//			}
//			if err := register.SetProperty(did, "errorCode", 16); err != nil {
//				t.Fatalf("set property error: %v", err)
//			}
//		}
//	})
//
//	// 获取属性
//	t.Run("GetProperty", func(t *testing.T) {
//		for i := 0; i < deviceNumber; i++ {
//			did := fmt.Sprintf("device_%d", i)
//
//			if v, err := register.GetProperty(did, "on"); err != nil {
//				t.Fatalf("get property error: %v", err)
//			} else {
//				if fmt.Sprintf("%v", v) != "1" {
//					t.Fatalf("get property [on] error: %v", v)
//				}
//			}
//
//			if v, err := register.GetProperty(did, "temp"); err != nil {
//				t.Fatalf("get property error: %v", err)
//			} else {
//				if fmt.Sprintf("%v", v) != "26.5" {
//					t.Fatalf("get property [temp] error: %v", v)
//				}
//			}
//
//			if v, err := register.GetProperty(did, "mode"); err != nil {
//				t.Fatalf("get property error: %v", err)
//			} else {
//				if fmt.Sprintf("%v", v) != "2" {
//					t.Fatalf("get property [mode] error: %v", v)
//				}
//			}
//
//			if v, err := register.GetProperty(did, "fanSpeed"); err != nil {
//				t.Fatalf("get property error: %v", err)
//			} else {
//				if fmt.Sprintf("%v", v) != "3" {
//					t.Fatalf("get property [fanSpeed] error: %v", v)
//				}
//			}
//
//			if v, err := register.GetProperty(did, "errorCode"); err != nil {
//				t.Fatalf("get property error: %v", err)
//			} else {
//				if fmt.Sprintf("%v", v) != "16" {
//					t.Fatalf("get property [errorCode] error: %v", v)
//				}
//			}
//		}
//	})
//}
//
//func BenchmarkRegister(b *testing.B) {
//	register := NewRegister()
//	deviceNumber := 1000
//	models := []Model{
//		{
//			Id: "20f35e630daf44dbfa4c3f68f5399d8c",
//			Properties: []Property{
//				{
//					Name:      "on",
//					ValueType: ValueTypeUint16,
//					Access:    AccessReadWrite,
//				},
//				{
//					Name:      "temp",
//					ValueType: ValueTypeFloat32,
//					Access:    AccessReadWrite,
//				},
//				{
//					Name:      "mode",
//					ValueType: ValueTypeUint16,
//					Access:    AccessReadWrite,
//				},
//				{
//					Name:      "fanSpeed",
//					ValueType: ValueTypeUint16,
//					Access:    AccessReadWrite,
//				},
//				{
//					Name:      "errorCode",
//					ValueType: ValueTypeUint16,
//					Access:    AccessReadWrite,
//				},
//			},
//		},
//	}
//
//	var devices []Device
//	for i := 0; i < deviceNumber; i++ {
//		devices = append(devices, Device{
//			Mid: "20f35e630daf44dbfa4c3f68f5399d8c",
//			Id:  fmt.Sprintf("device_%d", i),
//		})
//	}
//
//	err := register.Initialize(models, devices)
//	if err != nil {
//		b.Fatal(err)
//	}
//
//	b.ResetTimer()
//	b.Run("SetProperty", func(b *testing.B) {
//		for i := 0; i < b.N; i++ {
//			if err = register.SetProperty("device_0", "on", 1); err != nil {
//				b.Fatal(err)
//			}
//		}
//	})
//
//	b.ResetTimer()
//	b.Run("GetProperty", func(b *testing.B) {
//		for i := 0; i < b.N; i++ {
//			if _, err = register.GetProperty("device_0", "on"); err != nil {
//				b.Fatal(err)
//			}
//		}
//	})
//}
