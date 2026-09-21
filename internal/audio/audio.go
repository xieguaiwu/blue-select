// internal/audio/audio.go
package audio

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Sink 是 `pactl list sinks short` 的一行。
type Sink struct {
	ID    string // 首列编号，move-sink-input 用不到但 status 展示用
	Name  string
	State string
}

func pactl(timeoutSec string, args ...string) ([]byte, error) {
	full := append([]string{timeoutSec, "pactl"}, args...)
	out, err := exec.Command("timeout", full...).CombinedOutput()
	if err != nil {
		return out, fmt.Errorf("pactl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
	}
	return out, nil
}

// ParseSinks 解析 `pactl list sinks short`。行格式: ID\tName\tDriver\tSpec\tState
// （2026-08-22 本机实测 5 列，无 Channels 列）
func ParseSinks(data []byte) ([]Sink, error) {
	var sinks []Sink
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 5 {
			continue
		}
		sinks = append(sinks, Sink{ID: f[0], Name: f[1], State: f[4]})
	}
	if len(sinks) == 0 {
		return nil, fmt.Errorf("pactl list sinks short: 无 sink 或输出无法解析")
	}
	return sinks, nil
}

// Sinks 返回当前所有 sink。
func Sinks() ([]Sink, error) {
	out, err := pactl("10", "list", "sinks", "short")
	if err != nil {
		return nil, err
	}
	return ParseSinks(out)
}

// BluezSink 返回第一个 bluez_output sink。
func BluezSink() (Sink, bool) {
	sinks, err := Sinks()
	if err != nil {
		return Sink{}, false
	}
	for _, s := range sinks {
		if strings.HasPrefix(s.Name, "bluez_output.") {
			return s, true
		}
	}
	return Sink{}, false
}

// BluezSinkPrefix 返回指定设备 MAC 的所有 sink 名公共前缀（冒号换下划线）。
// 同一设备的 A2DP/HFP profile 会产生 .1/.2 多个 sink，用前缀一并排除。
func BluezSinkPrefix(mac string) string {
	return "bluez_output." + strings.ReplaceAll(mac, ":", "_") + "."
}

// PickFallback 从 sinks 中选回退输出：排除 exclude 前缀的 sink，
// 状态优先级 RUNNING > IDLE > SUSPENDED > 其他，同级保持原顺序。
func PickFallback(sinks []Sink, excludePrefix string) (Sink, bool) {
	prio := func(state string) int {
		switch state {
		case "RUNNING":
			return 0
		case "IDLE":
			return 1
		case "SUSPENDED":
			return 2
		default:
			return 3
		}
	}
	var best Sink
	found := false
	for _, s := range sinks {
		if strings.HasPrefix(s.Name, excludePrefix) {
			continue
		}
		// 严格小于保证同级时保留先出现的（稳定）
		if !found || prio(s.State) < prio(best.State) {
			best, found = s, true
		}
	}
	return best, found
}

// Default 返回默认 sink 名。
func Default() (string, error) {
	out, err := pactl("10", "get-default-sink")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// SetDefault 设默认输出。
func SetDefault(name string) error {
	_, err := pactl("10", "set-default-sink", name)
	return err
}

// MoveAllStreams 把所有播放流迁到指定 sink，返回迁移数。
func MoveAllStreams(sinkName string) (int, error) {
	out, err := pactl("10", "list", "sink-inputs", "short")
	if err != nil {
		return 0, err
	}
	moved := 0
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Split(strings.TrimRight(line, "\r"), "\t")
		if len(f) < 1 || f[0] == "" {
			continue
		}
		if _, err := pactl("10", "move-sink-input", f[0], sinkName); err == nil {
			moved++
		}
	}
	return moved, nil
}

