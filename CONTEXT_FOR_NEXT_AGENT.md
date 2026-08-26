# CONTEXT_FOR_NEXT_AGENT.md

## 项目当前状态
blue-select — Go CLI，蓝牙耳机一键连接 + 音频输出自动切换（connect / disconnect / status / watch / install 五个子命令）。**开发完成：代码全部提交、单测全绿、已部署 `~/.local/bin/blue-select`、systemd user 服务 blue-select-watch 已安装且 active。真机蓝牙端到端验证待耳机在场后补做（见遗留问题）。**

## 最后一次完成的工作（2026-08-26）
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
- [ ] **待真机验证（耳机不在场，2026-08-26）**：
  - `blue-select connect freearc` → 应输出「已连接，输出已切换」+ 电量
  - `blue-select disconnect freearc` → 已连时：默认输出先切回本地（若默认是耳机）→「✓ HUAWEI FreeArc 已断开」；断开后 `status` 应显示未连接
  - watch 守护验证：`bluetoothctl disconnect 30:96:10:FD:B6:88 && sleep 2 && bluetoothctl connect 30:96:10:FD:B6:88` → journalctl --user -u blue-select-watch 应出现「已切换默认输出」
  - 坑提醒：耳机连手机时不广播/拒绝电脑连接，先在手机端断开
- [ ] 可选：试验 PipeWire module-switch-on-connect 与 watch 的行为重叠（计划文档「前置实验」节）

## 技术要点（下一位 Agent 必读）
- **外部命令依赖**：bluetoothctl（包 timeout 10/30s）、pactl（包 timeout 10s）、pactl subscribe（watch 事件源）、systemctl --user。非 tty 下 bluetoothctl 会刷事件行，状态只信解析结果
- **关键实现细节**：
  - sink 名 = MAC 冒号换下划线 + `.1`（`bluez_output.30_96_10_FD_B6_88.1`）；匹配 bluez_card 同样要换下划线
  - `pactl list sinks short` 是 **5 列**（ID/Name/Driver/Spec/Channels 后直接 State？——以 audio.ParseSinks 为准，State 取最后一段），不是计划初稿写的 6 列定长
  - ParseActiveProfile 按 "Card #" 切块、块内同时含 bluez_card.<mac> 与 Active Profile: 即命中，不依赖行序
  - `bluetooth.Info_` 下划线后缀是为避免与类型 Info 重名
  - `TestWaitBluezTimeout` 在耳机恰好连着时会失败（BluezSink 找得到），届时跳过该用例即可
- **测试模式**：解析函数导出（ParseDevices/ParseInfo/ParseSinks/ParseActiveProfile），testdata/ 放真实命令输出 fixture
- **构建部署**：`go build -trimpath -ldflags="-s -w" -o ~/.local/bin/blue-select .`；测试 `go test ./...`
- **配置**：`~/.config/blue-select/config.json`（0644），首次 connect <名字> 成功即记录并设默认

## 知识图谱
graphify-out/ 已生成（2026-08-26，107 节点 / 193 边 / 9 社区），已提交 git。代码变更后重建：`graphify update .`

## 相关文档
- 实施计划：docs/plans/2026-08-22-blue-select-cli.md（含实测事实节与 Self-Review 记录）
- 命令来源 spec：~/prompt_boilerplates/System_Fix/bluetooth-pairing-troubleshoot.md v1.2.0
