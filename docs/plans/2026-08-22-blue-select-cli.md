# blue-select CLI Implementation Plan

> **For agentic workers:** 按本计划任务逐项实施。步骤用 checkbox（`- [ ]`）跟踪。

**Goal:** Go CLI 工具，一键连接蓝牙耳机并把音频输出切到耳机（connect / status / watch / install 四个子命令），解决「连接成功但默认输出没切」「唤醒后失联要手动连」两个高频痛点。

**Architecture:** 纯 stdlib Go，exec 调 `bluetoothctl`（连接管理）与 `pactl`（音频路由）。三个内部包：`config`（设备映射）、`bluetooth`（bluetoothctl 封装）、`audio`（pactl 封装）。watch 模式监听 `pactl subscribe` 事件流，新 bluez sink 出现即自动设默认并迁移播放流；install 注册 systemd user 服务实现常驻。

**Tech Stack:** Go 1.22+，零第三方依赖。外部依赖：bluez（bluetoothctl）、PipeWire/PulseAudio（pactl）、systemd user session。

**Spec:** `~/prompt_boilerplates/System_Fix/bluetooth-pairing-troubleshoot.md`（v1.2.0，命令来源）+ 本文件「实测事实」节（2026-08-22 本机验证记录）。

## 实测事实（执行者必读，2026-08-22 本机验证）

| 事实 | 值 |
|---|---|
| 目标设备 | HUAWEI FreeArc，MAC `30:96:10:FD:B6:88`，已 Paired + Trusted |
| sink 命名 | MAC 冒号换下划线 + `.1`：`bluez_output.30_96_10_FD_B6_88.1` |
| 连接命令 | `timeout 30 bluetoothctl connect <MAC>`，成功输出含 `Connection successful` |
| 音频切换 | `pactl set-default-sink <sink>` + 逐个 `pactl move-sink-input <id> <sink>` |
| sink 列表 | `pactl list sinks short`，制表符分隔：`ID  Name  Driver  Spec  Channels  State` |
| 播放流列表 | `pactl list sink-inputs short`，首列 ID |
| 默认输出 | `pactl get-default-sink` 输出单行 sink 名 |
| 事件流 | `pactl subscribe` 持续输出，sink 相关行含 `on sink` |
| 电量 | `bluetoothctl info <MAC>` 行格式 `Battery Percentage: 0x46 (70)`，括号内为百分数；无电池设备无此行 |
| 已知设备列表 | `bluetoothctl devices` 行格式 `Device <MAC> <Name>`，Name 可含空格 |
| profile | `pactl list cards` 中含 `bluez_card.<MAC>` 的块内 `Active Profile:` 行；音乐应为 `a2dp-sink` |
| 静音陷阱 | 内置扬声器 MUTED 与耳机无关；判断耳机静音只看 bluez sink 自身 `Mute:` 字段 |

## Global Constraints

- Go 标准库 only，禁止第三方依赖
- 所有外部命令调用必须包 `timeout`（bluetoothctl 30s / pactl 10s），防非 tty 挂起
- bluetoothctl 在非 tty 下会刷事件行（`[NEW]`/`[CHG]` 开头），状态判断只看解析结果，不看命令回显
- 用户可见文本用中文；错误输出到 stderr，退出码非 0；正常输出到 stdout
- 配置文件 `~/.config/blue-select/config.json`，权限 0644（无凭据，不涉密）
- 质量关卡（development-quality-gates）：`go vet ./...` 干净、`go test ./...` 全绿、完成后部署 `~/.local/bin/blue-select`（关卡 11）
- 平台限定：Linux + bluez + PipeWire（pactl 兼容 PulseAudio）
- 每个任务完成后 `git commit`；Task 1 含 `git init`
- 纯子命令设计，无交互式 TUI/菜单（豁免 interactive-cli-design.md 全套 PTY 验收）

## 前置实验（可选，不阻塞本计划）

PipeWire 的 pipewire-pulse 支持 `module-switch-on-connect`，可在新 sink 出现时自动切默认输出。执行者可在开工前花 10 分钟试验该配置。**无论实验结果如何都继续本计划**：connect/status/install 有独立价值；即使模块生效，watch 仍负责 `move-sink-input` 迁移已有播放流（模块不保证迁流）并提供日志可观测性。

