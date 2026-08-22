// main_test.go
package main

import (
	"strings"
	"testing"

	"blue-select/internal/bluetooth"
	"blue-select/internal/config"
)

var testDevs = []bluetooth.Device{
	{MAC: "30:96:10:FD:B6:88", Name: "HUAWEI FreeArc"},
	{MAC: "A4:C1:38:11:22:33", Name: "Mi Band 7"},
}

func TestResolveDeviceByArg(t *testing.T) {
	cfg := &config.Config{Devices: map[string]string{}}
	mac, err := ResolveDevice(cfg, "freearc", testDevs) // 大小写不敏感子串匹配
	if err != nil {
		t.Fatalf("ResolveDevice: %v", err)
	}
	if mac != "30:96:10:FD:B6:88" {
		t.Fatalf("mac = %s", mac)
	}
}

func TestResolveDeviceDefault(t *testing.T) {
	cfg := &config.Config{DefaultDevice: "HUAWEI FreeArc", Devices: map[string]string{}}
	mac, err := ResolveDevice(cfg, "", testDevs)
	if err != nil {
		t.Fatalf("ResolveDevice: %v", err)
	}
	if mac != "30:96:10:FD:B6:88" {
		t.Fatalf("mac = %s", mac)
	}
}

func TestResolveDeviceErrors(t *testing.T) {
	cfg := &config.Config{Devices: map[string]string{}}
	// 无参数且无默认设备：错误信息列出已知设备
	_, err := ResolveDevice(cfg, "", testDevs)
	if err == nil || !strings.Contains(err.Error(), "HUAWEI FreeArc") {
		t.Fatalf("want device list in error, got %v", err)
	}
	// 参数无匹配
	_, err = ResolveDevice(cfg, "airpods", testDevs)
	if err == nil {
		t.Fatal("want error for no match")
	}
	// 默认设备不在已知列表
	cfg2 := &config.Config{DefaultDevice: "AirPods", Devices: map[string]string{}}
	_, err = ResolveDevice(cfg2, "", testDevs)
	if err == nil {
		t.Fatal("want error for default not found")
	}
}
