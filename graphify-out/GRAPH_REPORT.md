# Graph Report - blue-select  (2026-09-21)

## Corpus Check
- 13 files · ~10,944 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 129 nodes · 254 edges · 10 communities (9 shown, 1 thin omitted)
- Extraction: 74% EXTRACTED · 26% INFERRED · 0% AMBIGUOUS · INFERRED: 66 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `1af77736`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- runConnect
- bluetooth.go
- blue-select
- 文件结构
- audio_test.go
- Load
- CONTEXT_FOR_NEXT_AGENT.md
- ResolveDevice
- blue-select
- audio.go

## God Nodes (most connected - your core abstractions)
1. `runConnect()` - 14 edges
2. `ensurePlaybackProfile()` - 10 edges
3. `ResolveDevice()` - 9 edges
4. `runStatus()` - 9 edges
5. `文件结构` - 9 edges
6. `ParseProfiles()` - 8 edges
7. `Load()` - 8 edges
8. `runDisconnect()` - 8 edges
9. `switchAwayFrom()` - 8 edges
10. `blue-select` - 8 edges

## Surprising Connections (you probably didn't know these)
- `runStatus()` --calls--> `Sinks()`  [INFERRED]
  main.go → internal/audio/audio.go
- `switchAwayFrom()` --calls--> `Sinks()`  [INFERRED]
  main.go → internal/audio/audio.go
- `runWatch()` --calls--> `BluezSink()`  [INFERRED]
  main.go → internal/audio/audio.go
- `switchAwayFrom()` --calls--> `BluezSinkPrefix()`  [INFERRED]
  main.go → internal/audio/audio.go
- `switchAwayFrom()` --calls--> `PickFallback()`  [INFERRED]
  main.go → internal/audio/audio.go

## Import Cycles
- None detected.

## Communities (10 total, 1 thin omitted)

### Community 0 - "runConnect"
Cohesion: 0.34
Nodes (13): Default(), MoveAllStreams(), pactl(), SetDefault(), ListDevices(), main(), runConnect(), runDisconnect() (+5 more)

### Community 1 - "bluetooth.go"
Cohesion: 0.24
Nodes (13): Device, Info, Connect(), fieldYes(), Info_(), isTimeoutExit(), ParseDevices(), ParseInfo() (+5 more)

### Community 2 - "blue-select"
Cohesion: 0.11
Nodes (16): blue-select, Build & install, Configuration, Known limitations, License, Problem it solves, systemd service, Troubleshooting (+8 more)

### Community 3 - "文件结构"
Cohesion: 0.13
Nodes (14): blue-select CLI Implementation Plan, Global Constraints, Self-Review 记录, Task 1: 项目脚手架 + config 包, Task 2: bluetooth 包（bluetoothctl 封装）, Task 3: audio 包（pactl 封装）, Task 4: connect 命令, Task 5: status 命令 (+6 more)

### Community 4 - "audio_test.go"
Cohesion: 0.20
Nodes (20): IsA2DP(), ParseActiveProfile(), ParseProfiles(), PickA2DPProfile(), PickFallback(), T, TestBluezSinkPrefix(), TestCardName() (+12 more)

### Community 5 - "Load"
Cohesion: 0.42
Nodes (7): Config, configPath(), Load(), Save(), T, TestLoadMissingFile(), TestSaveLoadRoundtrip()

### Community 6 - "CONTEXT_FOR_NEXT_AGENT.md"
Cohesion: 0.18
Nodes (9): 上次完成的工作（2026-08-22 晚）, 上次完成的工作（2026-08-26）, 上次完成的工作（2026-09-16）, 技术要点（下一位 Agent 必读）, 最后一次完成的工作（2026-09-21）, 相关文档, 知识图谱, 遗留问题 / 待办 (+1 more)

### Community 7 - "ResolveDevice"
Cohesion: 0.60
Nodes (5): ResolveDevice(), T, TestResolveDeviceByArg(), TestResolveDeviceDefault(), TestResolveDeviceErrors()

### Community 9 - "audio.go"
Cohesion: 0.20
Nodes (17): CardProfiles, Sink, Duration, ActiveProfile(), BluezSink(), BluezSinkPrefix(), cardBlock(), CardName() (+9 more)

## Knowledge Gaps
- **36 isolated node(s):** `blue-select`, `项目当前状态`, `最后一次完成的工作（2026-09-21）`, `上次完成的工作（2026-09-16）`, `上次完成的工作（2026-08-26）` (+31 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `runConnect()` connect `runConnect` to `bluetooth.go`, `audio_test.go`, `Load`, `ResolveDevice`, `audio.go`?**
  _High betweenness centrality (0.107) - this node is a cross-community bridge._
- **Why does `ResolveDevice()` connect `ResolveDevice` to `runConnect`, `bluetooth.go`, `Load`?**
  _High betweenness centrality (0.057) - this node is a cross-community bridge._
- **Why does `Load()` connect `Load` to `runConnect`?**
  _High betweenness centrality (0.048) - this node is a cross-community bridge._
- **Are the 10 inferred relationships involving `runConnect()` (e.g. with `Default()` and `IsA2DP()`) actually correct?**
  _`runConnect()` has 10 INFERRED edges - model-reasoned connections that need verification._
- **Are the 7 inferred relationships involving `ensurePlaybackProfile()` (e.g. with `IsA2DP()` and `PickA2DPProfile()`) actually correct?**
  _`ensurePlaybackProfile()` has 7 INFERRED edges - model-reasoned connections that need verification._
- **Are the 3 inferred relationships involving `ResolveDevice()` (e.g. with `TestResolveDeviceByArg()` and `TestResolveDeviceDefault()`) actually correct?**
  _`ResolveDevice()` has 3 INFERRED edges - model-reasoned connections that need verification._
- **Are the 7 inferred relationships involving `runStatus()` (e.g. with `ActiveProfile()` and `Default()`) actually correct?**
  _`runStatus()` has 7 INFERRED edges - model-reasoned connections that need verification._