## 文件结构

```text
blue-select/
├── go.mod
├── main.go                      # 子命令分发
├── main_test.go                 # ResolveDevice 纯函数测试
├── internal/
│   ├── config/config.go         # 配置加载/保存
│   ├── config/config_test.go
│   ├── bluetooth/bluetooth.go   # bluetoothctl 封装
│   ├── bluetooth/bluetooth_test.go
│   ├── bluetooth/testdata/      # 命令输出 fixture
│   ├── audio/audio.go           # pactl 封装
│   └── audio/audio_test.go
├── docs/plans/                  # 本文件
├── README.md                    # Task 8 创建
└── CONTEXT_FOR_NEXT_AGENT.md    # Task 8 更新
```

---

### Task 1: 项目脚手架 + config 包

**Files:**
- Create: `go.mod`
- Create: `main.go`
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Consumes: 无（首任务）
- Produces: `config.Config` 结构体、`config.Load() (*Config, error)`、`config.Save(c *Config) error`、`main.go` 四个命令入口函数签名 `runConnect(name string) error` / `runStatus() error` / `runWatch() error` / `runInstall() error`（本任务为桩实现）

- [ ] **Step 1: 初始化项目**

```bash
cd ~/Desktop/go-projects/blue-select
git init
go mod init blue-select
```

- [ ] **Step 2: 写 config 包（先写实现，配置逻辑简单，测试覆盖即可）**

```go
// internal/config/config.go
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config 保存设备名到 MAC 的映射与默认设备。
type Config struct {
	DefaultDevice string            `json:"default_device"`
	Devices       map[string]string `json:"devices"` // name -> MAC
}

func configPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "blue-select", "config.json"), nil
}

// Load 读取配置。文件不存在时返回空配置，不报错。
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{Devices: map[string]string{}}, nil
	}
	if err != nil {
		return nil, err
	}
	c := &Config{}
	if err := json.Unmarshal(data, c); err != nil {
		return nil, err
	}
	if c.Devices == nil {
		c.Devices = map[string]string{}
	}
	return c, nil
}

// Save 写入配置，目录不存在则创建，权限 0644。
func Save(c *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
```

- [ ] **Step 3: 写 config 测试**

```go
// internal/config/config_test.go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.DefaultDevice != "" || len(c.Devices) != 0 {
		t.Fatalf("want empty config, got %+v", c)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	want := &Config{
		DefaultDevice: "HUAWEI FreeArc",
		Devices:       map[string]string{"HUAWEI FreeArc": "30:96:10:FD:B6:88"},
	}
	if err := Save(want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	info, err := os.Stat(filepath.Join(dir, "blue-select", "config.json"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("perm = %v, want 0644", info.Mode().Perm())
	}
	got, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.DefaultDevice != want.DefaultDevice || got.Devices["HUAWEI FreeArc"] != want.Devices["HUAWEI FreeArc"] {
		t.Fatalf("roundtrip mismatch: got %+v", got)
	}
}
```

- [ ] **Step 4: 写 main.go 分发骨架（桩实现）**

```go
// main.go
package main

import (
	"errors"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "connect":
		name := ""
		if len(os.Args) > 2 {
			name = os.Args[2]
		}
		err = runConnect(name)
	case "status":
		err = runStatus()
	case "watch":
		err = runWatch()
	case "install":
		err = runInstall()
	case "-h", "--help", "help":
		usage()
	default:
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "blue-select:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprint(os.Stdout, `blue-select — 蓝牙耳机一键连接与音频切换

用法:
  blue-select connect [设备名]   连接设备并切换音频输出（无参用默认设备）
  blue-select status             显示连接、电量、profile、输出状态
  blue-select watch              监听新蓝牙 sink，自动设默认并迁移播放流
  blue-select install            安装并启用 watch 的 systemd user 服务
`)
}

// 以下为桩实现，Task 4-7 逐个替换。

func runConnect(name string) error { return errors.New("connect: not implemented") }
func runStatus() error             { return errors.New("status: not implemented") }
func runWatch() error              { return errors.New("watch: not implemented") }
func runInstall() error            { return errors.New("install: not implemented") }
```

