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

// ParseActiveProfile 从 `pactl list cards` 输出取指定蓝牙卡的 Active Profile。
// 实现按 Card 块切分：块内同时含 bluez_card.<mac> 与 Active Profile: 即命中，
// 不依赖两行的先后顺序。
func ParseActiveProfile(data []byte, mac string) (string, error) {
	// 卡名用下划线格式（bluez_card.30_96_10_FD_B6_88），MAC 冒号需替换
	needle := "bluez_card." + strings.ReplaceAll(mac, ":", "_")
	for _, card := range bytes.Split(data, []byte("Card #")) {
		block := string(card)
		if !strings.Contains(block, needle) {
			continue
		}
		for _, line := range strings.Split(block, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "Active Profile:") {
				_, v, _ := strings.Cut(line, ":")
				return strings.TrimSpace(v), nil
			}
		}
	}
	return "", fmt.Errorf("未找到 %s 的蓝牙卡或 Active Profile", mac)
}

// ActiveProfile 查询指定设备的蓝牙音频 profile。
func ActiveProfile(mac string) (string, error) {
	out, err := pactl("10", "list", "cards")
	if err != nil {
		return "", err
	}
	return ParseActiveProfile(out, mac)
}
