# 首次压缩率自适应设计

从终端用户视角，我希望一次转换就能更接近 Telegram 视频贴纸 256KB 上限，减少多次重复压缩带来的耗时与画质损失，同时仍然严格满足贴纸规格。

## 背景

当前编码策略固定从较高码率起步，若输出超 256KB 就继续降码率/分辨率/帧率。对于高复杂度或长时长素材，这会导致多轮重复压缩，时间成本高。

## 目标

- 首次编码更接近 256KB 上限，减少重复压缩次数。
- 覆盖视频、图片、GIF（输出均为视频贴纸）。
- 规格不变：WebM/VP9、无音频、<=3s、<=30fps、边长 512、<=256KB。
- 失败时仍保持现有降级兜底策略。

## 非目标

- 不优化 UI/交互展示。
- 不改变输出格式或贴纸校验规则。
- 不保证向后兼容既有策略接口（允许破坏性变更）。

## 约束

- 目标体积固定为 256KB。
- ffprobe 可能缺失 `bit_rate` 或 `duration`。
- 源文件可能是图片或 GIF，码率估算可能不准确。

## 现状

- `BuildAttempts` 按固定顺序生成尝试：先多个码率步进，再降分辨率，再降帧率。
- `baseBitrate` 仅由目标体积与时长估算，不考虑源文件复杂度或大小。

## 方案概述

在首次尝试前引入“源复杂度”信号：

1. 采集磁盘文件大小（`os.Stat`）。
2. 采集 ffprobe 的 `format.bit_rate` 与 `duration`，估算 `sizeByBitrate`。
3. `sourceSize = max(actualSize, sizeByBitrate)`。
4. 以 `ratio = targetSize / sourceSize` 选择首轮码率倍率，并将该倍率移动到尝试队列首位。
5. 其余策略与顺序保持不变，继续作为兜底。

## 详细设计

### 数据来源

- `actualSize`: 输入文件的磁盘大小（字节）。
- `bitrateBps`: ffprobe `format.bit_rate`（bps），若缺失则为 0。
- `duration`: 仍用现有规则得到 `baseDuration`。
- `sizeByBitrate = bitrateBps * duration / 8`（字节）。
- `sourceSize = max(actualSize, sizeByBitrate)`。

### 首次倍率选择

设 `targetSize = 256KB`，`ratio = targetSize / sourceSize`：

| ratio 区间 | 首次倍率 |
| --- | --- |
| >= 0.9 | 1.0 |
| [0.7, 0.9) | 0.85 |
| [0.5, 0.7) | 0.70 |
| < 0.5 | 0.55 |

若 `sourceSize <= 0` 或 `duration <= 0`，则不启用自适应，保持原有顺序。

### baseBitrate

保持现有计算：`baseBitrateKbps = max(150, targetSize*8/duration/1000)`。

### 尝试顺序调整

- 将“选中的倍率”移动到 `bitrateSteps` 的首位（去重）。
- 尝试队列仍然包含所有倍率 + 降分辨率 + 降帧率的组合。
- 输出校验失败时继续使用原有兜底逻辑。

### 适用范围

所有输入类型（视频/图片/GIF）都使用新首次倍率选择逻辑。

## 数据流变化

1. Pipeline 在 Probe 前后获取 `actualSize`（`os.Stat`）。
2. ffprobe 增加 `format.bit_rate` 解析，注入到 `MediaInfo`。
3. `BuildAttempts` 新增输入参数（或新结构体）以接收 `actualSize` 与 `bitrateBps`。
4. 构建 attempts 时先调整首轮倍率，再生成所有尝试。

## 影响范围

- `internal/infra/ffprobe.go`: 解析 `format.bit_rate`。
- `internal/domain/media.go`: 增加 `BitrateBps` 字段（或新结构体）。
- `internal/app/pipeline.go`: 读取输入文件大小并传递给策略。
- `internal/domain/strategy.go`: 选择首轮倍率并重排 `bitrateSteps`。
- 测试：策略、ffprobe 解析、pipeline 行为。

## 测试计划

- Domain：覆盖 ratio 分段选择与首轮倍率重排；覆盖无 `bit_rate`/无 `duration` 的回退。
- Infra：ffprobe 解析 `bit_rate` 字段。
- App：输入文件大小传递路径可用（使用 mock/临时文件）。

## 风险与缓解

- 码率估算不准导致首次过度压缩：仍保留后续尝试，且 ratio 分段较保守。
- ffprobe 数据缺失：自动回退旧逻辑，保证稳定性。

