// internal/audio/audio_test.go
package audio

import (
	"os"
	"strings"
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

func TestSinkMAC(t *testing.T) {
	mac, ok := SinkMAC("bluez_output.30_96_10_FD_B6_88.1")
	if !ok || mac != "30:96:10:FD:B6:88" {
		t.Fatalf("SinkMAC = %q ok=%v", mac, ok)
	}
	// 与 BluezSinkPrefix 往返一致
	if got := BluezSinkPrefix(mac); !strings.HasPrefix("bluez_output.30_96_10_FD_B6_88.1", got) {
		t.Fatalf("roundtrip mismatch: prefix %q", got)
	}
	for _, bad := range []string{
		"alsa_output.pci-0000_00_1f.3.analog-stereo",
		"bluez_output.short",
		"",
	} {
		if _, ok := SinkMAC(bad); ok {
			t.Errorf("SinkMAC(%q) = ok, want false", bad)
		}
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

func TestCardName(t *testing.T) {
	if got := CardName("30:96:10:FD:B6:88"); got != "bluez_card.30_96_10_FD_B6_88" {
		t.Fatalf("CardName = %q", got)
	}
}

func TestIsA2DP(t *testing.T) {
	for _, p := range []string{"a2dp-sink", "a2dp-sink-sbc", "a2dp-sink-sbc_xq"} {
		if !IsA2DP(p) {
			t.Errorf("IsA2DP(%q) = false, want true", p)
		}
	}
	for _, p := range []string{"headset-head-unit", "headset-head-unit-cvsd", "off", ""} {
		if IsA2DP(p) {
			t.Errorf("IsA2DP(%q) = true, want false", p)
		}
	}
}

// TestParseProfilesA2DP 覆盖健康状态：A2DP 可用且为 Active Profile。
func TestParseProfilesA2DP(t *testing.T) {
	data, err := os.ReadFile("testdata/cards_a2dp.txt")
	if err != nil {
		t.Fatal(err)
	}
	profs, err := ParseProfiles(data, "30:96:10:FD:B6:88")
	if err != nil {
		t.Fatalf("ParseProfiles: %v", err)
	}
	if !profs.Has("a2dp-sink") || !profs.Has("headset-head-unit") {
		t.Fatalf("profiles = %v, want both a2dp-sink and headset-head-unit", profs.Available)
	}
	if !profs.Has("a2dp-sink-sbc") {
		t.Fatalf("profiles = %v, want a2dp-sink-sbc", profs.Available)
	}
	if profs.Active != "a2dp-sink" {
		t.Fatalf("Active = %q, want a2dp-sink", profs.Active)
	}
	if _, err := ParseProfiles(data, "11:22:33:44:55:66"); err == nil {
		t.Fatal("want error for missing card")
	}
}

// TestParseProfilesHSPOnly 覆盖真实故障现场（2026-09-21 09:48 实测）：
// A2DP transport 失败后卡只剩 HSP/HFP，此时 connect 仍报成功但音频是 mono 16kHz。
func TestParseProfilesHSPOnly(t *testing.T) {
	data, err := os.ReadFile("testdata/cards_hsp_only.txt")
	if err != nil {
		t.Fatal(err)
	}
	profs, err := ParseProfiles(data, "30:96:10:FD:B6:88")
	if err != nil {
		t.Fatalf("ParseProfiles: %v", err)
	}
	if profs.Has("a2dp-sink") {
		t.Fatal("degraded fixture must not offer a2dp-sink")
	}
	if profs.Active != "headset-head-unit" {
		t.Fatalf("Active = %q, want headset-head-unit", profs.Active)
	}
	if IsA2DP(profs.Active) {
		t.Fatal("headset-head-unit must not classify as A2DP")
	}
}

// TestParseProfilesSkipsUnavailable A2DP profile 存在但 available: no 时不可选。
func TestParseProfilesSkipsUnavailable(t *testing.T) {
	data := []byte("Card #1\n\t\tdevice.name = \"bluez_card.30_96_10_FD_B6_88\"\n" +
		"\t\ta2dp-sink: High Fidelity Playback (A2DP Sink, codec AAC) (sinks: 1, sources: 0, priority: 19, available: no)\n" +
		"\t\theadset-head-unit: Headset Head Unit (HSP/HFP, codec mSBC) (sinks: 1, sources: 1, priority: 3, available: yes)\n" +
		"\tActive Profile: headset-head-unit\n")
	profs, err := ParseProfiles(data, "30:96:10:FD:B6:88")
	if err != nil {
		t.Fatalf("ParseProfiles: %v", err)
	}
	if profs.Has("a2dp-sink") {
		t.Fatal("available: no profile must not be selectable")
	}
	if _, ok := PickA2DPProfile(profs.Available); ok {
		t.Fatal("PickA2DPProfile must fail when only unavailable A2DP exists")
	}
}

// TestPickA2DPProfile 优先选 AAC 的 a2dp-sink（priority 最高），回落 SBC。
func TestPickA2DPProfile(t *testing.T) {
	got, ok := PickA2DPProfile([]string{"off", "headset-head-unit", "a2dp-sink-sbc", "a2dp-sink"})
	if !ok || got != "a2dp-sink" {
		t.Fatalf("PickA2DPProfile = %q ok=%v, want a2dp-sink", got, ok)
	}
	got, ok = PickA2DPProfile([]string{"off", "headset-head-unit", "a2dp-sink-sbc_xq"})
	if !ok || got != "a2dp-sink-sbc_xq" {
		t.Fatalf("PickA2DPProfile = %q ok=%v, want a2dp-sink-sbc_xq", got, ok)
	}
	if _, ok := PickA2DPProfile([]string{"off", "headset-head-unit"}); ok {
		t.Fatal("want not found when no A2DP available")
	}
	if _, ok := PickA2DPProfile(nil); ok {
		t.Fatal("want not found on empty input")
	}
}

// TestIsHFP 验证 HFP/HSP 档位识别。
func TestIsHFP(t *testing.T) {
	for _, p := range []string{"headset-head-unit", "headset-head-unit-cvsd"} {
		if !IsHFP(p) {
			t.Errorf("IsHFP(%q) = false, want true", p)
		}
	}
	for _, p := range []string{"a2dp-sink", "a2dp-sink-sbc", "off", ""} {
		if IsHFP(p) {
			t.Errorf("IsHFP(%q) = true, want false", p)
		}
	}
}

// TestPickHFPProfile 优先选 mSBC 的 headset-head-unit（音质优于 CVSD）。
func TestPickHFPProfile(t *testing.T) {
	got, ok := PickHFPProfile([]string{"off", "a2dp-sink", "headset-head-unit-cvsd", "headset-head-unit"})
	if !ok || got != "headset-head-unit" {
		t.Fatalf("PickHFPProfile = %q ok=%v, want headset-head-unit", got, ok)
	}
	got, ok = PickHFPProfile([]string{"off", "a2dp-sink", "headset-head-unit-cvsd"})
	if !ok || got != "headset-head-unit-cvsd" {
		t.Fatalf("PickHFPProfile = %q ok=%v, want headset-head-unit-cvsd", got, ok)
	}
	if _, ok := PickHFPProfile([]string{"off", "a2dp-sink"}); ok {
		t.Fatal("want not found when no HFP available")
	}
	if _, ok := PickHFPProfile(nil); ok {
		t.Fatal("want not found on empty input")
	}
}

// TestParseSources 验证 source 解析。
func TestParseSources(t *testing.T) {
	data := []byte("57\talsa_output.pci-0000_00_1f.3.analog-stereo.monitor\tPipeWire\ts32le 2ch 48000Hz\tRUNNING\n" +
		"58\talsa_input.pci-0000_00_1f.3.analog-stereo\tPipeWire\ts32le 2ch 48000Hz\tRUNNING\n" +
		"59\tbluez_input.30_96_10_FD_B6_88.1\tPipeWire\ts16le 1ch 16000Hz\tIDLE\n")
	sources, err := ParseSources(data)
	if err != nil {
		t.Fatalf("ParseSources: %v", err)
	}
	if len(sources) != 3 {
		t.Fatalf("want 3 sources, got %d", len(sources))
	}
	bt := sources[2]
	if bt.Name != "bluez_input.30_96_10_FD_B6_88.1" || bt.State != "IDLE" || bt.ID != "59" {
		t.Fatalf("bt source = %+v", bt)
	}
}

// TestBluezSourcePrefix 验证 source 前缀生成。
func TestBluezSourcePrefix(t *testing.T) {
	if got := BluezSourcePrefix("30:96:10:FD:B6:88"); got != "bluez_input.30_96_10_FD_B6_88." {
		t.Fatalf("prefix = %q", got)
	}
}

// TestSourceMAC 验证从 source 名反解 MAC。
func TestSourceMAC(t *testing.T) {
	mac, ok := SourceMAC("bluez_input.30_96_10_FD_B6_88.1")
	if !ok || mac != "30:96:10:FD:B6:88" {
		t.Fatalf("SourceMAC = %q ok=%v", mac, ok)
	}
	// 与 BluezSourcePrefix 往返一致
	if got := BluezSourcePrefix(mac); !strings.HasPrefix("bluez_input.30_96_10_FD_B6_88.1", got) {
		t.Fatalf("roundtrip mismatch: prefix %q", got)
	}
	for _, bad := range []string{
		"alsa_input.pci-0000_00_1f.3.analog-stereo",
		"bluez_input.short",
		"",
	} {
		if _, ok := SourceMAC(bad); ok {
			t.Errorf("SourceMAC(%q) = ok, want false", bad)
		}
	}
}
