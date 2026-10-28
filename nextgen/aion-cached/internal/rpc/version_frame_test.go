package rpc

import "testing"

// TestO16VersionFrame — v20 пуш-кадр байт-в-байт = capture 24.10 (O16 27Б).
func TestO16VersionFrame(t *testing.T) {
	want := "1b001000afefff9c0f03007c000000d80000000000800000000000" // capture: payload = 9c0f0300 7c000000 d8000000 00008000 00000000
	got := O16VersionFrame()
	if len(got) != 27 {
		t.Fatalf("len=%d, want 27", len(got))
	}
	if gotHex := string(bytes2hex(got)); gotHex != want {
		t.Fatalf("hex mismatch\n got %s\nwant %s", gotHex, want)
	}
}

func bytes2hex(b []byte) []byte {
	const hexd = "0123456789abcdef"
	out := make([]byte, 0, len(b)*2)
	for _, c := range b {
		out = append(out, hexd[c>>4], hexd[c&0xf])
	}
	return out
}
