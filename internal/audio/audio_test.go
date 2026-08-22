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
	// 空环境无 bluez sink，应超时而非挂死。
	if _, err := WaitBluez(800 * time.Millisecond); err == nil {
		t.Fatal("want timeout error")
	}
}
