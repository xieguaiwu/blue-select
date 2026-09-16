# Graph Report - blue-select  (2026-08-26)

## Corpus Check
- 13 files · ~8,366 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 107 nodes · 193 edges · 9 communities (8 shown, 1 thin omitted)
- Extraction: 76% EXTRACTED · 24% INFERRED · 0% AMBIGUOUS · INFERRED: 47 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `38c04b3d`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- audio.go
- bluetooth.go
- blue-select
- 文件结构
- audio_test.go
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
8. `Sinks()` - 7 edges
9. `PickFallback()` - 7 edges
10. `main()` - 7 edges

## Surprising Connections (you probably didn't know these)
- `runConnect()` --calls--> `Connect()`  [INFERRED]
  main.go → internal/bluetooth/bluetooth.go
- `switchAwayFrom()` --calls--> `PickFallback()`  [INFERRED]
  main.go → internal/audio/audio.go
- `runConnect()` --calls--> `ListDevices()`  [INFERRED]
  main.go → internal/bluetooth/bluetooth.go
- `runStatus()` --calls--> `ListDevices()`  [INFERRED]
  main.go → internal/bluetooth/bluetooth.go
- `runConnect()` --calls--> `Info_()`  [INFERRED]
  main.go → internal/bluetooth/bluetooth.go

## Import Cycles
- None detected.

## Communities (9 total, 1 thin omitted)

### Community 0 - "audio.go"
Cohesion: 0.25
Nodes (19): Sink, Duration, ActiveProfile(), BluezSink(), BluezSinkPrefix(), Default(), MoveAllStreams(), pactl() (+11 more)

### Community 1 - "bluetooth.go"
Cohesion: 0.23
Nodes (14): Device, Info, Connect(), Disconnect(), fieldYes(), Info_(), ListDevices(), ParseDevices() (+6 more)

### Community 2 - "blue-select"
Cohesion: 0.12
Nodes (14): blue-select, Build & install, Configuration, Known limitations, License, Problem it solves, systemd service, blue-select (+6 more)

### Community 3 - "文件结构"
Cohesion: 0.13
Nodes (14): blue-select CLI Implementation Plan, Global Constraints, Self-Review 记录, Task 1: 项目脚手架 + config 包, Task 2: bluetooth 包（bluetoothctl 封装）, Task 3: audio 包（pactl 封装）, Task 4: connect 命令, Task 5: status 命令 (+6 more)

### Community 4 - "audio_test.go"
Cohesion: 0.31
Nodes (12): ParseActiveProfile(), PickFallback(), T, TestBluezSinkPrefix(), TestParseActiveProfile(), TestParseActiveProfileRealLayout(), TestParseSinks(), TestPickFallback() (+4 more)

### Community 5 - "Load"
Cohesion: 0.42
Nodes (7): Config, configPath(), Load(), Save(), T, TestLoadMissingFile(), TestSaveLoadRoundtrip()

### Community 6 - "CONTEXT_FOR_NEXT_AGENT.md"
Cohesion: 0.22
Nodes (7): 上次完成的工作（2026-08-22 晚）, 技术要点（下一位 Agent 必读）, 最后一次完成的工作（2026-08-26）, 相关文档, 知识图谱, 遗留问题 / 待办, 项目当前状态

### Community 7 - "ResolveDevice"
Cohesion: 0.48
Nodes (6): deviceNames(), ResolveDevice(), T, TestResolveDeviceByArg(), TestResolveDeviceDefault(), TestResolveDeviceErrors()

## Knowledge Gaps
- **32 isolated node(s):** `blue-select`, `项目当前状态`, `最后一次完成的工作（2026-08-26）`, `上次完成的工作（2026-08-22 晚）`, `遗留问题 / 待办` (+27 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `runConnect()` connect `audio.go` to `bluetooth.go`, `Load`, `ResolveDevice`?**
  _High betweenness centrality (0.099) - this node is a cross-community bridge._
- **Why does `ResolveDevice()` connect `ResolveDevice` to `audio.go`, `bluetooth.go`, `Load`?**
  _High betweenness centrality (0.064) - this node is a cross-community bridge._
- **Why does `switchAwayFrom()` connect `audio.go` to `bluetooth.go`, `audio_test.go`?**
  _High betweenness centrality (0.058) - this node is a cross-community bridge._
- **Are the 9 inferred relationships involving `runConnect()` (e.g. with `Default()` and `MoveAllStreams()`) actually correct?**
  _`runConnect()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **Are the 3 inferred relationships involving `ResolveDevice()` (e.g. with `TestResolveDeviceByArg()` and `TestResolveDeviceDefault()`) actually correct?**
  _`ResolveDevice()` has 3 INFERRED edges - model-reasoned connections that need verification._
- **Are the 5 inferred relationships involving `Load()` (e.g. with `TestLoadMissingFile()` and `TestSaveLoadRoundtrip()`) actually correct?**
  _`Load()` has 5 INFERRED edges - model-reasoned connections that need verification._
- **What connects `blue-select`, `项目当前状态`, `最后一次完成的工作（2026-08-26）` to the rest of the system?**
  _32 weakly-connected nodes found - possible documentation gaps or missing edges._