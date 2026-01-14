# Sticker Tool 设计

从终端用户视角，目标是提供一个快速、稳定的 TUI 工具：我只需选择视频或目录，就能得到符合 Telegram 视频贴纸要求（WebM/VP9、<=3s、<=30fps、512 边约束、<=256KB）的输出，无需了解 ffmpeg 细节。

## 范围

- 使用 Bubble Tea 构建 TUI，支持单文件与目录批量处理。
- 先实现第一种类型：视频贴纸（video sticker）。
- 支持视频 / 图片 / GIF 输入并转换为视频贴纸。
- 使用 u2takey/ffmpeg-go 驱动 ffmpeg/ffprobe。
- 严格校验 Telegram 视频贴纸规格。
- 自动降码率，若仍超 256KB 再降分辨率/帧率。

## 非目标

- v1 不支持 emoji（但需保留扩展空间）。
- 不做复杂循环内容生成，只保证可循环播放的基础约束。
- 不考虑向后兼容旧格式。

## 用户体验（TUI）

状态：
- 输入选择：文件/目录。
- 参数概览：展示媒体信息与输出策略。
- 处理进度：任务列表 + 总体进度。
- 结果汇总：成功/失败统计与简要错误。

批量行为：
- 目录模式递归扫描并过滤可处理格式。
- 单个失败不影响其它任务。

## 架构

采用 Clean Architecture，严格分层：
- UI/TUI：Bubble Tea model 与组件。
- App：用例编排、任务规划、管线调度、进度汇总。
- Domain：规格定义、策略与校验规则。
- Infra：ffmpeg/ffprobe 适配器、文件系统、临时文件、日志。

依赖方向：UI -> App -> Domain -> Infra（仅通过接口）。

## 模块拆分

- internal/ui
  - model：状态机与渲染。
  - components：list/progress/spinner/textinput。
- internal/app
  - JobPlanner：扫描输入、过滤格式、生成 Job。
  - Pipeline：并发执行、取消、进度事件。
  - ResultAggregator：汇总输出与错误。
- internal/domain
  - StickerSpec：视频贴纸约束。
  - ResizeStrategy：多轮降码率/降分辨率/降帧率策略。
  - Validator：输出校验。
- internal/infra
  - FFmpegRunner/FFprobeRunner：ffmpeg-go 封装。
  - FileStore：临时/输出路径管理。
  - Logger：结构化日志。

## 数据流

1. UI 获取输入并触发规划。
2. JobPlanner 生成 Job 列表（自动识别视频/图片/GIF）。
3. Pipeline 处理每个 Job：
   - 读取媒体信息（图片/GIF 走基础探测）。
   - 生成策略。
   - 编码输出。
   - 校验并产出结果。
4. UI 展示进度与汇总。

## 编码策略（视频贴纸）

约束：
- WebM + VP9。
- 无音频。
- FPS <= 30。
- 一边 = 512，另一边 <= 512。
- 时长 <= 3s。
- 体积 <= 256KB。

流程：
- 探测元信息。
- 缩放：最长边 = 512，等比缩放。
- 裁切：>3s 时截前 3s。
- 去音频：统一 -an。
- 首轮：按目标体积估算码率编码。
- 若 >256KB：
  1) 降码率。
  2) 仍超限则降分辨率。
  3) 仍超限则降帧率到最低阈值。
- 最终校验，不满足则报错。

图片与 GIF 输入：
- 静态图片：默认 3s / 30fps，TUI 可调整；使用循环图片生成视频，再走同一编码与校验流程。
- GIF：保留帧序并限制 <=30fps；若 >3s 则截前 3s，若 <3s 则循环补齐到 3s；之后进入统一降级与校验流程。

## 校验规则

- 容器：WebM。
- 编码：VP9。
- 音频：无。
- 时长：<= 3.0s。
- FPS：<= 30。
- 尺寸：一边 = 512，另一边 <= 512。
- 体积：<= 256KB。

## 错误处理与日志

- 错误分层：扫描/探测/编码/校验/写入。
- 批量不中断，单任务失败继续。
- UI 展示简要错误；详细诊断写入日志。
- 记录 ffmpeg/ffprobe exit code 与 stderr 尾部。

## 并发与性能

- 采用 worker pool 并发。
- 限制并发 ffmpeg 数量，避免资源打满。
- 临时文件任务结束后清理。

## 可扩展性

- 新增规格只需扩展 Domain 层。
- Infra 可替换编码器实现。
- UI 可添加模式选择，不改底层逻辑。

## 测试策略

- Domain 单元测试：规则与策略分支。
- App 单元测试：规划与调度（mock runner）。
- 集成测试：小样本视频/图片/GIF 走全链路。

## 风险与缓解

- 体积仍超限：清晰提示失败原因与建议。
- 输入格式异常：探测阶段快速失败。
- 编码耗时：提供进度与取消。

## 验收标准（v1）

- TUI 支持文件与目录批量处理。
- 输出满足 Telegram 视频贴纸规格。
- 自动降级策略生效并可诊断。
- 产生可用日志。
