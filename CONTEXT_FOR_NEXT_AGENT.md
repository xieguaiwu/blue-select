# CONTEXT_FOR_NEXT_AGENT.md

## 项目当前状态
blue-select — Go CLI，蓝牙耳机一键连接 + 音频输出自动切换（connect / disconnect / status / watch / install 五个子命令）。**开发完成 + 真机端到端验证通过（2026-09-16）：connect 2.93s exit 0、disconnect 2.64s exit 0、watch 自动切换有日志证据、单测全绿（4 包）、go vet 干净、已部署 `~/.local/bin/blue-select`、blue-select-watch 服务 active。** 仅剩一项需用户 root 执行的 USB 自动挂起永久修复（见遗留问题）。

## 最后一次完成的工作（2026-09-16）
真机回归暴露并修复「连接超时误报失败」——用户报告 `blue-select connect FreeArc` 卡 15–30s 后 `exit status 124`，但设备实际已连上。

**根因 1（代码）**：`bluetoothctl connect` 在部分设备上连接成功后不退出，被 `timeout 30` 强杀返回 124；`bluetooth.Connect()` 把 124 一律当失败。
修复：新增 `isTimeoutExit(err)`（`errors.As` + `ExitCode() == 124`），超时后回查 `Info_(mac)`，已连接则视为成功。

**根因 2（环境）**：配对密钥损坏（`br-connection-key-missing`）→ 连接建立后立即被 bluez 断开，表现为反复重试直至挂起。
修复：`bluetoothctl remove` 后重新配对。要点是配对提示 `Confirm passkey NNNNNN (yes/no)` 必须应答 `yes`（非 tty 管道坑见技术要点）。

**其他改动**：
- `Connect()` 检测到 `br-connection-key-missing` 时改为输出可操作指引（照抄 bluez 原文对用户无意义）。
- 修复测试：`TestWaitBluezTimeout` 在耳机连着时必失败（BluezSink 立即命中）→ 改为检测到 bluez sink 则 `t.Skip`。单测恢复全绿。
- 文档：README.md / README_zh.md 新增「故障排查 / Troubleshooting」章节；本文档更新。
- 部署：`go build -trimpath -ldflags="-s -w" -o ~/.local/bin/blue-select .`

**真机验证证据（2026-09-16 08:20–08:25）**：

| 验证项 | 结果 |
|:---|:---|
| `blue-select connect FreeArc` | 2.93s，exit 0，「✓ HUAWEI FreeArc 已连接，输出已切换到 bluez_output.30_96_10_FD_B6_88.1（迁移 1 个播放流）」+ 电量 100% |
| `blue-select disconnect FreeArc` | 2.64s，exit 0，默认输出先切回 `alsa_output.pci-0000_00_1f.3.analog-stereo`（迁移 1 流）再断开 |
| `blue-select status` | 显示 `HUAWEI FreeArc [已连接] 电量 100% (默认)` + `profile: a2dp-sink` + sink 列表带 `*` 标记 |
| watch 守护 | journalctl 出现「已切换默认输出到 bluez_output.30_96_10_FD_B6_88.1（迁移 1 个播放流）」 |
| `go test ./...` / `go vet ./...` | 4 包全 ok / 无输出 |
| 配置文件 | `~/.config/blue-select/config.json`：default_device = HUAWEI FreeArc |

## 上次完成的工作（2026-08-26）
- 新增 `disconnect` 子命令：解析设备（复用 ResolveDevice，无参用默认设备）→ 未连接则幂等提示退出 0 → 若默认输出是该设备的 bluez sink 先 `switchAwayFrom` 迁到回退 sink → `bluetoothctl disconnect`
- bluetooth 包新增 `Disconnect(mac)`（对称于 Connect，带回显摘要）；audio 包新增 `BluezSinkPrefix(mac)`（MAC 冒号换下划线 + 后缀 `.`）与 `PickFallback(sinks, excludePrefix)`（排除本设备全部 profile sink，状态优先级 RUNNING > IDLE > SUSPENDED > 其他，同级稳定保序）
- 新增 4 个单测：TestPickFallback（RUNNING 优先 + 排除前缀）/ TestPickFallbackNoExcluded（IDLE > SUSPENDED）/ TestPickFallbackAllExcluded / TestPickFallbackEmpty + TestBluezSinkPrefix；go test 全绿（4 包 ok）
- README.md + README_zh.md 双语同步 disconnect 示例；已部署 `~/.local/bin/blue-select` 并冒烟：无参/指定名未连接→「未连接」exit 0，不存在的设备→报错 exit 1

## 上次完成的工作（2026-08-22 晚）
- Task 1-8 全部实施：config / bluetooth / audio 三包 + main.go 四命令，fixture 单测覆盖解析函数
- hephaestus 完成 Task 1-4（期间修复计划两处真机不符：①`pactl list sinks short` 实际 5 列非 6 列 ②ParseActiveProfile 匹配 bluez_card 需把 MAC 冒号换下划线）；Task 5 超时中断后由主会话内联完成 Task 5-7
- 部署 `~/.local/bin/blue-select`；`blue-select install` 写入 `~/.config/systemd/user/blue-select-watch.service` 并 enable --now，is-active=active
- README.md（英中双语定位，实际英文主体）+ 本文档更新