- [ ] **Step 5: 验证并提交**

```bash
go vet ./... && go test ./...
./$(go build -o /tmp/blue-select .) 2>&1; echo "exit=$?"   # 期望 usage + exit=2
git add -A && git commit -m "feat: scaffold + config package"
```

Expected: vet 干净、测试 PASS、无参运行打印 usage 且退出码 2。

---

### Task 2: bluetooth 包（bluetoothctl 封装）

**Files:**
- Create: `internal/bluetooth/bluetooth.go`
- Create: `internal/bluetooth/testdata/devices.txt`
- Create: `internal/bluetooth/testdata/info_connected.txt`
- Create: `internal/bluetooth/testdata/info_disconnected.txt`
- Test: `internal/bluetooth/bluetooth_test.go`

**Interfaces:**
- Consumes: 无
- Produces: `bluetooth.Device{MAC, Name string}`、`bluetooth.Info{Connected, Paired, Trusted bool, Battery int, HasBattery bool}`、`bluetooth.ListDevices() ([]Device, error)`、`bluetooth.Info_(mac string) (Info, error)`、`bluetooth.Connect(mac string) error`、`bluetooth.ParseDevices(data []byte) ([]Device, error)`、`bluetooth.ParseInfo(data []byte) (Info, error)`（Parse 函数导出供测试直接喂 fixture）

- [ ] **Step 1: 创建 fixture 文件**

`internal/bluetooth/testdata/devices.txt`（逐字复制，含真实设备与干扰项）：

```text
Device 30:96:10:FD:B6:88 HUAWEI FreeArc
Device A4:C1:38:11:22:33 Mi Band 7
Device 5F:2A:90:BC:DE:01 Random LE Device
```

`internal/bluetooth/testdata/info_connected.txt`（截取真实输出关键字段）：

```text
Device 30:96:10:FD:B6:88 (public)
	Name: HUAWEI FreeArc
	Alias: HUAWEI FreeArc
	Paired: yes
	Trusted: yes
	Blocked: no
	Connected: yes
	Modalias: bluetooth:v02B0p0000d001F
	Battery Percentage: 0x46 (70)
```

`internal/bluetooth/testdata/info_disconnected.txt`：

```text
Device 30:96:10:FD:B6:88 (public)
	Name: HUAWEI FreeArc
	Paired: yes
	Trusted: yes
	Blocked: no
	Connected: no
	Modalias: bluetooth:v02B0p0000d001F
```

- [ ] **Step 2: 写解析函数的失败测试**

```go
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
```

- [ ] **Step 3: 跑测试确认失败**

Run: `go test ./internal/bluetooth/ -v`
Expected: FAIL，`ParseDevices undefined`

- [ ] **Step 4: 写实现**

