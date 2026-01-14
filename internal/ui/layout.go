package ui

type paneLayout struct {
	contentHeight   int
	innerHeight     int
	listHeight      int
	dividerWidth    int
	borderSize      int
	leftWidth       int
	rightWidth      int
	leftInnerWidth  int
	rightInnerWidth int
}

func calcPaneLayout(width, height int) paneLayout {
	const statusHeight = 1
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

	return paneLayout{
		contentHeight:   contentHeight,
		innerHeight:     innerHeight,
		listHeight:      listHeight,
		dividerWidth:    dividerWidth,
		borderSize:      borderSize,
		leftWidth:       leftWidth,
		rightWidth:      rightWidth,
		leftInnerWidth:  leftInnerWidth,
		rightInnerWidth: rightInnerWidth,
	}
}
