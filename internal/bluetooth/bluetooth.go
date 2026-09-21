// internal/bluetooth/bluetooth.go
package bluetooth

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
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

// Trust 标记设备为可信（trusted 设备会被 bluetoothd 自动回连）。
func Trust(mac string) error {
	out, err := exec.Command("timeout", "10", "bluetoothctl", "trust", mac).CombinedOutput()
	msg := strings.TrimSpace(string(out))
	if err != nil {
		return fmt.Errorf("trust %s: %w: %s", mac, err, msg)
	}
	if strings.Contains(msg, "Failed") {
		return fmt.Errorf("trust %s: %s", mac, msg)
	}
	return nil
}

// lastMeaningfulLine 取最后一行非空、非事件行（不以 [ 开头）的输出，用于错误摘要。
func lastMeaningfulLine(s string) string {
	lines := strings.Split(s, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		l := strings.TrimSpace(lines[i])
		if l == "" || strings.HasPrefix(l, "[") {
			continue
		}
		return l
	}
	return ""
}

// PairResult 汇总一次配对会话的结果。
// Found=false 表示扫描窗口内根本没看到该 MAC（耳机未广播）。
type PairResult struct {
	Found  bool
	Paired bool
}

// bluezSession 是一个长驻 bluetoothctl 进程，支持实时读取输出并自动应答
// `(yes/no)` 确认提示（passkey / 服务授权）。
//
// 非 tty 下 bluetoothctl 会刷事件行，且提示不带换行（如 "(yes/no): "），
// 因此按字节块累积再匹配，不做逐行扫描。
type bluezSession struct {
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.WriteCloser
	chunks   chan string
	acc      strings.Builder
	answered int
	logf     func(format string, a ...any)
}

func newBluezSession(logf func(format string, a ...any)) (*bluezSession, error) {
	stdinR, stdinW, err := os.Pipe()
	if err != nil {
		return nil, err
	}
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		stdinW.Close()
		return nil, err
	}
	cmd := exec.Command("bluetoothctl")
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdinR, stdoutW, stdoutW
	if err := cmd.Start(); err != nil {
		stdinW.Close()
		stdoutW.Close()
		return nil, fmt.Errorf("启动 bluetoothctl: %w", err)
	}
	s := &bluezSession{cmd: cmd, stdin: stdinW, stdout: stdoutW, chunks: make(chan string, 64), logf: logf}
	go func() {
		defer close(s.chunks)
		buf := make([]byte, 4096)
		for {
			n, err := stdoutR.Read(buf)
			if n > 0 {
				select {
				case s.chunks <- string(buf[:n]):
				default: // 消费不及时则丢弃，避免阻塞读取
				}
			}
			if err != nil {
				return
			}
		}
	}()
	return s, nil
}

func (s *bluezSession) send(cmd string) { io.WriteString(s.stdin, cmd+"\n") }

func (s *bluezSession) close() {
	io.WriteString(s.stdin, "quit\n")
	s.stdout.Close()
	s.cmd.Process.Kill()
	_ = s.cmd.Wait()
}

// waitFor 累积输出直到 pred 为真或超时。返回累积文本与是否命中。
// 期间自动应答每个 (yes/no) 提示一次 yes。
func (s *bluezSession) waitFor(timeout time.Duration, pred func(text string) bool) (string, bool, error) {
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for {
		select {
		case c, ok := <-s.chunks:
			if !ok {
				return s.acc.String(), false, fmt.Errorf("bluetoothctl 会话意外结束")
			}
			s.acc.WriteString(c)
			text := s.acc.String()
			if len(text) > 256*1024 { // 防无限增长：只保留尾部
				tail := text[len(text)-32*1024:]
				s.acc.Reset()
				s.acc.WriteString(tail)
				text = tail
			}
			// 每个 (yes/no) 提示应答一次 yes（提示可能跨 chunk，故对累积文本计数）
			if n := strings.Count(text, "(yes/no)"); n > s.answered {
				for i := s.answered; i < n; i++ {
					s.send("yes")
				}
				s.answered = n
				s.logf("已应答配对确认（passkey / 服务授权）")
			}
			if pred(text) {
				return text, true, nil
			}
		case <-deadline.C:
			return s.acc.String(), false, nil
		}
	}
}

// Pair 重新配对设备：断开 → 扫描确认耳机在配对模式 → 删旧记录 → 同会话扫描+配对。
//
// 为什么先扫描再删记录：删记录会破坏现有配对。若耳机没进配对模式就先把记录删了，
// 用户会从「配对损坏」变成「彻底没配对」。所以扫描不到就原样返回，什么都不动。
//
// 为什么扫描与 pair 必须同一会话：bluez 会丢弃非活跃扫描期间发现的未配对设备，
// 扫描进程结束后设备会从 `bluetoothctl devices` 消失，后续 pair 报
// `Device <MAC> not available`。
//
// 为什么要实时读输出：配对会弹 `[agent] Confirm passkey NNNNNN (yes/no):`，
// 必须应答 yes。用管道批量喂命令时这条提示会把下一条命令当成答案吃掉，
// 表现为 `Failed to pair: org.bluez.Error.AuthenticationFailed`（看着像配对失败，
// 实为无人应答确认）。
//
// 配对成功后由调用方另行 Trust/Connect——设备此时已持久化，无需留在会话内。
// logf 可为 nil。
func Pair(mac string, scanTimeout time.Duration, logf func(format string, a ...any)) (PairResult, error) {
	var res PairResult
	if logf == nil {
		logf = func(string, ...any) {}
	}

	// 连着的设备不广播，进不了配对模式
	_ = Disconnect(mac)

	// 阶段 1：确认耳机在配对模式。此时不动现有配对记录。
	s1, err := newBluezSession(logf)
	if err != nil {
		return res, err
	}
	s1.send("scan on")
	logf("扫描中（最多 %s）——请让耳机进入配对模式", scanTimeout)
	_, hit, err := s1.waitFor(scanTimeout, func(t string) bool { return strings.Contains(t, mac) })
	s1.close()
	if err != nil {
		return res, err
	}
	if !hit {
		return res, fmt.Errorf("扫描 %s 未发现 %s：确认耳机已进入配对模式、未连着手机，且在本机蓝牙范围内（现有配对记录未改动）", scanTimeout, mac)
	}
	res.Found = true
	logf("发现 %s", mac)

	// 阶段 2：删掉旧记录——损坏的配对记录（br-connection-key-missing）正是重配的主因
	if out, _ := exec.Command("timeout", "10", "bluetoothctl", "remove", mac).CombinedOutput(); strings.Contains(string(out), "removed") {
		logf("已删除旧配对记录 %s", mac)
	}

	// 阶段 3：同一会话内 扫描 + 配对
	s2, err := newBluezSession(logf)
	if err != nil {
		return res, err
	}
	defer s2.close()
	s2.send("scan on")
	pairSent := false
	text, _, err := s2.waitFor(scanTimeout, func(t string) bool {
		if !pairSent && strings.Contains(t, mac) {
			pairSent = true
			s2.send("pair " + mac)
			logf("开始配对...")
		}
		return strings.Contains(t, "Pairing successful") || strings.Contains(t, "Failed to pair")
	})
	if err != nil {
		return res, err
	}
	switch {
	case strings.Contains(text, "Pairing successful"):
		res.Paired = true
		logf("配对成功")
		return res, nil
	case strings.Contains(text, "Failed to pair"):
		return res, fmt.Errorf("配对失败: %s", lastMeaningfulLine(text))
	default:
		return res, fmt.Errorf("已发现 %s 但 %s 内配对未完成", mac, scanTimeout)
	}
}
