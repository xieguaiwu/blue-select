// internal/bluetooth/bluetooth_test.go
package bluetooth

import (
	"os"
	"testing"
)

func TestParseDevices(t *testing.T) {
	data, err := os.ReadFile("testdata/devices.txt")
	if err != nil {
		t.Fatal(err)
	}
	devs, err := ParseDevices(data)
	if err != nil {
		t.Fatalf("ParseDevices: %v", err)
	}
	if len(devs) != 3 {
		t.Fatalf("want 3 devices, got %d", len(devs))
	}
	if devs[0].MAC != "30:96:10:FD:B6:88" || devs[0].Name != "HUAWEI FreeArc" {
		t.Fatalf("devs[0] = %+v", devs[0])
	}
}

func TestParseInfoConnected(t *testing.T) {
	data, err := os.ReadFile("testdata/info_connected.txt")
	if err != nil {
		t.Fatal(err)
	}
	info, err := ParseInfo(data)
	if err != nil {
		t.Fatalf("ParseInfo: %v", err)
	}
	if !info.Connected || !info.Paired || !info.Trusted {
		t.Fatalf("flags wrong: %+v", info)
	}
	if !info.HasBattery || info.Battery != 70 {
		t.Fatalf("battery wrong: %+v", info)
	}
}

func TestParseInfoDisconnected(t *testing.T) {
	data, err := os.ReadFile("testdata/info_disconnected.txt")
	if err != nil {
		t.Fatal(err)
	}
	info, err := ParseInfo(data)
	if err != nil {
		t.Fatalf("ParseInfo: %v", err)
	}
	if info.Connected {
		t.Fatal("want Connected=false")
	}
	if info.HasBattery {
		t.Fatal("want HasBattery=false")
	}
}

func TestLastMeaningfulLine(t *testing.T) {
	// 真实失败现场：事件行刷屏，错误在最后一行
	in := "[NEW] Device 30:96:10:FD:B6:88 HUAWEI FreeArc\n" +
		"[CHG] Device 30:96:10:FD:B6:88 Connected: yes\n" +
		"Failed to pair: org.bluez.Error.AuthenticationFailed\n"
	want := "Failed to pair: org.bluez.Error.AuthenticationFailed"
	if got := lastMeaningfulLine(in); got != want {
		t.Fatalf("lastMeaningfulLine = %q, want %q", got, want)
	}
	// 尾部空行 / 只有事件行 / 空输入
	if got := lastMeaningfulLine("Pairing successful\n\n"); got != "Pairing successful" {
		t.Fatalf("trailing blank not skipped: %q", got)
	}
	if got := lastMeaningfulLine("[CHG] foo\n[NEW] bar\n"); got != "" {
		t.Fatalf("event-only should yield empty, got %q", got)
	}
	if got := lastMeaningfulLine(""); got != "" {
		t.Fatalf("empty input = %q", got)
	}
}
