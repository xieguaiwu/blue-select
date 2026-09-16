// internal/bluetooth/bluetooth.go
package bluetooth

import (
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// isTimeoutExit 判断 err 是否为 timeout 命令的 124 超时退出。
func isTimeoutExit(err error) bool {
	var ee *exec.ExitError
	if ok := errors.As(err, &ee); ok {
		return ee.ExitCode() == 124
	}
	return false
}

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
// bluetoothctl connect 在部分设备上连接成功后不退出（等待信号/卡住），
// timeout 124 强杀后设备实际已连上——此时回查 info 确认连接状态，连上即视为成功。
func Connect(mac string) error {
	out, err := exec.Command("timeout", "30", "bluetoothctl", "connect", mac).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	// 配对密钥丢失：原始输出对用户无意义，改为可操作的修复指引。
	if strings.Contains(msg, "br-connection-key-missing") {
		return fmt.Errorf("connect %s: 配对密钥丢失，需重新配对——先 `bluetoothctl remove %s`，再让耳机进入配对模式后重新配对", mac, mac)
	}
	if err != nil {
		if isTimeoutExit(err) {
			if info, ierr := Info_(mac); ierr == nil && info.Connected {
				return nil
			}
		}
		return fmt.Errorf("connect %s: %w: %s", mac, err, msg)
	}
	if strings.Contains(msg, "Failed") {
		return fmt.Errorf("connect %s: %s", mac, msg)
	}
	return nil
}

// Disconnect 断开设备。失败时带回显摘要。
func Disconnect(mac string) error {
	out, err := exec.Command("timeout", "30", "bluetoothctl", "disconnect", mac).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("disconnect %s: %w: %s", mac, err, msg)
	}
	if strings.Contains(msg, "Failed") {
		return fmt.Errorf("disconnect %s: %s", mac, msg)
	}
	return nil
}
