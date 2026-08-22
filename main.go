// main.go
package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"blue-select/internal/audio"
	"blue-select/internal/bluetooth"
	"blue-select/internal/config"
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
func runWatch() error   { return errors.New("watch: not implemented") }
func runInstall() error { return errors.New("install: not implemented") }
