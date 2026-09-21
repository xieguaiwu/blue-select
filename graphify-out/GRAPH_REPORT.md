# Graph Report - blue-select  (2026-09-21)

## Corpus Check
- 13 files · ~12,443 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 148 nodes · 302 edges · 9 communities (8 shown, 1 thin omitted)
- Extraction: 74% EXTRACTED · 26% INFERRED · 0% AMBIGUOUS · INFERRED: 78 edges (avg confidence: 0.8)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `1ab41615`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- main.go
- bluetooth.go
- blue-select
- 文件结构
- audio_test.go
- Load
- CONTEXT_FOR_NEXT_AGENT.md
- blue-select
- audio.go

## God Nodes (most connected - your core abstractions)
1. `runConnect()` - 15 edges
2. `runWatch()` - 11 edges
3. `Pair()` - 10 edges
4. `ResolveDevice()` - 10 edges
5. `ensurePlaybackProfile()` - 10 edges
6. `Load()` - 9 edges
7. `main()` - 9 edges
8. `runPair()` - 9 edges
9. `runStatus()` - 9 edges
10. `文件结构` - 9 edges

## Surprising Connections (you probably didn't know these)
- `runStatus()` --calls--> `Sinks()`  [INFERRED]
  main.go → internal/audio/audio.go
- `switchAwayFrom()` --calls--> `BluezSinkPrefix()`  [INFERRED]
  main.go → internal/audio/audio.go
- `switchAwayFrom()` --calls--> `PickFallback()`  [INFERRED]
  main.go → internal/audio/audio.go
- `runStatus()` --calls--> `Default()`  [INFERRED]
  main.go → internal/audio/audio.go
- `runStatus()` --calls--> `ActiveProfile()`  [INFERRED]
  main.go → internal/audio/audio.go

## Import Cycles
- None detected.

## Communities (9 total, 1 thin omitted)

### Community 0 - "main.go"
Cohesion: 0.21
Nodes (17): Device, ListDevices(), ParseDevices(), Trust(), deviceNames(), Duration, main(), ResolveDevice() (+9 more)

### Community 1 - "bluetooth.go"
Cohesion: 0.14
Nodes (21): bluezSession, Info, PairResult, Builder, Cmd, Connect(), Disconnect(), fieldYes() (+13 more)

### Community 2 - "blue-select"
Cohesion: 0.11
Nodes (16): blue-select, Build & install, Configuration, Known limitations, License, Problem it solves, systemd service, Troubleshooting (+8 more)

### Community 3 - "文件结构"
Cohesion: 0.13
Nodes (14): blue-select CLI Implementation Plan, Global Constraints, Self-Review 记录, Task 1: 项目脚手架 + config 包, Task 2: bluetooth 包（bluetoothctl 封装）, Task 3: audio 包（pactl 封装）, Task 4: connect 命令, Task 5: status 命令 (+6 more)

### Community 4 - "audio_test.go"
Cohesion: 0.20
Nodes (20): BluezSinkPrefix(), ParseActiveProfile(), ParseProfiles(), PickFallback(), T, TestBluezSinkPrefix(), TestCardName(), TestIsA2DP() (+12 more)

### Community 5 - "Load"
Cohesion: 0.42
Nodes (7): Config, configPath(), Load(), Save(), T, TestLoadMissingFile(), TestSaveLoadRoundtrip()

### Community 6 - "CONTEXT_FOR_NEXT_AGENT.md"
Cohesion: 0.17
Nodes (10): 上次完成的工作（2026-08-22 晚）, 上次完成的工作（2026-08-26）, 上次完成的工作（2026-09-16）, 同日续作：pair 子命令 + watch 档位校验（2026-09-21 10:20–11:00）, 技术要点（下一位 Agent 必读）, 最后一次完成的工作（2026-09-21）, 相关文档, 知识图谱 (+2 more)

### Community 9 - "audio.go"
Cohesion: 0.19
Nodes (25): CardProfiles, Sink, ActiveProfile(), BluezSink(), cardBlock(), CardName(), Default(), Duration (+17 more)

## Knowledge Gaps
- **36 isolated node(s):** `blue-select`, `项目当前状态`, `同日续作：pair 子命令 + watch 档位校验（2026-09-21 10:20–11:00）`, `上次完成的工作（2026-09-16）`, `上次完成的工作（2026-08-26）` (+31 more)
  These have ≤1 connection - possible missing edges or undocumented components.
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `runConnect()` connect `audio.go` to `main.go`, `bluetooth.go`, `Load`?**
  _High betweenness centrality (0.100) - this node is a cross-community bridge._
- **Why does `ResolveDevice()` connect `main.go` to `audio.go`, `Load`?**
  _High betweenness centrality (0.053) - this node is a cross-community bridge._
- **Why does `ensurePlaybackProfile()` connect `audio.go` to `main.go`, `bluetooth.go`?**
  _High betweenness centrality (0.051) - this node is a cross-community bridge._
- **Are the 10 inferred relationships involving `runConnect()` (e.g. with `Default()` and `IsA2DP()`) actually correct?**
  _`runConnect()` has 10 INFERRED edges - model-reasoned connections that need verification._
- **Are the 9 inferred relationships involving `runWatch()` (e.g. with `BluezSink()` and `Default()`) actually correct?**
  _`runWatch()` has 9 INFERRED edges - model-reasoned connections that need verification._
- **Are the 3 inferred relationships involving `ResolveDevice()` (e.g. with `TestResolveDeviceByArg()` and `TestResolveDeviceDefault()`) actually correct?**
  _`ResolveDevice()` has 3 INFERRED edges - model-reasoned connections that need verification._
- **What connects `blue-select`, `项目当前状态`, `同日续作：pair 子命令 + watch 档位校验（2026-09-21 10:20–11:00）` to the rest of the system?**
  _36 weakly-connected nodes found - possible documentation gaps or missing edges._