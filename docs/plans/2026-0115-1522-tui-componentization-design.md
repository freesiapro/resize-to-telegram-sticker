# TUI 组件化拆分设计

## 设计目标（用户视角）
用户希望在终端里浏览与选择文件、确认配置、执行处理并查看总结时，界面结构清晰、响应一致、反馈明确；各个区域行为稳定，焦点切换直观，错误提示及时且不会干扰主要流程。

## 背景与问题
当前 TUI 主要逻辑集中在 `internal/ui/model.go`，包含状态机、输入处理、数据组织和渲染拼接。职责耦合导致：
- 浏览、确认、配置、处理、总结的代码混在同一文件，定位和修改成本高。
- 视图渲染与交互逻辑交织，复用困难（如 modal、header、status/help）。
- 未来扩展（例如增加新视图或调整布局）会产生“牵一发而动全身”的风险。

## 现状概览
- 状态切换：`stateBrowse/stateConfirm/stateConfig/stateProcessing/stateSummary`。
- 视图生成：`viewBrowse/viewConfirm/viewConfig/viewProcessing/viewSummary`。
- 交互处理：`updateBrowse/updateConfirm/updateConfig`。
- 布局与样式：`layout.go/styles.go`。

## 设计原则
- 以 Bubble Tea 的 `Model` 生命周期为边界，组件内部负责自身 Update/View，父级负责路由与依赖注入。
- 数据流单向：父级持有“业务状态”，组件只负责该视图下的交互与展示。
- 最小 diff：只拆分必要的视图/交互，不改变现有行为。

## 方案概述
将 TUI 拆为“屏幕组件 + 共享子组件”的组合，结构类似前端页面组件树：
- 屏幕组件（Screen）承接一个业务状态的 UI/交互。
- 共享组件（Component）提供可复用的渲染或小型交互单元。

## 组件拆分
### 屏幕组件
1. **BrowseScreen**
   - 负责：左右列表浏览、过滤输入、选择集合维护、状态栏与帮助栏。
   - 内部子组件：
     - `BrowsePane`（左右区域合成）
     - `ListHeader`（搜索、选择数量）
     - `StatusBar` + `HelpBar`

2. **ConfirmScreen**
   - 负责：展示扫描结果或错误，确认进入配置。
   - 复用：`Modal`。

3. **ConfigScreen**
   - 负责：输出目录输入与确认执行。
   - 复用：`Modal`。

4. **ProcessingScreen**
   - 负责：处理中提示。
   - 复用：`Modal`。

5. **SummaryScreen**
   - 负责：展示成功/失败统计。
   - 复用：`Modal`。

### 共享组件
- `ModalView`：统一居中框渲染（title + body）。
- `Pane`：左右 pane 带边框与 header 渲染。
- `StatusBar`/`HelpBar`：底部两行拼接与裁剪逻辑。

## 数据流与状态
- 根模型持有业务依赖（目录扫描、Expand、Pipeline）和全局数据（结果、输出目录、样式、窗口尺寸）。
- 屏幕组件维护自身交互状态（例如 Browse 的 focus、filterInput、left/right list）。
- 组件之间只通过显式数据传递，不共享可变状态。

## 交互与消息
- 根模型继续作为 Bubble Tea 的唯一 `Model`。
- Update 流程：
  1) 根模型收到 `tea.Msg`。
  2) 分发给当前 Screen 的 `Update`。
  3) Screen 返回 `Cmd` 与“意图事件”（例如进入确认、开始处理）。
  4) 根模型处理事件完成状态切换。

## 文件结构调整
为避免“同层平铺”导致的混乱，按父子关系分层：Screen 归档到 `screens/`，公用组件归档到 `components/`，基础能力统一放到 `core/`。建议结构如下：

```
internal/ui/
  model.go
  screens/
    browse.go
    confirm.go
    config.go
    processing.go
    summary.go
  components/
    modal.go
    pane.go
    status.go
  core/
    entries.go
    layout.go
    list.go
    messages.go
    styles.go
```

说明：
- `screens/`：每个文件对应一个 Screen 组件。
- `components/`：只放可复用的视图/渲染单元。
- `core/`：布局、样式、列表与消息类型等基础能力。
- `model.go`：保留在 `ui/` 根目录，作为状态路由与依赖注入入口。

备注：Go 以目录为包边界，拆分为 `core/screens/components` 将引入子包依赖，需要把跨包使用的类型/方法导出，避免循环依赖。

## 迁移步骤
1. 抽出 `ModalView/StatusBar/Pane` 渲染函数，保持输出一致。
2. 将 `viewConfirm/viewConfig/viewProcessing/viewSummary` 挪到各自 Screen。
3. 将 browse 相关逻辑拆到 `BrowseScreen`，根模型只做路由与状态切换。
4. 清理 `model.go` 中的重复辅助函数与未再使用的逻辑。

## 风险与应对
- **焦点切换/输入更新遗漏**：为 BrowseScreen 补齐输入更新与 list 更新路径。
- **布局尺寸不一致**：继续使用 `layout.go` 现有计算，抽离不改算法。

## 验证方式
- 手动操作：
  - 浏览/过滤/选择/取消选择/进入确认/配置/处理/总结流程。
  - 窗口 resize 观察左右 pane、header 与 status 的尺寸是否正确。
