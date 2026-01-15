package core

type PaneLayout struct {
	ContentHeight   int
	InnerHeight     int
	ListHeight      int
	DividerWidth    int
	BorderSize      int
	LeftWidth       int
	RightWidth      int
	LeftInnerWidth  int
	RightInnerWidth int
}

const (
	outerPadX  = 2
	outerPadY  = 1
	headerPadX = 1
)

func ContentSize(width, height int) (int, int) {
	w := width - 2*outerPadX
	h := height - 2*outerPadY
	if w < 1 {
		w = 1
	}
	if h < 1 {
		h = 1
	}
	return w, h
}

func CalcPaneLayout(width, height int) PaneLayout {
	const statusHeight = 3
	const headerHeight = 1

	contentHeight := height - statusHeight
	if contentHeight < 1 {
		contentHeight = 1
	}

	dividerWidth := 1
	if width < 3 {
		dividerWidth = 0
	}

	leftWidth := (width - dividerWidth) / 2
	rightWidth := width - dividerWidth - leftWidth
	if leftWidth < 1 {
		leftWidth = 1
	}
	if rightWidth < 1 {
		rightWidth = 1
	}

	borderSize := 1
	if contentHeight < 3 || leftWidth < 3 || rightWidth < 3 {
		borderSize = 0
	}

	leftInnerWidth := leftWidth - 2*borderSize
	if leftInnerWidth < 1 {
		leftInnerWidth = 1
	}
	rightInnerWidth := rightWidth - 2*borderSize
	if rightInnerWidth < 1 {
		rightInnerWidth = 1
	}

	innerHeight := contentHeight - 2*borderSize
	if innerHeight < 1 {
		innerHeight = 1
	}

	listHeight := innerHeight - headerHeight
	if listHeight < 1 {
		listHeight = 1
	}

	return PaneLayout{
		ContentHeight:   contentHeight,
		InnerHeight:     innerHeight,
		ListHeight:      listHeight,
		DividerWidth:    dividerWidth,
		BorderSize:      borderSize,
		LeftWidth:       leftWidth,
		RightWidth:      rightWidth,
		LeftInnerWidth:  leftInnerWidth,
		RightInnerWidth: rightInnerWidth,
	}
}
