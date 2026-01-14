package ui

import "testing"

func TestCalcPaneLayoutWidths(t *testing.T) {
	layout := calcPaneLayout(80, 24)
	if layout.leftWidth+layout.rightWidth+layout.dividerWidth != 80 {
		t.Fatalf("width mismatch: %+v", layout)
	}
	if layout.contentHeight != 23 {
		t.Fatalf("unexpected content height: %d", layout.contentHeight)
	}
	if layout.leftInnerWidth > layout.leftWidth {
		t.Fatalf("inner width exceeds outer: %+v", layout)
	}
	if layout.rightInnerWidth > layout.rightWidth {
		t.Fatalf("inner width exceeds outer: %+v", layout)
	}
	if layout.listHeight < 1 {
		t.Fatalf("expected list height >= 1, got %d", layout.listHeight)
	}
}

func TestContentSizePadding(t *testing.T) {
	width, height := contentSize(80, 24)
	if width != 80-2*outerPadX {
		t.Fatalf("unexpected width: %d", width)
	}
	if height != 24-2*outerPadY {
		t.Fatalf("unexpected height: %d", height)
	}
}

func TestContentSizeMinimum(t *testing.T) {
	width, height := contentSize(1, 1)
	if width != 1 || height != 1 {
		t.Fatalf("expected minimum size 1x1, got %dx%d", width, height)
	}
}