```go
// internal/bluetooth/bluetooth.go
package bluetooth

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Device 是 bluetoothctl devices 里的一条已知设备。
type Device struct {
	MAC  string
	Name string
}

// Info 是 bluetoothctl info 的关键字段。
type Info struct {
	Connected  bool
	Paired     bool
	Trusted    bool
	Battery    int
	HasBattery bool
}

// ParseDevices 解析 `bluetoothctl devices` 输出。行格式: Device <MAC> <Name...>
func ParseDevices(data []byte) ([]Device, error) {
	var devs []Device
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "Device ") {
			continue
		}
		rest := strings.TrimPrefix(line, "Device ")
		mac, name, ok := strings.Cut(rest, " ")
		if !ok || mac == "" {
			continue
		}
		devs = append(devs, Device{MAC: mac, Name: name})
	}
	if len(devs) == 0 {
		return nil, fmt.Errorf("bluetoothctl devices: 无设备或输出无法解析")
	}
	return devs, nil
}

// ParseInfo 解析 `bluetoothctl info <MAC>` 输出。
func ParseInfo(data []byte) (Info, error) {
	var info Info
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(line, "Connected:"):
			info.Connected = fieldYes(line)
		case strings.HasPrefix(line, "Paired:"):
			info.Paired = fieldYes(line)
		case strings.HasPrefix(line, "Trusted:"):
			info.Trusted = fieldYes(line)
		case strings.HasPrefix(line, "Battery Percentage:"):
			// 格式: Battery Percentage: 0x46 (70)
			if i := strings.LastIndex(line, "("); i >= 0 {
				if n, err := strconv.Atoi(strings.Trim(line[i+1:], ") ")); err == nil {
					info.Battery, info.HasBattery = n, true
				}
			}
		}
	}
	return info, nil
}

func fieldYes(line string) bool {
	_, v, _ := strings.Cut(line, ":")
	return strings.TrimSpace(v) == "yes"
}

func run(args ...string) ([]byte, error) {
	out, err := exec.Command("timeout", "30", "bluetoothctl"). // 占位，见下方 Connect/ListDevices/Info_
		Output()
	return out, err
}

// ListDevices 返回已知设备。
func ListDevices() ([]Device, error) {
	out, err := exec.Command("timeout", "10", "bluetoothctl", "devices").Output()
	if err != nil {
		return nil, fmt.Errorf("bluetoothctl devices: %w", err)
	}
	return ParseDevices(out)
}

// Info_ 查询设备状态。（命名 Info_ 避免与类型 Info 冲突）
func Info_(mac string) (Info, error) {
	out, err := exec.Command("timeout", "10", "bluetoothctl", "info", mac).Output()
	if err != nil {
		return Info{}, fmt.Errorf("bluetoothctl info %s: %w", mac, err)
	}
	return ParseInfo(out)
}

// Connect 连接设备。失败时带回显摘要。
func Connect(mac string) error {
	out, err := exec.Command("timeout", "30", "bluetoothctl", "connect", mac).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("connect %s: %w: %s", mac, err, msg)
	}
	if strings.Contains(msg, "Failed") {
		return fmt.Errorf("connect %s: %s", mac, msg)
	}
	return nil
}
```

**注意**：上面 `run` 占位函数是草稿残留，实现时**删除**，只保留 `ListDevices` / `Info_` / `Connect` 三个入口。

- [ ] **Step 5: 跑测试确认通过**

Run: `go vet ./... && go test ./internal/bluetooth/ -v`
Expected: PASS

- [ ] **Step 6: 提交**

```bash
git add -A && git commit -m "feat: bluetooth package (bluetoothctl wrapper + parsers)"
```

---

### Task 3: audio 包（pactl 封装）

**Files:**
- Create: `internal/audio/audio.go`
- Create: `internal/audio/testdata/sinks.txt`
- Create: `internal/audio/testdata/sink_inputs.txt`
- Create: `internal/audio/testdata/cards.txt`
- Test: `internal/audio/audio_test.go`

**Interfaces:**
- Consumes: 无
- Produces: `audio.Sink{ID, Name, State string}`、`audio.Sinks() ([]Sink, error)`、`audio.BluezSink() (Sink, bool)`、`audio.Default() (string, error)`、`audio.SetDefault(name string) error`、`audio.MoveAllStreams(sinkName string) (int, error)`、`audio.WaitBluez(timeout time.Duration) (Sink, error)`、`audio.ActiveProfile(mac string) (string, error)`、`audio.ParseSinks(data []byte) ([]Sink, error)`、`audio.ParseActiveProfile(data []byte, mac string) (string, error)`（Parse 导出供 fixture 测试）

- [ ] **Step 1: 创建 fixture**

`internal/audio/testdata/sinks.txt`：

```text
49	alsa_output.pci-0000_00_1f.3.analog-stereo	PipeWire	s16le 2ch 44100Hz	SUSPENDED
3241	bluez_output.30_96_10_FD_B6_88.1	PipeWire	s16le 2ch 48000Hz	RUNNING
```

`internal/audio/testdata/sink_inputs.txt`：

```text
3149	49	3148	PipeWire	float32le 2ch 44100Hz
```

`internal/audio/testdata/cards.txt`（真实结构：两个卡，bluez 卡含 Active Profile）：

```text
Card #0
	Name: alsa_card.pci-0000_00_1f.3
	Active Profile: output:analog-stereo+input:analog-stereo

Card #3239
	Name: bluez_card.30_96_10_FD_B6_88
	Active Profile: a2dp-sink
```

- [ ] **Step 2: 写失败测试**

```go
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
```

- [ ] **Step 3: 跑测试确认失败**