// WaitBluez 轮询等待 bluez sink 出现，500ms 间隔，超时返回错误。
func WaitBluez(timeout time.Duration) (Sink, error) {
	deadline := time.Now().Add(timeout)
	for {
		if s, ok := BluezSink(); ok {
			return s, nil
		}
		if time.Now().After(deadline) {
			return Sink{}, fmt.Errorf("等待蓝牙 sink 超时（%v）", timeout)
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// CardName 返回 bluez 音频卡名（MAC 冒号换下划线）。
func CardName(mac string) string {
	return "bluez_card." + strings.ReplaceAll(mac, ":", "_")
}

// CardProfiles 描述一张蓝牙卡的档位状态。
// Available 只收录 `available: yes` 的档位——`available: no` 的档位切不过去。
// 2026-09-21 实测：A2DP transport 失败后，卡的 Profiles 段里 a2dp-sink 可能整个消失。
// 此时 Active 会回落到 headset-head-unit（HSP/HFP，mono 16kHz）。
type CardProfiles struct {
	Active    string
	Available []string
}

// Has 报告某档位当前是否可选。
func (p CardProfiles) Has(name string) bool {
	for _, a := range p.Available {
		if a == name {
			return true
		}
	}
	return false
}

// cardBlock 返回含 bluez_card.<mac> 的那个 Card 块。
func cardBlock(data []byte, mac string) (string, bool) {
	needle := CardName(mac)
	for _, card := range bytes.Split(data, []byte("Card #")) {
		block := string(card)
		if strings.Contains(block, needle) {
			return block, true
		}
	}
	return "", false
}

// ParseProfiles 解析指定蓝牙卡的 Active Profile 与可用档位列表。
// 按 Card 块切分，块内同时含 bluez_card.<mac> 与 Active Profile: 即命中，
// 不依赖两行的先后顺序。Profiles 段以缩进更深的行（两个 tab）为条目，
// 遇到同级字段（Active Profile: / Ports:）即结束。
func ParseProfiles(data []byte, mac string) (CardProfiles, error) {
	block, ok := cardBlock(data, mac)
	if !ok {
		return CardProfiles{}, fmt.Errorf("未找到 %s 的蓝牙卡", mac)
	}
	var out CardProfiles
	inProfiles := false
	for _, line := range strings.Split(block, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case trimmed == "Profiles:":
			inProfiles = true
			continue
		case strings.HasPrefix(trimmed, "Active Profile:"):
			_, v, _ := strings.Cut(trimmed, ":")
			out.Active = strings.TrimSpace(v)
			inProfiles = false
			continue
		}
		if !inProfiles || trimmed == "" {
			continue
		}
		// 条目缩进更深（两个 tab）；缩进变浅说明 Profiles 段已结束
		if !strings.HasPrefix(line, "\t\t") {
			inProfiles = false
			continue
		}
		name, rest, ok := strings.Cut(trimmed, ":")
		if !ok || name == "" {
			continue
		}
		if strings.Contains(rest, "available: yes") {
			out.Available = append(out.Available, name)
		}
	}
	if out.Active == "" {
		return CardProfiles{}, fmt.Errorf("未找到 %s 的 Active Profile", mac)
	}
	return out, nil
}

// ParseActiveProfile 从 `pactl list cards` 输出取指定蓝牙卡的 Active Profile。
func ParseActiveProfile(data []byte, mac string) (string, error) {
	p, err := ParseProfiles(data, mac)
	if err != nil {
		return "", err
	}
	return p.Active, nil
}

// ActiveProfile 查询指定设备的蓝牙音频 profile。
func ActiveProfile(mac string) (string, error) {
	p, err := Profiles(mac)
	if err != nil {
		return "", err
	}
	return p.Active, nil
}

// Profiles 查询指定蓝牙卡的档位状态。
func Profiles(mac string) (CardProfiles, error) {
	out, err := pactl("10", "list", "cards")
	if err != nil {
		return CardProfiles{}, err
	}
	return ParseProfiles(out, mac)
}

// SetCardProfile 切换蓝牙卡的音频档位。
func SetCardProfile(mac, profile string) error {
	_, err := pactl("10", "set-card-profile", CardName(mac), profile)
	return err
}

// IsA2DP 判断档位是否为 A2DP 播放档。
// 非 A2DP 即 HSP/HFP 通话档（mono 16kHz 低音质），或 off。
func IsA2DP(profile string) bool {
	return strings.HasPrefix(profile, "a2dp-sink")
}

// PickA2DPProfile 从可用档位中挑优先级最高的 A2DP 档。
// 优先级：a2dp-sink(AAC) > a2dp-sink-sbc_xq > a2dp-sink-sbc > 其他 a2dp-sink*。
// 无 A2DP 档时返回 false。
func PickA2DPProfile(profiles []string) (string, bool) {
	for _, want := range []string{"a2dp-sink", "a2dp-sink-sbc_xq", "a2dp-sink-sbc"} {
		for _, p := range profiles {
			if p == want {
				return p, true
			}
		}
	}
	for _, p := range profiles {
		if IsA2DP(p) {
			return p, true
		}
	}
	return "", false
}
