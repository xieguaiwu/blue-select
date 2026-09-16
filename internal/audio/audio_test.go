// internal/audio/audio_test.go
package audio

import (
	"os"
	"testing"
	"time"
)

func TestParseSinks(t *testing.T) {
	data, err := os.ReadFile("testdata/sinks.txt")
	if err != nil {
		t.Fatal(err)
	}
	sinks, err := ParseSinks(data)
	if err != nil {
		t.Fatalf("ParseSinks: %v", err)
	}
	if len(sinks) != 2 {
		t.Fatalf("want 2 sinks, got %d", len(sinks))
	}
	bt := sinks[1]
	if bt.Name != "bluez_output.30_96_10_FD_B6_88.1" || bt.State != "RUNNING" || bt.ID != "3241" {
		t.Fatalf("bt sink = %+v", bt)
	}
}

func TestParseActiveProfile(t *testing.T) {
	data, err := os.ReadFile("testdata/cards.txt")
	if err != nil {
		t.Fatal(err)
	}
	p, err := ParseActiveProfile(data, "30:96:10:FD:B6:88")
	if err != nil {
		t.Fatalf("ParseActiveProfile: %v", err)
	}
	if p != "a2dp-sink" {
		t.Fatalf("profile = %q, want a2dp-sink", p)
	}
	if _, err := ParseActiveProfile(data, "11:22:33:44:55:66"); err == nil {
		t.Fatal("want error for missing card")
	}
}

func TestParseActiveProfileRealLayout(t *testing.T) {
	// Active Profile 行在 Name 行之前出现（真实 pactl 输出顺序），验证不依赖行序。
	data := []byte("Card #3239\n\t\tActive Profile: a2dp-sink\n\t\tdevice.name = \"bluez_card.30_96_10_FD_B6_88\"\n")
	p, err := ParseActiveProfile(data, "30:96:10:FD:B6:88")
	if err != nil || p != "a2dp-sink" {
		t.Fatalf("p=%q err=%v", p, err)
	}
}

func TestWaitBluezTimeout(t *testing.T) {
	// 无 bluez sink 时应超时而非挂死；耳机连着时 BluezSink 立即命中，跳过该用例。
	if s, ok := BluezSink(); ok {
		t.Skipf("检测到 bluez sink（耳机连着）: %s，跳过超时用例", s.Name)
	}
	if _, err := WaitBluez(800 * time.Millisecond); err == nil {
		t.Fatal("want timeout error")
	}
}

func TestBluezSinkPrefix(t *testing.T) {
	if got := BluezSinkPrefix("30:96:10:FD:B6:88"); got != "bluez_output.30_96_10_FD_B6_88." {
		t.Fatalf("prefix = %q", got)
	}
}

func TestPickFallback(t *testing.T) {
	prefix := "bluez_output.30_96_10_FD_B6_88."
	sinks := []Sink{
		{Name: "alsa_output.pci-0000_00_1f.3.analog-stereo", State: "SUSPENDED"},
		{Name: "bluez_output.30_96_10_FD_B6_88.1", State: "RUNNING"},
		{Name: "bluez_output.30_96_10_FD_B6_88.2", State: "IDLE"},
		{Name: "alsa_output.usb.analog", State: "RUNNING"},
	}
	got, ok := PickFallback(sinks, prefix)
	if !ok {
		t.Fatal("want fallback found")
	}
	if got.Name != "alsa_output.usb.analog" {
		t.Fatalf("fallback = %q, want RUNNING 优先", got.Name)
	}
}

func TestPickFallbackNoExcluded(t *testing.T) {
	prefix := "bluez_output.30_96_10_FD_B6_88."
	sinks := []Sink{
		{Name: "alsa_output.pci-0000_00_1f.3.analog-stereo", State: "SUSPENDED"},
		{Name: "alsa_output.usb.analog", State: "IDLE"},
	}
	got, ok := PickFallback(sinks, prefix)
	if !ok || got.Name != "alsa_output.usb.analog" {
		t.Fatalf("want IDLE 优先于 SUSPENDED, got %+v ok=%v", got, ok)
	}
}

func TestPickFallbackAllExcluded(t *testing.T) {
	prefix := "bluez_output.30_96_10_FD_B6_88."
	sinks := []Sink{
		{Name: "bluez_output.30_96_10_FD_B6_88.1", State: "RUNNING"},
	}
	if _, ok := PickFallback(sinks, prefix); ok {
		t.Fatal("want not found when all excluded")
	}
}

func TestPickFallbackEmpty(t *testing.T) {
	if _, ok := PickFallback(nil, "x."); ok {
		t.Fatal("want not found on empty input")
	}
}