Run: `go test ./internal/audio/ -v`
Expected: FAIL，`ParseSinks undefined`

- [ ] **Step 4: 写实现**

```go
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

// ParseSinks 解析 `pactl list sinks short`。行格式: ID\tName\tDriver\tSpec\tChannels\tState
func ParseSinks(data []byte) ([]Sink, error) {
	var sinks []Sink
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimRight(line, "\r")
		if line == "" {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) < 6 {
			continue
		}
		sinks = append(sinks, Sink{ID: f[0], Name: f[1], State: f[5]})
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
	needle := "bluez_card." + mac
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
```

- [ ] **Step 5: 跑测试确认通过**

Run: `go vet ./... && go test ./internal/audio/ -v`
Expected: PASS（`TestWaitBluezTimeout` 在本机无蓝牙 sink 连接时也 PASS——若当时恰好连着耳机，该测试会失败，属预期，跳过即可：`go test ./internal/audio/ -run 'TestParse|TestWaitBluez' -v` 时排除或临时断开设备）

- [ ] **Step 6: 提交**

```bash
git add -A && git commit -m "feat: audio package (pactl wrapper + parsers)"
```

---

### Task 4: connect 命令

**Files:**
- Modify: `main.go`（替换 `runConnect` 桩）
- Test: `main_test.go`

**Interfaces:**
- Consumes: Task 1 `config.Load/Save`、Task 2 `bluetooth.ListDevices/Info_/Connect`、Task 3 `audio.WaitBluez/SetDefault/MoveAllStreams`
- Produces: `ResolveDevice(cfg *config.Config, arg string, devs []bluetooth.Device) (mac string, err error)`（main 包纯函数，main_test.go 测试）；connect 成功时若设备是首次连接则写入 `cfg.Devices` 并设 `cfg.DefaultDevice`，`Save` 持久化

- [ ] **Step 1: 写 ResolveDevice 失败测试**

```go
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
```

- [ ] **Step 2: 跑测试确认失败**

Run: `go test . -v`
Expected: FAIL，`ResolveDevice undefined`

- [ ] **Step 3: 实现 ResolveDevice 与 runConnect**

在 `main.go` 中追加（并删除 `runConnect` 桩）：

```go
// ResolveDevice 把用户参数解析成 MAC。
// 优先级：参数子串匹配（大小写不敏感）> 配置默认设备 > 报错并列出已知设备。
func ResolveDevice(cfg *config.Config, arg string, devs []bluetooth.Device) (string, error) {
	if arg != "" {
		lower := strings.ToLower(arg)
		var hits []bluetooth.Device
		for _, d := range devs {
			if strings.Contains(strings.ToLower(d.Name), lower) || strings.Contains(strings.ToLower(d.MAC), lower) {
				hits = append(hits, d)
			}
		}
		switch len(hits) {
		case 1:
			return hits[0].MAC, nil
		case 0:
			return "", fmt.Errorf("已知设备中无匹配 %q（可用: %s）", arg, deviceNames(devs))
		default:
			return "", fmt.Errorf("%q 匹配到多台设备: %s，请用更精确的名字", arg, deviceNames(hits))
		}
	}
	if cfg.DefaultDevice != "" {
		for _, d := range devs {
			if d.Name == cfg.DefaultDevice {
				return d.MAC, nil
			}
		}
		return "", fmt.Errorf("默认设备 %q 不在已知列表（可用: %s）", cfg.DefaultDevice, deviceNames(devs))
	}
	return "", fmt.Errorf("请指定设备名（可用: %s），或先用 connect <名字> 建立默认", deviceNames(devs))
}

func deviceNames(devs []bluetooth.Device) string {
	names := make([]string, len(devs))
	for i, d := range devs {
		names[i] = d.Name
	}
	return strings.Join(names, ", ")
}

func runConnect(name string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	devs, err := bluetooth.ListDevices()
	if err != nil {
		return err
	}
	mac, err := ResolveDevice(cfg, name, devs)
	if err != nil {
		return err
	}
	devName := name
	for _, d := range devs {
		if d.MAC == mac {
			devName = d.Name
		}
	}

	info, err := bluetooth.Info_(mac)
	if err != nil {
		return err
	}
	if !info.Connected {
		fmt.Printf("连接 %s (%s)...\n", devName, mac)
		if err := bluetooth.Connect(mac); err != nil {
			return err
		}
	}

	sink, err := audio.WaitBluez(15 * time.Second)
	if err != nil {
		return fmt.Errorf("%w（设备可能连上了但音频链路未就绪，检查手机端是否抢占）", err)
	}
	if cur, _ := audio.Default(); cur != sink.Name {
		if err := audio.SetDefault(sink.Name); err != nil {
			return err
		}
	}
	moved, _ := audio.MoveAllStreams(sink.Name)

	// 首次连接记录 + 设默认
	if _, ok := cfg.Devices[devName]; !ok || cfg.DefaultDevice != devName {
		cfg.Devices[devName] = mac
		cfg.DefaultDevice = devName
		if err := config.Save(cfg); err != nil {
			fmt.Fprintln(os.Stderr, "警告: 配置保存失败:", err)
		}
	}

	info, _ = bluetooth.Info_(mac)
	fmt.Printf("✓ %s 已连接，输出已切换到 %s（迁移 %d 个播放流）\n", devName, sink.Name, moved)
	if info.HasBattery {
		fmt.Printf("  电量: %d%%\n", info.Battery)
	}
	return nil
}
```

