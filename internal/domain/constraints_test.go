package domain

import "testing"

func TestScaleToFit(t *testing.T) {
	cases := []struct {
		name string
		src  Size
		want Size
	}{
		{"wider", Size{Width: 1000, Height: 500}, Size{Width: 512, Height: 256}},
		{"taller", Size{Width: 500, Height: 1000}, Size{Width: 256, Height: 512}},
		{"square", Size{Width: 512, Height: 512}, Size{Width: 512, Height: 512}},
	}

	for _, c := range cases {
		got, err := ScaleToFit(c.src, MaxStickerSide)
		if err != nil {
			t.Fatalf("%s: unexpected err: %v", c.name, err)
		}
		if got != c.want {
			t.Fatalf("%s: got=%+v want=%+v", c.name, got, c.want)
		}
	}
}

func TestScaleToFitInvalid(t *testing.T) {
	_, err := ScaleToFit(Size{Width: 0, Height: 10}, MaxStickerSide)
	if err == nil {
		t.Fatal("expected error")
	}
}
