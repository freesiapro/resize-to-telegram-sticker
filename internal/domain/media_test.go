package domain

import "testing"

func TestDetectInputKind(t *testing.T) {
	cases := []struct {
		path string
		want InputKind
	}{
		{"a.mp4", InputKindVideo},
		{"a.png", InputKindImage},
		{"a.GIF", InputKindGIF},
	}

	for _, c := range cases {
		got, err := DetectInputKind(c.path)
		if err != nil {
			t.Fatalf("unexpected err: %v", err)
		}
		if got != c.want {
			t.Fatalf("path=%s got=%s want=%s", c.path, got, c.want)
		}
	}
}

func TestDetectInputKindUnsupported(t *testing.T) {
	_, err := DetectInputKind("a.txt")
	if err == nil {
		t.Fatal("expected error")
	}
}