`main.go` 顶部 import 需补：`"time"`、`"blue-select/internal/audio"`、`"blue-select/internal/bluetooth"`、`"blue-select/internal/config"`。

- [ ] **Step 4: 跑测试确认通过**

Run: `go vet ./... && go test ./... -v`
Expected: PASS

- [ ] **Step 5: 真机冒烟**

```bash
go build -o /tmp/blue-select . 
/tmp/blue-select connect freearc
/tmp/blue-select status 2>/dev/null || true   # status 未实现，报 not implemented 属预期
pactl get-default-sink   # 期望 bluez_output.30_96_10_FD_B6_88.1
```

Expected: connect 打印「已连接，输出已切换」，`pactl get-default-sink` 指向耳机。若耳机当前连着手机，先在手机端断开（设备连手机时不广播、可能拒绝连接）。

- [ ] **Step 6: 提交**

```bash
git add -A && git commit -m "feat: connect command with device resolution and auto sink switch"
```

---

### Task 5: status 命令

**Files:**
- Modify: `main.go`（替换 `runStatus` 桩）
- Test: `main_test.go`（status 无新纯函数，解析已测；本任务只加一条冒烟断言可省——解析测试已在 Task 2/3 覆盖）

**Interfaces:**
- Consumes: Task 2 `bluetooth.ListDevices/Info_`、Task 3 `audio.Sinks/Default/ActiveProfile`
- Produces: `runStatus() error`（无返回值契约，stdout 输出状态报告）

- [ ] **Step 1: 实现 runStatus**

替换 `main.go` 中 `runStatus` 桩：

```go
func runStatus() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	devs, err := bluetooth.ListDevices()
	if err != nil {
		return err
	}
	fmt.Println("== 蓝牙设备 ==")
	for _, d := range devs {
		info, err := bluetooth.Info_(d.MAC)
		if err != nil {
			continue
		}
		if !info.Paired && !info.Connected {
			continue // 只显示有意义的设备
		}
		state := "未连接"
		if info.Connected {
			state = "已连接"
		}
		line := fmt.Sprintf("  %s [%s]", d.Name, state)
		if info.HasBattery && info.Connected {
			line += fmt.Sprintf("  电量 %d%%", info.Battery)
		}
		if d.Name == cfg.DefaultDevice {
			line += "  (默认)"
		}
		fmt.Println(line)
		if info.Connected {
			if p, err := audio.ActiveProfile(d.MAC); err == nil {
				fmt.Printf("    profile: %s\n", p)
			}
		}
	}

	fmt.Println("== 音频输出 ==")
	def, _ := audio.Default()
	sinks, err := audio.Sinks()
	if err != nil {
		return err
	}
	for _, s := range sinks {
		marker := " "
		if s.Name == def {
			marker = "*"
		}
		fmt.Printf(" %s [%s] %s (%s)\n", marker, s.ID, s.Name, s.State)
	}
	return nil
}
```

