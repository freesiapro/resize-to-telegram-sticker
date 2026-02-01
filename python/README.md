# PySide6 GUI 纯 Python 重写方案

本文档说明如何将当前 Go/TUI 应用重写为 PySide6 桌面 GUI，并用纯 Python
实现全部处理逻辑（不依赖 Go 二进制）。目标是在保留现有行为和输出约束的前提下，
提供可用性更强的图形界面与清晰的打包方式。

## 目标与范围

- 用 PySide6（Qt Widgets）替换当前 TUI 向导。
- 处理流程、命名规则、约束规则保持与现有 Go 版一致。
- 纯 Python 处理核心，GUI 与核心解耦，便于测试与未来扩展。
- 支持批量处理、进度展示、失败原因与日志定位。

## 技术栈调研（要点）

- Qt for Python（PySide6）是 Qt 官方 Python 绑定，采用 LGPLv3/GPLv3
  或商业授权双许可。商业项目需使用正确分发渠道，避免许可风险。
- Qt Widgets 适合传统桌面工具，QMainWindow 自带菜单、工具栏、状态栏与
  中央控件区域，便于组织主界面布局。
- QThreadPool + QRunnable 适合后台任务并发执行，避免阻塞 UI 线程。
- QProcess 是 Qt 原生的外部进程管理方式，可捕获 stdout/stderr 与退出状态，
  适合对 ffmpeg/ffprobe 进行细粒度控制。
- pyside6-deploy 是官方部署工具（封装 Nuitka），通过 pysidedeploy.spec
  控制打包参数，适合构建可执行文件。

## 需要保留的现有行为（来自 Go 版）

目标类型与输入：
- 视频贴纸：视频/GIF -> WebM（VP9，无音频）。
- 静态贴纸：图片 -> PNG（至少一边 512px）。
- Emoji：图片 -> PNG（100x100，需补边成正方形）。

约束：
- 最大边长：512 px
- 最大帧率：30 FPS
- 最大时长：3 秒
- 最大文件体积：256 KB
- 输出格式必须满足 WebM + VP9 + 无音频（视频贴纸）

命名规则：
- 视频：<basename>_sticker.webm
- 静态贴纸：<basename>_sticker.png
- Emoji：<basename>_emoji.png

## 推荐架构（纯 Python）

保持“核心逻辑与 GUI 解耦”：

python/
  app/
    main.py              # GUI 入口
  core/
    constraints.py       # 常量与尺寸缩放逻辑
    media.py             # 输入类型识别、媒体信息模型
    strategy.py          # 编码尝试（bitrate/fps/scale）
    validate.py          # 输出校验规则
    selection.py         # 输入扫描与任务规划
    job.py               # Job/Attempt/Result/Issue 数据结构
  infra/
    ffprobe.py           # ffprobe 调用与 JSON 解析
    ffmpeg.py            # ffmpeg 调用、参数拼装、日志落盘
    files.py             # 目录递归扫描
  ui/
    main_window.py       # QMainWindow + 布局
    widgets.py           # 可复用控件
    workers.py           # QRunnable + 信号定义

关键原则：
- core 不依赖 Qt，便于单元测试与未来 CLI/服务化复用。
- UI 仅负责输入、调度与展示，不包含业务算法。
- 通过 dataclasses 定义 Job、Attempt、Result、Issue。

## 处理流程（Python 版）

### 输入扫描与任务规划

- 支持“文件”与“目录递归扫描”两种模式。
- 根据扩展名识别输入类型：
  - 视频：.mp4 .mov .webm .mkv .avi
  - 图片：.png .jpg .jpeg .webp
  - GIF：.gif
- 跳过不支持的文件，记录原因并展示在 GUI 中。

### 视频/GIF 处理管线

1. ffprobe 获取媒体信息：尺寸、FPS、时长、编码格式、音频流。
2. 生成编码尝试列表（与 Go 版算法一致）：
   - 基础尺寸缩放到 512 以内
   - Bitrate 分段降级
   - FPS 降级与缩放回退
3. 逐个尝试：
   - ffmpeg 缩放、裁剪时长（<= 3s）
   - 输出 VP9 + WebM（无音频）
   - 再次 ffprobe + 文件大小校验
4. 首个通过校验的尝试即为最终结果。

### 图片/Emoji 处理管线

1. ffmpeg 缩放到目标边长：
   - 静态贴纸：512
   - Emoji：100
2. Emoji 需要补边成正方形（透明背景）。
3. 输出 PNG，使用 Pillow 读取并校验尺寸与格式。

### 错误与日志策略

- 保留与 Go 版一致的行为：ffmpeg 失败时输出
  <output>.ffmpeg-error.log，便于定位问题。
- GUI 可提供“展开日志”与“复制错误信息”按钮。

## GUI 设计要点（Qt Widgets）

主窗口布局（QMainWindow）：
- 目标选择（Combo 或 Radio）
- 输入模式切换（文件 / 目录）
- 输入路径与输出目录选择
- 运行 / 取消按钮
- 任务列表（表格或列表视图）
- 结果摘要（成功/失败/跳过数）
- 日志面板（可折叠）

后台任务：
- 使用 QThreadPool + QRunnable 执行每个 Job。
- 任务信号：started / progress / finished / error。
- 取消：共享“取消标志”，每次编码尝试前检查。

## 依赖与运行环境

- Python 版本：建议 >= 3.8（与 Qt 官方要求一致）。
- 依赖：
  - PySide6
  - ffmpeg-python
  - Pillow（图片校验）
- ffmpeg/ffprobe 需要可执行文件在 PATH 中，或随应用一起打包。

## 打包与发布

- 使用 pyside6-deploy 生成可执行文件。
- 首次运行会生成 pysidedeploy.spec，后续由该文件控制打包参数。
- 将 ffmpeg/ffprobe 放入发布包中并配置 PATH，或在启动时检测并提示。

## 迁移步骤（建议）

1. 迁移 domain 层：常量、缩放、校验、策略算法。
2. 迁移 infra 层：ffprobe 解析、ffmpeg 调用与日志。
3. 迁移 pipeline：视频/图片处理流程。
4. 搭建 GUI 与任务执行器，连通进度与取消逻辑。
5. 增加测试与示例资源（小图/短视频）。
6. 引入打包脚本与发布说明。

## 测试建议

- 单元测试：尺寸缩放、attempt 生成、校验逻辑。
- 集成测试：对样例媒体进行端到端输出验证。
- 兼容性测试：Windows/ macOS/ Linux 路径与编码差异。

## 待确认事项

- 是否支持拖拽导入？
- 是否需要输出预览（图像/视频）？
- 日志默认保存位置？
- 失败项是否提供“单条重试”？

## 参考资料（原始链接）

```text
https://doc.qt.io/qtforpython-6/commercial/index.html
https://doc.qt.io/qtforpython-6/PySide6/QtWidgets/QMainWindow.html
https://doc.qt.io/qtforpython-6/PySide6/QtCore/QThreadPool.html
https://doc.qt.io/qtforpython-6/PySide6/QtCore/QRunnable.html
https://doc.qt.io/qtforpython-6/PySide6/QtCore/QProcess.html
https://doc.qt.io/qtforpython-6/gettingstarted.html
https://doc.qt.io/qtforpython-6.8/deployment/deployment-pyside6-deploy.html
https://doc.qt.io/qt-6/licensing.html
```
