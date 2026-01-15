# 并发执行器设计

## 设计目标（用户视角）
用户在处理阶段能充分利用本机 CPU 核心并发执行，整体耗时显著缩短；同时进度与结果稳定可追踪，界面不会乱跳，且未来增加新任务类型时无需重写执行逻辑。

## 背景与动机
当前处理流程按顺序执行，吞吐受限。未来将扩展到多任务类型，若继续在 UI 层手写串行/并发逻辑，会导致耦合和重复。

## 目标
- 并发执行任务，最大并发数不超过本机核心数。
- 任务类型可扩展（视频贴纸只是其中一种）。
- 统一的开始/完成事件，供 UI 显示进度与排序。
- 任务结果可稳定归档，具备确定性 ID。

## 非目标
- 不做细粒度进度（如帧级别）。
- 不做失败重试策略（后续可扩展）。

## 方案概述
引入通用并发执行器 `Executor`，将“任务规划”和“任务执行”解耦：
- 规划阶段生成 `[]Task`，每个任务携带 `ID`、`Type`、`Payload`。
- 执行器根据 `Type` 从注册的 `Handler` 中分发执行。
- 执行过程中发送 `TaskStarted`、`TaskFinished` 事件。

## 核心数据结构（建议）
- `TaskType`（string）
- `Task`：
  - `ID int`
  - `Type TaskType`
  - `Payload any`（或 `interface{}`，后续可用泛型/结构体封装）
- `TaskHandler`：`Handle(ctx context.Context, task Task) (app.Result, error)`
- `Executor`：
  - `Concurrency int`（默认 `runtime.GOMAXPROCS(0)`）
  - `Handlers map[TaskType]TaskHandler`

## 执行流程
1. UI 请求“开始处理” → 生成任务列表 `[]Task`。
2. 执行器启动 `N` 个 worker（N=核心数，且不超过任务数）。
3. 每个任务：
   - 发送 `TaskStarted{ID, Path}` 事件
   - 执行 `Handler`
   - 发送 `TaskFinished{ID, Result}` 事件
4. 执行器汇总结果（`[]Result`），返回给调用者。

## 事件消息（给 UI）
- `TaskStartedMsg{ID, Path, Type}`
- `TaskFinishedMsg{ID, Result}`
- UI 依据 ID 更新状态并排序（Running → Pending → Done/Failed）。

## 任务类型扩展
- 为每种任务新增 `TaskType` 与 `TaskHandler`。
- 规划阶段决定 `TaskType` 与 `Payload`，执行器无需感知业务细节。

## 并发与取消
- 使用 `context.Context` 支持取消（用户中断或退出）。
- worker 监听 ctx.Done()，确保快速退出。

## 失败处理
- 单任务失败不影响其他任务。
- `Result` 内保留错误和校验问题；错误会在 UI 列表中展示。

## 与现有 Pipeline 的关系
- 新增 `VideoStickerHandler` 复用现有 `Pipeline` 逻辑（单 job 执行）。
- 原 `pipeline.Run` 保留作为内部工具（或逐步弃用）。

## 迁移步骤
1. 引入 `Executor` 与 `TaskHandler` 接口。
2. 实现 `VideoStickerHandler`，包装现有 `Pipeline`。
3. UI 使用任务列表替换当前逐个执行逻辑。
4. 接入事件消息驱动进度 UI。

## 风险与应对
- **过多并发导致 IO 压力**：并发上限 = 核心数（可配置）。
- **任务类型扩展带来的 payload 混乱**：建议为每个任务类型定义专用结构体。

## 验证方式
- 并发任务数等于核心数，处理顺序不保证，但 UI 排序稳定。
- 中途取消后所有 worker 能退出，UI 状态更新正确。