- [ ] **Step 2: 构建验证**

Run: `go vet ./... && go test ./... && go build -o /tmp/blue-select . && /tmp/blue-select status`
Expected: 打印设备列表（FreeArc 已连接 + 电量 + profile: a2dp-sink）与 sink 列表（`*` 标记默认）。

- [ ] **Step 3: 提交**

```bash
git add -A && git commit -m "feat: status command"
```

---

### Task 6: watch 命令

**Files:**
- Modify: `main.go`（替换 `runWatch` 桩）

**Interfaces:**
- Consumes: Task 3 `audio.BluezSink/Default/SetDefault/MoveAllStreams`
- Produces: `runWatch() error`——常驻进程，stdout 打印带时间戳的切换日志，供 systemd journal 收集

- [ ] **Step 1: 实现 runWatch**

替换 `main.go` 中 `runWatch` 桩：

```go
func runWatch() error {
	cmd := exec.Command("pactl", "subscribe")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动 pactl subscribe: %w", err)
	}
	defer cmd.Process.Kill()
	logf := func(format string, a ...any) {
		fmt.Printf("%s %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, a...))
	}
	logf("watch 启动，监听音频事件...")

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// 只关心 sink 事件；"change" 也处理——sink 状态翻转（RUNNING/SUSPENDED）时默认输出可能被系统改走
		if !strings.Contains(line, "on sink") {
			continue
		}
		time.Sleep(300 * time.Millisecond) // 防抖：等 sink 注册完成
		s, ok := audio.BluezSink()
		if !ok {
			continue // 蓝牙 sink 消失（耳机断开），不动作
		}
		cur, err := audio.Default()
		if err == nil && cur == s.Name {
			continue // 已经是默认
		}
		if err := audio.SetDefault(s.Name); err != nil {
			logf("错误: 设默认失败: %v", err)
			continue
		}
		moved, _ := audio.MoveAllStreams(s.Name)
		logf("已切换默认输出到 %s（迁移 %d 个播放流）", s.Name, moved)
	}
	return scanner.Err()
}
```

`main.go` 顶部 import 需补：`"bufio"`、`"os/exec"`。

- [ ] **Step 2: 真机验证**

```bash
go build -o /tmp/blue-select .
/tmp/blue-select watch &   # 后台跑
sleep 1
bluetoothctl disconnect 30:96:10:FD:B6:88   # 断开耳机
sleep 2
bluetoothctl connect 30:96:10:FD:B6:88      # 重连
sleep 5
pactl get-default-sink    # 期望自动切回 bluez_output...
kill %1
```

Expected: watch 日志打印「已切换默认输出到 bluez_output...」，默认输出自动恢复耳机。若耳机连着手机导致重连失败，先断手机。

- [ ] **Step 3: 提交**

```bash
git add -A && git commit -m "feat: watch command (auto switch on new bluez sink)"
```

---

### Task 7: install 命令（systemd user 服务）

**Files:**
- Modify: `main.go`（替换 `runInstall` 桩）

**Interfaces:**
- Consumes: Task 6 `runWatch`
- Produces: `runInstall() error`——写 `~/.config/systemd/user/blue-select-watch.service`，`systemctl --user daemon-reload && systemctl --user enable --now blue-select-watch.service`，验证 active

- [ ] **Step 1: 实现 runInstall**

替换 `main.go` 中 `runInstall` 桩：

```go
const unitTemplate = `[Unit]
Description=blue-select watch: 蓝牙 sink 自动切换
After=pipewire.service wireplumber.service

[Service]
ExecStart=%s watch
Restart=on-failure
RestartSec=5

