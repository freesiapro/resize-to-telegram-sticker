package infra

import "testing"

func TestParseProbe(t *testing.T) {
	jsonStr := `{"streams":[{"codec_type":"video","width":512,"height":256,"r_frame_rate":"30/1","codec_name":"vp9"},{"codec_type":"audio"}],"format":{"format_name":"webm","duration":"2.9","bit_rate":"1234567"}}`

	info, err := parseProbeJSON([]byte(jsonStr))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if info.Width != 512 || info.Height != 256 {
		t.Fatalf("unexpected size: %+v", info)
	}
	if info.HasAudio != true {
		t.Fatalf("expected audio")
	}
	if info.FPS != 30 {
		t.Fatalf("unexpected fps: %v", info.FPS)
	}
	if info.FormatName != "webm" || info.CodecName != "vp9" {
		t.Fatalf("unexpected format/codec: %+v", info)
	}
	if info.BitrateBps != 1234567 {
		t.Fatalf("unexpected bitrate: %d", info.BitrateBps)
	}
}

func TestParseFrameRate(t *testing.T) {
	fps := parseFrameRate("30000/1001")
	if fps < 29.9 || fps > 30.1 {
		t.Fatalf("unexpected fps: %v", fps)
	}
}
