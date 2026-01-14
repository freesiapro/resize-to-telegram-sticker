# TUI Two-Pane Browser Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 实现全屏双栏文件浏览与选择、直接输入过滤、确认展开与配置步骤，并使用英文 UI 文案与 Lip Gloss 样式。

**Architecture:** UI 负责交互与状态机，App 负责选择展开与任务规划，Infra 负责文件系统读取。UI 通过接口依赖 App/Infra；输出路径规则在 App 层集中实现。

**Tech Stack:** Go, Bubble Tea, Bubbles (list/textinput), Lip Gloss.

**Skills:** @superpowers:test-driven-development @superpowers:verification-before-completion

---

### Task 1: 新增当前目录非递归列表

**Files:**
- Create: `internal/infra/dir.go`
- Create: `internal/infra/dir_test.go`

**Step 1: Write the failing test**

```go
package infra

import (
	"os"
	"path/filepath"
	"testing"
)

func TestListDirEntries(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.gif"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	entries, err := ListDirEntries(root)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	var hasDir, hasFile bool
	for _, e := range entries {
		if e.IsDir && e.Name == "sub" {
			hasDir = true
		}
		if !e.IsDir && e.Name == "a.gif" {
			hasFile = true
		}
	}
	if !hasDir || !hasFile {
		t.Fatalf("missing dir/file: %v", entries)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.gocache go test ./internal/infra -v`
Expected: FAIL (ListDirEntries undefined)

**Step 3: Write minimal implementation**

```go
package infra

import (
	"os"
	"path/filepath"
)

type DirEntry struct {
	Name string
	Path string
	IsDir bool
}

func ListDirEntries(root string) ([]DirEntry, error) {
	dirEntries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	entries := make([]DirEntry, 0, len(dirEntries))
	for _, e := range dirEntries {
		entries = append(entries, DirEntry{
			Name: e.Name(),
			Path: filepath.Join(root, e.Name()),
			IsDir: e.IsDir(),
		})
	}
	return entries, nil
}
```

**Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.gocache go test ./internal/infra -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/infra/dir.go internal/infra/dir_test.go
git commit -m "feat(infra): add non-recursive dir listing"
```

---

### Task 2: 选择展开与输出目录规则

**Files:**
- Create: `internal/app/selection.go`
- Create: `internal/app/selection_test.go`
- Modify: `internal/app/job.go`

**Step 1: Write the failing test**

```go
package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExpandSelections(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "cats"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "cats", "a.gif"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	selections := []SelectionItem{
		{Path: filepath.Join(root, "cats"), IsDir: true},
		{Path: filepath.Join(root, "b.png"), IsDir: false},
	}
	if err := os.WriteFile(filepath.Join(root, "b.png"), []byte("x"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	expander := SelectionExpander{}
	result, err := expander.Expand(selections, filepath.Join(root, "output"))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}

	if result.DirCount != 1 || result.FileCount != 1 {
		t.Fatalf("unexpected counts: %+v", result)
	}
	if result.TotalFiles != 2 {
		t.Fatalf("unexpected total files: %+v", result)
	}

	var hasDirOutput, hasOutputDir bool
	for _, job := range result.Jobs {
		if job.OutputDir == filepath.Join(root, "cats") {
			hasDirOutput = true
		}
		if job.OutputDir == filepath.Join(root, "output") {
			hasOutputDir = true
		}
	}
	if !hasDirOutput || !hasOutputDir {
		t.Fatalf("unexpected output dirs: %+v", result.Jobs)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.gocache go test ./internal/app -v`
Expected: FAIL (SelectionExpander undefined)

**Step 3: Write minimal implementation**

```go
package app

import (
	"path/filepath"
	"sort"

	"github.com/resize-to-telegram-sticker/internal/domain"
	"github.com/resize-to-telegram-sticker/internal/infra"
)

type SelectionItem struct {
	Path string
	IsDir bool
}

type ExpandResult struct {
	Jobs []Job
	DirCount int
	FileCount int
	TotalFiles int
	OutputDirs []string
	Skipped []Skipped
}

type SelectionExpander struct {
	ListFiles func(root string) ([]string, error)
}

func (e SelectionExpander) Expand(selections []SelectionItem, outputDir string) (ExpandResult, error) {
	result := ExpandResult{}
	if outputDir == "" {
		outputDir = "./output"
	}
	listFiles := e.ListFiles
	if listFiles == nil {
		listFiles = infra.ListFiles
	}

	jobs := make([]Job, 0)
	seen := make(map[string]struct{})
	outputSet := make(map[string]struct{})

	files := make([]SelectionItem, 0)
	dirs := make([]SelectionItem, 0)
	for _, s := range selections {
		if s.IsDir {
			dirs = append(dirs, s)
			continue
		}
		files = append(files, s)
	}

	for _, s := range files {
		kind, err := domain.DetectInputKind(s.Path)
		if err != nil {
			result.Skipped = append(result.Skipped, Skipped{Path: s.Path, Reason: err.Error()})
			continue
		}
		if _, ok := seen[s.Path]; ok {
			continue
		}
		seen[s.Path] = struct{}{}
		jobs = append(jobs, Job{InputPath: s.Path, Kind: kind, OutputDir: outputDir})
		result.FileCount++
		result.TotalFiles++
		outputSet[outputDir] = struct{}{}
	}

	for _, s := range dirs {
		filesInDir, err := listFiles(s.Path)
		if err != nil {
			return ExpandResult{}, err
		}
		result.DirCount++
		for _, path := range filesInDir {
			kind, err := domain.DetectInputKind(path)
			if err != nil {
				result.Skipped = append(result.Skipped, Skipped{Path: path, Reason: err.Error()})
				continue
			}
			if _, ok := seen[path]; ok {
				continue
			}
			seen[path] = struct{}{}
			jobs = append(jobs, Job{InputPath: path, Kind: kind, OutputDir: filepath.Clean(s.Path)})
			result.TotalFiles++
			outputSet[filepath.Clean(s.Path)] = struct{}{}
		}
	}

	result.Jobs = jobs
	result.OutputDirs = sortedKeys(outputSet)
	return result, nil
}

func sortedKeys(set map[string]struct{}) []string {
	keys := make([]string, 0, len(set))
	for k := range set {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
```

```go
package app

import "github.com/resize-to-telegram-sticker/internal/domain"

type Job struct {
	InputPath string
	Kind domain.InputKind
	OutputDir string
}
```

**Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.gocache go test ./internal/app -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/app/selection.go internal/app/selection_test.go internal/app/job.go
git commit -m "feat(app): add selection expansion and output rules"
```

---

### Task 3: Pipeline 输出目录与路径计算

**Files:**
- Modify: `internal/app/pipeline.go`
- Modify: `internal/app/pipeline_test.go`

**Step 1: Write the failing test**

```go
func TestPipelineUsesOutputDir(t *testing.T) {
	var gotOutput string
	type captureEncode struct{}
	func (captureEncode) Encode(_ context.Context, _ string, _ domain.EncodeAttempt, outputPath string, _ domain.EncodeOptions) error {
		gotOutput = outputPath
		return nil
	}

	p := Pipeline{Probe: fakeProbe{}, Encode: captureEncode{}}
	jobs := []Job{{InputPath: "/tmp/a.gif", Kind: domain.InputKindGIF, OutputDir: "/tmp/out"}}
	_ = p.Run(context.Background(), jobs)
	if gotOutput == "" || filepath.Dir(gotOutput) != "/tmp/out" {
		t.Fatalf("unexpected output: %s", gotOutput)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.gocache go test ./internal/app -v`
Expected: FAIL (output path uses input dir)

**Step 3: Write minimal implementation**

```go
func outputPath(job Job) string {
	base := strings.TrimSuffix(filepath.Base(job.InputPath), filepath.Ext(job.InputPath))
	name := base + "_sticker.webm"
	if job.OutputDir == "" {
		return filepath.Join(filepath.Dir(job.InputPath), name)
	}
	return filepath.Join(job.OutputDir, name)
}

// In Run(): ensure output dir exists before encoding
if job.OutputDir != "" {
	if err := os.MkdirAll(job.OutputDir, 0o755); err != nil {
		results = append(results, Result{InputPath: job.InputPath, Err: err})
		continue
	}
}
```

**Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.gocache go test ./internal/app -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/app/pipeline.go internal/app/pipeline_test.go
git commit -m "feat(app): use output dir in pipeline"
```

---

### Task 4: 目录条目与过滤逻辑（UI 纯函数）

**Files:**
- Create: `internal/ui/entries.go`
- Create: `internal/ui/entries_test.go`

**Step 1: Write the failing test**

```go
package ui

import "testing"

func TestFilterEntries(t *testing.T) {
	entries := []entryItem{
		{name: "cats", isDir: true},
		{name: "cat01.gif", isDir: false},
		{name: "dogs", isDir: true},
	}
	filtered := filterEntries(entries, "cat")
	if len(filtered) != 2 {
		t.Fatalf("expected 2, got %d", len(filtered))
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.gocache go test ./internal/ui -v`
Expected: FAIL (filterEntries undefined)

**Step 3: Write minimal implementation**

```go
package ui

import "strings"

type entryItem struct {
	path string
	name string
	isDir bool
	isParent bool
	selected bool
}

func filterEntries(entries []entryItem, filter string) []entryItem {
	if filter == "" {
		return entries
	}
	needle := strings.ToLower(filter)
	out := make([]entryItem, 0)
	for _, e := range entries {
		if e.isParent {
			out = append(out, e)
			continue
		}
		if strings.Contains(strings.ToLower(e.name), needle) {
			out = append(out, e)
		}
	}
	return out
}
```

**Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.gocache go test ./internal/ui -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/entries.go internal/ui/entries_test.go
git commit -m "feat(ui): add entry filtering helpers"
```

---

### Task 5: 双栏 UI 状态机与选择

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/messages.go`
- Modify: `internal/ui/model_test.go`
- Create: `internal/ui/styles.go`
- Modify: `main.go`

**Step 1: Write the failing test**

```go
func TestModelFilterInput(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	m.filterText = ""
	m.focus = focusLeft

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}).(model)
	if m.filterText != "c" {
		t.Fatalf("expected filter 'c', got '%s'", m.filterText)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.gocache go test ./internal/ui -v`
Expected: FAIL (NewModelWithDeps undefined)

**Step 3: Write minimal implementation**

```go
// styles.go
package ui

import "github.com/charmbracelet/lipgloss"

type styles struct {
	paneFocused lipgloss.Style
	paneBlurred lipgloss.Style
	statusBar lipgloss.Style
	header lipgloss.Style
}

func newStyles() styles {
	return styles{
		paneFocused: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("69")),
		paneBlurred: lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("238")),
		statusBar: lipgloss.NewStyle().Foreground(lipgloss.Color("245")),
		header: lipgloss.NewStyle().Bold(true),
	}
}
```

```go
// messages.go (新增)
package ui

type dirEntriesMsg struct {
	path string
	entries []entryItem
	err error
}

type confirmMsg struct {
	result app.ExpandResult
	err error
}
```

```go
// model.go (核心变更，仅示意)
func NewModelWithDeps(cwd string, lister DirLister, expander SelectionExpander) model { /* ... */ }

// Update: 
// - Tab 切换 focus
// - Space 选中/取消
// - KeyRunes 更新 filterText 并刷新列表
// - Backspace 在左栏删除 filterText
// - Enter: 左栏进目录；右栏进入确认层
```

**Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.gocache go test ./internal/ui -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/model.go internal/ui/messages.go internal/ui/model_test.go internal/ui/styles.go

git commit -m "feat(ui): add two-pane browse and filter"
```

---

### Task 6: 确认层与配置步

**Files:**
- Modify: `internal/ui/model.go`
- Modify: `internal/ui/model_test.go`

**Step 1: Write the failing test**

```go
func TestConfirmRequiresSelection(t *testing.T) {
	m := NewModelWithDeps("/tmp", fakeDirLister{}, fakeExpander{})
	m.focus = focusRight
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter}).(model)
	if m.err == nil {
		t.Fatal("expected error on empty selection")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `GOCACHE=$PWD/.gocache go test ./internal/ui -v`
Expected: FAIL (no error handling)

**Step 3: Write minimal implementation**

```go
// 在确认层显示统计信息
// Enter: 进入 config
// Esc: 返回 browse
// config 使用 textinput 编辑输出目录 (默认 ./output)
```

**Step 4: Run test to verify it passes**

Run: `GOCACHE=$PWD/.gocache go test ./internal/ui -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/ui/model.go internal/ui/model_test.go
git commit -m "feat(ui): add confirm and config steps"
```

---

Plan complete and saved to `docs/plans/2026-0114-1424-tui-two-pane-implementation-plan.md`. Two execution options:

1. Subagent-Driven (this session) - I dispatch fresh subagent per task, review between tasks, fast iteration
2. Parallel Session (separate) - Open new session with executing-plans, batch execution with checkpoints

Which approach?