[Install]
WantedBy=default.target
`

func runInstall() error {
	bin, err := os.Executable()
	if err != nil {
		return err
	}
	bin, err = filepath.EvalSymlinks(bin)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	unitDir := filepath.Join(home, ".config", "systemd", "user")
	if err := os.MkdirAll(unitDir, 0o755); err != nil {
		return err
	}
	unitPath := filepath.Join(unitDir, "blue-select-watch.service")
	unit := fmt.Sprintf(unitTemplate, bin)
	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		return err
	}
	fmt.Println("已写入", unitPath)
	for _, args := range [][]string{
		{"--user", "daemon-reload"},
		{"--user", "enable", "--now", "blue-select-watch.service"},
	} {
		if out, err := exec.Command("systemctl", args...).CombinedOutput(); err != nil {
			return fmt.Errorf("systemctl %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(out)))
		}
	}
	// 验证
	out, err := exec.Command("systemctl", "--user", "is-active", "blue-select-watch.service").Output()
	state := strings.TrimSpace(string(out))
	if err != nil || state != "active" {
		return fmt.Errorf("服务未 active（当前: %s），用 journalctl --user -u blue-select-watch 查日志", state)
	}
	fmt.Println("✓ blue-select-watch 服务已启用并运行（active）")
	fmt.Println("  日志: journalctl --user -u blue-select-watch -f")
	return nil
}
```

`main.go` 顶部 import 需补：`"path/filepath"`（`strings` 已有）。

- [ ] **Step 2: 部署到 ~/.local/bin 后真机验证**

```bash
go build -trimpath -ldflags="-s -w" -o ~/.local/bin/blue-select .
blue-select install
systemctl --user status blue-select-watch --no-pager | head -5
```

Expected: 服务 active (running)。再断开/重连耳机验证守护生效（同 Task 6 Step 2，但无需手动跑 watch）。

- [ ] **Step 3: 提交**

```bash
git add -A && git commit -m "feat: install command (systemd user unit for watch)"
```

---

### Task 8: 文档 + 收尾验证

**Files:**
- Create: `README.md`
- Modify: `CONTEXT_FOR_NEXT_AGENT.md`（项目根已有骨架文件，更新为完成态）

**Interfaces:**
- Consumes: 全部前序任务
- Produces: 完成态文档

- [ ] **Step 1: 写 README.md**

内容必须包含：一句话定位、四个子命令用法示例（connect/status/watch/install）、配置文件路径与格式、systemd 服务管理命令（`journalctl --user -u blue-select-watch -f` / `systemctl --user restart blue-select-watch`）、构建安装命令（`go build -trimpath -ldflags="-s -w" -o ~/.local/bin/blue-select .`）、已知限制（设备连手机时不广播需先断手机；watch 只切 sink 不做重连，重连依赖 bluetoothd trusted 自动回连）。

- [ ] **Step 2: 更新 CONTEXT_FOR_NEXT_AGENT.md**

按 LLM-api-check 的 CONTEXT 格式：项目当前状态、最后一次完成的工作、遗留问题、技术要点（外部命令依赖、fixture 测试模式、`Info_` 命名原因、WaitBluezTimeout 测试在耳机连着时会失败的说明）。

- [ ] **Step 3: 终验（Gate Function，verification-before-completion）**

```bash
go vet ./... && go test ./... -v
blue-select --version 2>/dev/null || blue-select status   # 二进制可用
systemctl --user is-active blue-select-watch.service
bluetoothctl disconnect 30:96:10:FD:B6:88 && sleep 2 && blue-select connect
pactl get-default-sink
```

Expected: 全绿；connect 后默认输出为 `bluez_output.30_96_10_FD_B6_88.1`；服务 active。

- [ ] **Step 4: 提交**

```bash
git add -A && git commit -m "docs: README + CONTEXT"
```

---

## Self-Review 记录

1. **Spec 覆盖**：skill 步骤 6（pair/connect）→ Task 4；步骤 7（set-default + move + 验证）→ Task 4/6；日常维护（自动重连兜底、电量、profile）→ Task 5；一劳永逸 → Task 6/7。坑「连接成功≠声音切过去」「静音显示误导」→ Task 5 status 区分。✅
2. **占位符扫描**：无 TBD；Task 2 草稿中的 `run` 占位函数已显式标注「实现时删除」。README 内容以要点清单给出（文档类任务，非代码，允许要点式）。✅
3. **类型一致性**：`ResolveDevice(cfg *config.Config, arg string, devs []bluetooth.Device)` 在 Task 4 测试与实现签名一致；`audio.Sink{ID,Name,State}` 在 Task 3 定义、Task 4/5 使用字段一致；`bluetooth.Info_.HasBattery` Task 2 定义、Task 4 使用一致。✅
