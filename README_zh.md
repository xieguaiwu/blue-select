# blue-select

[English](README.md) | **中文**

Linux 蓝牙耳机一键连接 + 音频输出自动切换工具（bluez + PipeWire/PulseAudio）。Go 编写，零第三方依赖。

## 解决什么问题

1. **连上了但声音还是从扬声器出**——`bluetoothctl connect` 不会切换 PulseAudio/PipeWire 默认输出。blue-select 等待蓝牙 sink 就绪后自动设为默认输出，并把已有播放流全部迁移过去。
2. **睡眠唤醒后要手动重连**——一条命令：`blue-select connect`。
3. **连上了但声音是单声道、音质差**——A2DP 传输失败后 bluez 会回落到 HSP/HFP 通话档（`headset-head-unit`，mono 16kHz），但设备仍显示已连接。blue-select 会校验当前音频档位并切回 A2DP；若卡上 A2DP 档已整个消失，则自动断连重连一次让 bluez 重新协商。

```bash
# 连接设备（名字模糊匹配）并切换音频输出
blue-select connect freearc

# 连接设备并切换到 HFP 档位（麦克风输入）
blue-select connect-mic freearc

# 重新配对设备（删旧记录、自动应答 passkey，然后连接）
blue-select pair freearc

# 断开设备；默认输出自动切回本地 sink
blue-select disconnect freearc

# 显示设备、电量、音频档位（非 A2DP 会告警）、sink 列表与默认输出
blue-select status

# 守护模式：监听新蓝牙 sink，自动切换默认输出
blue-select watch

# 把 watch 安装为 systemd user 服务
blue-select install
```

首次 `connect <名字>` 成功后会记录该设备并设为默认（`~/.config/blue-select/config.json`）。之后直接 `blue-select connect` 即可重连默认设备。

## 配置文件

`~/.config/blue-select/config.json`：

```json
{
  "default_device": "HUAWEI FreeArc",
  "devices": { "HUAWEI FreeArc": "AA:BB:CC:DD:EE:FF" }
}
```

## systemd 服务

```bash
systemctl --user status blue-select-watch
journalctl --user -u blue-select-watch -f      # 跟踪切换日志
systemctl --user restart blue-select-watch
```

unit 文件由 `blue-select install` 生成（ExecStart 指向执行 install 时的二进制路径——移动二进制后需重新执行 install）。

## 构建安装

依赖：Go 1.22+，Linux 且有 `bluetoothctl`（bluez）与 `pactl`（PipeWire pipewire-pulse 或 PulseAudio）。

```bash
go build -trimpath -ldflags="-s -w" -o ~/.local/bin/blue-select .
go test ./...
```

## 已知限制

- 耳机正连着手机时可能拒绝电脑连接（不广播/连接竞态），先在手机端断开。
- `watch` 负责音频路由切换，并只用低风险的 `set-card-profile` 修复音频档位；它不会重连耳机。重连依赖 bluetoothd 对 trusted 设备的自动回连。
- 仅提供非交互子命令，无 TUI。
- `connect-mic` 会切换到 HFP/HSP 档位（mono 16 kHz）以启用麦克风输入；需要音乐播放时用 `connect` 切回 A2DP。

## 故障排查

**报错「配对密钥丢失」（底层 `br-connection-key-missing`）**

配对记录损坏，须删除后重新配对：

```bash
blue-select pair freearc      # 删旧记录 + 自动应答 passkey + 重连，一气呵成
```

`pair` 先扫描、确认耳机真的在广播后才删旧记录——所以耳机没进配对模式时，现有配对不会被破坏。可选 `--scan-timeout=<秒>`（默认 90）调整扫描窗口。

手工做法：

```bash
bluetoothctl remove <MAC>
# 让耳机进入配对模式。FreeArc：双耳入盒 + 开盖 + 长按盒内功能键 2-5 秒，盒身白灯闪烁
```

配对时 bluez 要求确认 6 位配对码（`Confirm passkey 123456 (yes/no)`），须回答 `yes`。用管道喂命令时这条提示会吞掉你的下一条命令——改用能自动应答的脚本，或直接用 `blue-select pair`。详见 skill `bluetooth-pairing-troubleshoot`。

**扫描不到设备 / 连接偶发超时**

Intel 蓝牙适配器的 USB 自动挂起会导致此问题（内核日志：`hci0: Reading supported features failed (-16)`）。用脚本修复，需 sudo，立即生效且重启后保持：

```bash
sudo bash ~/Downloads/fix-bluetooth-usb-autosuspend.sh
```

**连上了，但声音是单声道、音质差（走 HSP/HFP 而非 A2DP）**

A2DP 传输失败后，bluez 会回落到 HSP/HFP 通话档（`headset-head-unit`，mono 16kHz），但设备仍显示已连接。`blue-select status` 会把它标为 `⚠ 非 A2DP`。重跑 `blue-select connect`：工具会把卡切回 `a2dp-sink`；若该档位已整个消失，则自动断连重连一次让 bluez 重新协商 A2DP。若档位仍不恢复，检查耳机是否被手机等其他主机占用、电量是否充足。

## 许可证

[MIT](LICENSE)
