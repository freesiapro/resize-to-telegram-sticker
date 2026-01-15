# Processing 进度列表设计

## 设计目标（用户视角）
用户在处理阶段能看到全屏列表：当前正在处理的文件排在最上方，已完成的自动沉到底部，未处理的排在中间；同时能看到整体进度（已处理/总数）和当前处理项。

## 需求拆解
- 处理界面替换为“全屏列表”。
- 排序规则：Processing 在最上，Pending 在中间，Done/Failed 在最下（同组内保持稳定顺序）。
- 展示进度：已处理数量/总数，并显示当前处理项与进度条。
- 兼容未来并发：允许多个 Processing 项并列在顶部。

## 现状
- `ProcessingScreen` 仅显示固定文本。
- Pipeline 为顺序执行，`runPipelineCmd` 只在完成时回传结果。

## 方案概述
1. **数据模型**：新增处理项状态列表，用于渲染与排序。
2. **进度更新**：处理流程中发送“开始/完成”事件，驱动 UI 更新。
3. **进度条**：使用 `bubbles/progress` 渲染整体进度，展示 `done/total`。
4. **全屏列表**：ProcessingScreen 持有列表模型（或自绘列表），按规则排序并渲染。

## 数据结构建议
- `ProcessingStatus`: `Pending | Processing | Done | Failed`
- `ProcessingItem`:
  - `ID`（稳定排序用）
  - `Path`
  - `Status`
  - `Err`（失败时展示）

## 事件与消息流
- 新增处理事件消息：
  - `ProcessingStarted{JobID, Path}`
  - `ProcessingFinished{JobID, Result}`
- 处理命令改为“逐任务执行”，在每个任务开始/结束时发消息。
- `ProcessingScreen.Update` 接收这些消息并更新列表与进度。

## 进度条与渲染
- `progress.Model` 由 `ProcessingScreen` 持有。
- 每次 `doneCount` 变化调用 `progress.SetPercent(done/total)`。
- 处理 `progress.FrameMsg` 以支持动画（如需）。

## 排序规则
- Primary：`Processing` → `Pending` → `Done/Failed`
- Secondary：`ID` 保持输入顺序稳定。
- 当并发存在多个 Processing 项时，均置顶。

## 代码改动范围
- `internal/ui/screens/processing.go`：新增列表与进度条渲染。
- `internal/ui/model.go`：处理事件分发与状态切换。
- `internal/app` 或 `internal/ui`：增加进度事件消息与触发逻辑（具体实现见下一节）。

## 实现策略（两种可选）
1. **UI 驱动（最小侵入）**
   - `runPipelineCmd` 改为逐任务调用 `pipeline.Run`（单 job），在 UI 层发送开始/完成消息。
   - Pipeline 逻辑保持不变。

2. **Pipeline 驱动（更清晰）**
   - 为 `Pipeline` 增加 `RunWithProgress`，在内部发送开始/完成回调。
   - UI 只消费事件。

> 建议采用方案 1（最小 diff），后续并发时再下沉到 Pipeline 层。

## 风险与应对
- **UI 频繁更新**：使用稳定排序与最小重绘，避免列表跳动。
- **进度条动画干扰**：可先固定进度条不启用动画，后续再开。

## 验证方式
- 手动：处理流程中观察列表排序、进度计数是否正确。
- 模拟失败：确保失败项沉底并带错误信息。
