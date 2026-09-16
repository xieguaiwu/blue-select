# Graph Report - blue-select  (2026-09-16)

## Corpus Check
- 13 files · ~8,927 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 111 nodes · 200 edges · 9 communities (8 shown, 1 thin omitted)
- Extraction: 76% EXTRACTED · 24% INFERRED · 0% AMBIGUOUS · INFERRED: 48 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `9836636e`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- runConnect
- bluetooth.go
- blue-select
- 文件结构
- audio.go
- Load
- CONTEXT_FOR_NEXT_AGENT.md
- ResolveDevice
- blue-select

## God Nodes (most connected - your core abstractions)
1. `runConnect()` - 12 edges
2. `ResolveDevice()` - 9 edges
3. `文件结构` - 9 edges
4. `Load()` - 8 edges
5. `runDisconnect()` - 8 edges
6. `switchAwayFrom()` - 8 edges
7. `runStatus()` - 8 edges
8. `blue-select` - 8 edges
9. `blue-select` - 8 edges
10. `Sinks()` - 7 edges

## Surprising Connections (you probably didn't know these)
- `runDisconnect()` --calls--> `Disconnect()`  [INFERRED]
  main.go → internal/bluetooth/bluetooth.go
- `runStatus()` --calls--> `Sinks()`  [INFERRED]
  main.go → internal/audio/audio.go
- `switchAwayFrom()` --calls--> `Sinks()`  [INFERRED]
  main.go → internal/audio/audio.go
- `runWatch()` --calls--> `BluezSink()`  [INFERRED]
  main.go → internal/audio/audio.go
- `switchAwayFrom()` --calls--> `BluezSinkPrefix()`  [INFERRED]
  main.go → internal/audio/audio.go

## Import Cycles
- None detected.

## Communities (9 total, 1 thin omitted)

### Community 0 - "runConnect"
Cohesion: 0.38
Nodes (11): Default(), MoveAllStreams(), SetDefault(), main(), runConnect(), runDisconnect(), runInstall(), runStatus() (+3 more)

### Community 1 - "bluetooth.go"
Cohesion: 0.21
Nodes (15): Device, Info, Connect(), Disconnect(), fieldYes(), Info_(), isTimeoutExit(), ListDevices() (+7 more)

### Community 2 - "blue-select"
Cohesion: 0.11
Nodes (16): blue-select, Build & install, Configuration, Known limitations, License, Problem it solves, systemd service, Troubleshooting (+8 more)

### Community 3 - "文件结构"
Cohesion: 0.13
Nodes (14): blue-select CLI Implementation Plan, Global Constraints, Self-Review 记录, Task 1: 项目脚手架 + config 包, Task 2: bluetooth 包（bluetoothctl 封装）, Task 3: audio 包（pactl 封装）, Task 4: connect 命令, Task 5: status 命令 (+6 more)

### Community 4 - "audio.go"
Cohesion: 0.20
Nodes (21): Sink, Duration, ActiveProfile(), BluezSink(), BluezSinkPrefix(), pactl(), ParseActiveProfile(), ParseSinks() (+13 more)

### Community 5 - "Load"
Cohesion: 0.42
Nodes (7): Config, configPath(), Load(), Save(), T, TestLoadMissingFile(), TestSaveLoadRoundtrip()

### Community 6 - "CONTEXT_FOR_NEXT_AGENT.md"
Cohesion: 0.20
Nodes (8): 上次完成的工作（2026-08-22 晚）, 上次完成的工作（2026-08-26）, 技术要点（下一位 Agent 必读）, 最后一次完成的工作（2026-09-16）, 相关文档, 知识图谱, 遗留问题 / 待办, 项目当前状态

### Community 7 - "ResolveDevice"
Cohesion: 0.60
Nodes (5): ResolveDevice(), T, TestResolveDeviceByArg(), TestResolveDeviceDefault(), TestResolveDeviceErrors()

## Knowledge Gaps
- **35 isolated node(s):** `blue-select`, `项目当前状态`, `最后一次完成的工作（2026-09-16）`, `上次完成的工作（2026-08-26）`, `上次完成的工作（2026-08-22 晚）` (+30 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `runConnect()` connect `runConnect` to `bluetooth.go`, `audio.go`, `Load`, `ResolveDevice`?**
  _High betweenness centrality (0.097) - this node is a cross-community bridge._
- **Why does `ResolveDevice()` connect `ResolveDevice` to `runConnect`, `bluetooth.go`, `Load`?**
  _High betweenness centrality (0.061) - this node is a cross-community bridge._
- **Why does `runDisconnect()` connect `runConnect` to `bluetooth.go`, `Load`, `ResolveDevice`?**
  _High betweenness centrality (0.054) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `runConnect()` (e.g. with `Default()` and `MoveAllStreams()`) actually correct?**
  _`runConnect()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **Are the 3 inferred relationships involving `ResolveDevice()` (e.g. with `TestResolveDeviceByArg()` and `TestResolveDeviceDefault()`) actually correct?**
  _`ResolveDevice()` has 3 INFERRED edges - model-reasoned connections that need verification._
- **Are the 5 inferred relationships involving `Load()` (e.g. with `TestLoadMissingFile()` and `TestSaveLoadRoundtrip()`) actually correct?**
  _`Load()` has 5 INFERRED edges - model-reasoned connections that need verification._
- **What connects `blue-select`, `项目当前状态`, `最后一次完成的工作（2026-09-16）` to the rest of the system?**
  _35 weakly-connected nodes found - possible documentation gaps or missing edges._