## 遗留问题 / 待办
- [ ] **USB 自动挂起永久修复待用户执行（需 root；agent 环境无 passwordless sudo）**
  - 现象：适配器 `power/control=auto` + `runtime=suspended`，内核日志 `hci0: Reading supported features failed (-16)`；可能造成扫描不到设备、连接偶发超时
  - 执行：`sudo bash ~/Downloads/fix-bluetooth-usb-autosuspend.sh`（立即生效 + 重启后保持）
  - 注意：已存在的 `/etc/modprobe.d/disable-usb-autosuspend.conf` **无效**——usbcore 是内核内置（非可加载模块），`/sys/module/usbcore/parameters/autosuspend` 仍为 2；脚本改用 udev 规则 `99-bluetooth-no-autosuspend.rules`
- [ ] 可选：新增 `pair` 子命令，自动化「密钥丢失 → 重新配对」流程（含 passkey 自动应答）；当前只能按 README 故障排查手工操作
- [ ] 可选：试验 PipeWire module-switch-on-connect 与 watch 的行为重叠（计划文档「前置实验」节）

## 技术要点（下一位 Agent 必读）
- **外部命令依赖**：bluetoothctl（包 timeout 10/30s）、pactl（包 timeout 10s）、pactl subscribe（watch 事件源）、systemctl --user。非 tty 下 bluetoothctl 会刷事件行，状态只信解析结果
- **超时语义**：`timeout` 命令强杀目标时退出码为 124，与目标自身失败（1）不同。`Connect()` 已区分：124 → 回查 `Info_` 判定真实连接状态；其他 → 原样报错
- **非 tty 配对坑（2026-09-16 实测）**：
  - bluez 配对会弹出 `[agent] Confirm passkey NNNNNN (yes/no):`，**必须应答 `yes`**。用管道喂命令时这条提示会吞掉下一条命令——本次实测 `trust <MAC>` 被当作答案吃进去 → `Failed to pair: org.bluez.Error.AuthenticationFailed`
  - 对策：脚本实时监听输出，命中 `yes/no` 立即写 `yes`（样例 `/tmp/bt_pair2.py`：`select` 轮询 stdout + 自动应答）
  - FreeArc 不是「无 PIN 自动完成」，必须走这个确认流程
- **bluez 会清除未配对设备**：非活跃扫描期间发现的设备会从 `bluetoothctl devices` 消失，此后 `pair` 报 `Device ... not available`。**扫描与配对必须在同一 bluetoothctl 会话内完成**（先 `scan on`，发现设备后立即 `pair`）
- **FreeArc 配对姿势**：双耳入盒 + 保持开盖 + 长按**盒内功能键** 2–5 秒（盒身白灯闪烁）——不是长按耳机触控区
- **USB 自动挂起**：`usbcore` 内核内置 → `/etc/modprobe.d/` 的 `options usbcore autosuspend=-1` 永不生效；正确做法是 udev 规则（在 `60-autosuspend.rules` 之后执行）或内核 cmdline `usbcore.autosuspend=-1`
- **关键实现细节**：
  - sink 名 = MAC 冒号换下划线 + `.1`（`bluez_output.30_96_10_FD_B6_88.1`）；匹配 bluez_card 同样要换下划线
  - `pactl list sinks short` 是 **5 列**（ID/Name/Driver/Spec/State），State 取第 5 列
  - ParseActiveProfile 按 "Card #" 切块、块内同时含 bluez_card.<mac> 与 Active Profile: 即命中，不依赖行序
  - `bluetooth.Info_` 下划线后缀是为避免与类型 Info 重名
  - `TestWaitBluezTimeout` 依赖「无 bluez sink」环境，耳机连着时必须跳过（已改为自动 `t.Skip`）
- **测试模式**：解析函数导出（ParseDevices/ParseInfo/ParseSinks/ParseActiveProfile），testdata/ 放真实命令输出 fixture
- **构建部署**：`go build -trimpath -ldflags="-s -w" -o ~/.local/bin/blue-select .`；测试 `go test ./...`
- **配置**：`~/.config/blue-select/config.json`（0644），首次 connect <名字> 成功即记录并设默认

## 知识图谱
graphify-out/ 已生成（2026-08-26，107 节点 / 193 边 / 9 社区），已提交 git。代码变更后重建：`graphify update .`

## 相关文档
- 实施计划：docs/plans/2026-08-22-blue-select-cli.md（含实测事实节与 Self-Review 记录）
- 命令来源 spec：~/prompt_boilerplates/System_Fix/bluetooth-pairing-troubleshoot.md（v1.3.0，已补入非 tty passkey 坑与设备清除机制）
