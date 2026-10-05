package render

import (
	"image/color"
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestRenderSizeAndText(t *testing.T) {
	cfg := Config{}
	cfg.FillDefaults()
	rnd := rand.New(rand.NewSource(1))
	text := TextGen(rnd, cfg)
	if len(text) != 6 {
		t.Fatalf("text=%q", text)
	}
	if strings.Trim(text, "1234567890") != "" {
		t.Fatalf("text=%q вне charset", text)
	}
	img := Image(rnd, cfg, text)
	if img.Rect.Dx() != 128 || img.Rect.Dy() != 32 {
		t.Fatalf("size=%v", img.Rect)
	}
	// картинка не пустая: есть пиксели ярче фона (цифры)
	bright := 0
	for y := 0; y < 32; y++ {
		for x := 0; x < 128; x++ {
			c := img.RGBAAt(x, y)
			if c.R > 80 || c.G > 80 || c.B > 80 {
				bright++
			}
		}
	}
	if bright < 200 {
		t.Fatalf("bright=%d — цифры не нарисованы", bright)
	}
}

func TestDXT1LengthAndHeader(t *testing.T) {
	cfg := Config{}
	cfg.FillDefaults()
	rnd := rand.New(rand.NewSource(2))
	img := Image(rnd, cfg, "757084")
	data, err := EncodeDXT1(img)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != 2048 {
		t.Fatalf("dxt1 len=%d", len(data))
	}
	dds, err := DDSBlobOf(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(dds) != 2176 {
		t.Fatalf("dds len=%d", len(dds))
	}
	if string(dds[0:4]) != "DDS " {
		t.Fatalf("magic=%q", dds[0:4])
	}
	// fourcc = DXT1
	if string(dds[84:88]) != "DXT1" {
		t.Fatalf("fourcc=%q", dds[84:88])
	}
}

func TestRenderSpeed(t *testing.T) {
	cfg := Config{}
	cfg.FillDefaults()
	rnd := rand.New(rand.NewSource(3))
	// прогрев
	Image(rnd, cfg, "000000")
	start := time.Now()
	const n = 60
	for i := 0; i < n; i++ {
		text := TextGen(rnd, cfg)
		img := Image(rnd, cfg, text)
		if _, err := EncodeDXT1(img); err != nil {
			t.Fatal(err)
		}
	}
	dur := time.Since(start)
	per := dur / n
	t.Logf("per-image: %v (rate %.1f/s)", per, float64(time.Second)/float64(per))
	if per > 30*time.Millisecond {
		t.Fatalf("слишком медленно: %v/картинку", per)
	}
}

func TestUTF16LE(t *testing.T) {
	b, err := UTF16LE("757084")
	if err != nil {
		t.Fatal(err)
	}
	want := []byte{0x37, 0x00, 0x35, 0x00, 0x37, 0x00, 0x30, 0x00, 0x38, 0x00, 0x34, 0x00}
	for i := range want {
		if b[i] != want[i] {
			t.Fatalf("byte %d: %02x want %02x", i, b[i], want[i])
		}
	}
	if _, err := UTF16LE("75708а"); err == nil { // кириллическая 'а' — не ASCII
		t.Fatal("должна быть ошибка на не-ASCII")
	}
}

func TestPaletteUsed(t *testing.T) {
	cfg := Config{}
	cfg.FillDefaults()
	rnd := rand.New(rand.NewSource(4))
	img := Image(rnd, cfg, "123456")
	found := map[color.RGBA]bool{}
	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			found[img.RGBAAt(x, y)] = true
		}
	}
	// белый из палитры должен присутствовать
	if !found[color.RGBA{255, 255, 255, 255}] {
		t.Fatal("белый из палитры не найден")
	}
}
