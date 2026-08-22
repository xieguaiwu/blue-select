# CONTEXT_FOR_NEXT_AGENT.md

## 项目当前状态
blue-select — Go CLI，蓝牙耳机一键连接 + 音频输出自动切换（connect / status / watch / install）。**当前状态：仅完成实施计划，代码未开始。**

## 下一步（如何开工）
1. 读实施计划：`docs/plans/2026-08-22-blue-select-cli.md`——自足文档，含实测事实、Global Constraints、Task 1-8 逐步指令与完整代码
2. 命令来源 spec：`~/prompt_boilerplates/System_Fix/bluetooth-pairing-troubleshoot.md`（v1.2.0）
3. 执行方式二选一：subagent 驱动（每任务派 hephaestus/quick，任务间审查）或内联逐任务执行；每 subagent 调用前过 `pi-resmon --recommend`
4. 编码全程对照 `~/prompt_boilerplates/Coding/development-quality-gates.md`（13 关卡）；完成后按 `project-documentation-protocol.md` 阶段 B 更新本文档

## 背景（为什么做这个工具）
2026-08-22 蓝牙排查实战发现两个高频痛点：
1. 连接成功但默认音频输出没切到耳机（最高频坑），需手动 `pactl set-default-sink` + 逐个 `move-sink-input`
2. 睡眠唤醒后偶发失联，要手动 `bluetoothctl connect`
本工具把这两步固化为一键命令 + systemd 常驻守护。

## 关键实测事实速查（详见计划的「实测事实」节）
- 主设备：HUAWEI FreeArc，MAC `30:96:10:FD:B6:88`（已 Paired+Trusted）
- sink 名格式：`bluez_output.30_96_10_FD_B6_88.1`（MAC 冒号换下划线 + `.1`）
- 设备连着手机时不广播且可能拒绝电脑连接——测试前先断手机端

## 技术约束
- Go stdlib only，零第三方依赖
- 所有外部命令（bluetoothctl/pactl/systemctl）包 `timeout` 防非 tty 挂起
- 纯子命令设计，无交互式 TUI（豁免 interactive-cli-design PTY 验收）
