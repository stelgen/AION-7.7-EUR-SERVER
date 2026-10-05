// Package render — генерация изображения капчи (по стилю default.xml/en.xml оригинала)
// и компрессия в DXT1-текстуру 128×32 (DDS blob для wire-протокола).
//
// Стиль оригинала (XML): 6 цифр «1234567890», тёмный фон 0-30, цветные цифры из палитры,
// поворот ±8°, 2 кривые-помехи (3 точки, ширина 1.7), 200 точек шума 2.3×2.3.
// Шрифт: встроенные битмап-глифы 5×7 с масштабом (TTF-движок не нужен — капча каждый раз
// случайна, пиксель-в-пиксель совпадение с оригиналом не требуется).
package render

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"math"
	"math/rand"
)

// Palette — 10 цветов из default.xml оригинала.
var defaultPalette = []color.RGBA{
	{255, 192, 203, 255}, // розовый
	{255, 165, 0, 255},   // оранжевый
	{255, 255, 0, 255},   // жёлтый
	{173, 255, 47, 255},  // зелёный луга
	{240, 128, 128, 255}, // светло-коралловый
	{245, 255, 250, 255}, // мятный крем
	{255, 255, 255, 255},
	{255, 255, 255, 255},
	{255, 255, 255, 255},
	{255, 255, 255, 255},
}

// digits 5×7 битмапы, «X» = закрашен.
var glyphDigits = map[rune][]string{
	'0': {
		".XXX.",
		"X...X",
		"X...X",
		"X...X",
		"X...X",
		"X...X",
		".XXX.",
	},
	'1': {
		"..X..",
		".XX..",
		"..X..",
		"..X..",
		"..X..",
		"..X..",
		".XXX.",
	},
	'2': {
		".XXX.",
		"X...X",
		"....X",
		"...X.",
		"..X..",
		".X...",
		"XXXXX",
	},
	'3': {
		"XXXXX",
		"...X.",
		"..X..",
		"...X.",
		"....X",
		"X...X",
		".XXX.",
	},
	'4': {
		"...X.",
		"..XX.",
		".X.X.",
		"X..X.",
		"XXXXX",
		"...X.",
		"...X.",
	},
	'5': {
		"XXXXX",
		"X....",
		"XXXX.",
		"....X",
		"....X",
		"X...X",
		".XXX.",
	},
	'6': {
		"..XX.",
		".X...",
		"X....",
		"XXXX.",
		"X...X",
		"X...X",
		".XXX.",
	},
	'7': {
		"XXXXX",
		"....X",
		"...X.",
		"..X..",
		"..X..",
		"..X..",
		"..X..",
	},
	'8': {
		".XXX.",
		"X...X",
		"X...X",
		".XXX.",
		"X...X",
		"X...X",
		".XXX.",
	},
	'9': {
		".XXX.",
		"X...X",
		"X...X",
		".XXXX",
		"....X",
		"...X.",
		".XX..",
	},
}

// Config — параметры генерации (зеркало XML-полей оригинала).
type Config struct {
	Width      int `yaml:"width"`
	Height     int `yaml:"height"`
	Charset    string `yaml:"charset"`
	Length     int    `yaml:"length"`
	FontSize   int    `yaml:"font_size"`
	Bold       bool   `yaml:"font_bold"`
	RotateMin  float64 `yaml:"rotate_min"`
	RotateMax  float64 `yaml:"rotate_max"`
	BgMax      int    `yaml:"bg_max"`
	CurveCount int    `yaml:"curve_count"`
	CurvePts   int    `yaml:"curve_point_count"`
	CurveWidth float64 `yaml:"curve_width"`
	PointCount int    `yaml:"point_count"`
	PointSize  float64 `yaml:"point_size"`

	// внутреннее (заполнить через FillDefaults)
	scale int
}

// FillDefaults — значения из default.xml оригинала (если поля нулевые).
func (c *Config) FillDefaults() {
	if c.Width <= 0 {
		c.Width = 128
	}
	if c.Height <= 0 {
		c.Height = 32
	}
	if c.Charset == "" {
		c.Charset = "1234567890"
	}
	if c.Length <= 0 {
		c.Length = 6
	}
	if c.FontSize <= 0 {
		c.FontSize = 32
	}
	if c.BgMax <= 0 {
		c.BgMax = 30
	}
	if c.CurveCount <= 0 {
		c.CurveCount = 2
	}
	if c.CurvePts <= 0 {
		c.CurvePts = 3
	}
	if c.CurveWidth <= 0 {
		c.CurveWidth = 1.7
	}
	if c.PointCount <= 0 {
		c.PointCount = 200
	}
	if c.PointSize <= 0 {
		c.PointSize = 2.3
	}
	// scale глифа: высота глифа ≈ 0.75 высоты картинки
	s := int(float64(c.Height) * 0.75 / 7)
	if s < 2 {
		s = 2
	}
	if s > 6 {
		s = 6
	}
	c.scale = s
}

// TextGen — генератор случайного текста из charset.
func TextGen(rnd *rand.Rand, cfg Config) string {
	out := make([]byte, cfg.Length)
	for i := range out {
		out[i] = cfg.Charset[rnd.Intn(len(cfg.Charset))]
	}
	return string(out)
}

// Image — RGBA-картинка капчи. text = сгенерированный код.
func Image(rnd *rand.Rand, cfg Config, text string) *image.RGBA {
	cfg.FillDefaults()
	img := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))

	// фон: случайные тёмные оттенки 0..BgMax
	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			r := uint8(rnd.Intn(cfg.BgMax + 1))
			g := uint8(rnd.Intn(cfg.BgMax + 1))
			b := uint8(rnd.Intn(cfg.BgMax + 1))
			img.SetRGBA(x, y, color.RGBA{r, g, b, 255})
		}
	}

	// цифры: равномерно + джиттер, случайные цвета из палитры, случайный поворот
	gw := 5 * cfg.scale
	gh := 7 * cfg.scale
	slot := (cfg.Width - gw) / maxInt(cfg.Length-1, 1)
	for i := 0; i < cfg.Length; i++ {
		ch := rune(text[i])
		mask, ok := glyphDigits[ch]
		if !ok {
			continue
		}
		x0 := i*slot + (slot-gw)/2 + rnd.Intn(7) - 3
		y0 := (cfg.Height-gh)/2 + rnd.Intn(5) - 2
		angle := deg2rad(cfg.RotateMin + rnd.Float64()*(cfg.RotateMax-cfg.RotateMin))
		cx := float64(x0) + float64(gw)/2
		cy := float64(y0) + float64(gh)/2
		col := defaultPalette[rnd.Intn(len(defaultPalette))]
		drawGlyph(img, mask, cfg.scale, x0, y0, cx, cy, angle, col, cfg.Bold)
	}

	// кривые-помехи: квадратичные Безье через CurvePts случайных точек
	for c := 0; c < cfg.CurveCount; c++ {
		pts := make([]float64, 0, cfg.CurvePts*2)
		for i := 0; i < cfg.CurvePts; i++ {
			pts = append(pts, float64(rnd.Intn(cfg.Width)), float64(rnd.Intn(cfg.Height)))
		}
		col := defaultPalette[rnd.Intn(len(defaultPalette))]
		drawQuadBezier(img, pts, cfg.CurveWidth, col)
	}

	// точки шума
	for i := 0; i < cfg.PointCount; i++ {
		x := rnd.Intn(cfg.Width)
		y := rnd.Intn(cfg.Height)
		sz := int(math.Ceil(cfg.PointSize))
		col := defaultPalette[rnd.Intn(len(defaultPalette))]
		for dy := 0; dy < sz; dy++ {
			for dx := 0; dx < sz; dx++ {
				px, py := x+dx, y+dy
				if px < cfg.Width && py < cfg.Height {
					img.SetRGBA(px, py, col)
				}
			}
		}
	}
	return img
}

func deg2rad(d float64) float64 { return d * math.Pi / 180 }

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// drawGlyph — битмап-глиф с масштабом, поворотом вокруг (cx,cy) и толщиной (bold = +1 пиксель).
func drawGlyph(img *image.RGBA, mask []string, scale int, x0, y0 int, cx, cy, angle float64, col color.RGBA, bold bool) {
	h := len(mask)
	if h == 0 {
		return
	}
	w := len(mask[0])
	gw, gh := w*scale, h*scale
	for ty := 0; ty < gh; ty++ {
		for tx := 0; tx < gw; tx++ {
			// обратный поворот цели → исходный пиксель глифа
			dx := float64(tx) + 0.5 - (cx - float64(x0))
			dy := float64(ty) + 0.5 - (cy - float64(y0))
			ca, sa := math.Cos(-angle), math.Sin(-angle)
			sx := dx*ca - dy*sa
			sy := dx*sa + dy*ca
			ux := int(sx/float64(scale)) + w/2
			uy := int(sy/float64(scale)) + h/2
			if ux < 0 || ux >= w || uy < 0 || uy >= h {
				continue
			}
			if mask[uy][ux] != 'X' {
				continue
			}
			px := x0 + tx
			py := y0 + ty
			if px < 0 || py < 0 || px >= img.Rect.Dx() || py >= img.Rect.Dy() {
				continue
			}
			if bold {
				// 2×2 утолщение
				for dy2 := 0; dy2 <= 1; dy2++ {
					for dx2 := 0; dx2 <= 1; dx2++ {
						setIfInside(img, px+dx2, py+dy2, col)
					}
				}
			} else {
				setIfInside(img, px, py, col)
			}
		}
	}
}

func setIfInside(img *image.RGBA, x, y int, col color.RGBA) {
	if x < 0 || y < 0 || x >= img.Rect.Dx() || y >= img.Rect.Dy() {
		return
	}
	img.SetRGBA(x, y, col)
}

// drawQuadBezier — квадратичная Безье через последовательность контрольных точек
// (пары [x y x y ...]), толщина ≈ CurveWidth px.
func drawQuadBezier(img *image.RGBA, pts []float64, width float64, col color.RGBA) {
	if len(pts) < 6 {
		return
	}
	thick := int(math.Max(1, math.Round(width)))
	for i := 0; i+4 <= len(pts); i += 2 {
		x0, y0 := pts[i], pts[i+1]
		x1, y1 := pts[i+2], pts[i+3]
		// середина сегмента — контрольная
		for t := 0.0; t <= 1.0; t += 0.02 {
			mx := (x0 + x1) / 2
			my := (y0 + y1) / 2
			// quadratic: P = (1-t)^2*P0 + 2(1-t)t*C + t^2*P1, C = mid точка выше/ниже на шаг
			jitter := 6.0
			cx := mx + jitter
			cy := my - jitter
			px := (1-t)*(1-t)*x0 + 2*(1-t)*t*cx + t*t*x1
			py := (1-t)*(1-t)*y0 + 2*(1-t)*t*cy + t*t*y1
			for dy := 0; dy < thick; dy++ {
				for dx := 0; dx < thick; dx++ {
					setIfInside(img, int(px)+dx, int(py)+dy, col)
				}
			}
		}
	}
}

// EncodeDXT1 — RGBA → DXT1-блочная компрессия (256 блоков × 8Б для 128×32).
// Возвращает данные текстуры БЕЗ DDS-заголовка (2048 байт).
func EncodeDXT1(img *image.RGBA) ([]byte, error) {
	w := img.Rect.Dx()
	h := img.Rect.Dy()
	if w%4 != 0 || h%4 != 0 {
		return nil, fmt.Errorf("render: size %dx%d не кратна 4", w, h)
	}
	bw, bh := w/4, h/4
	out := make([]byte, bw*bh*8)
	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			encodeBlock(img, bx*4, by*4, out[(by*bw+bx)*8:(by*bw+bx)*8+8])
		}
	}
	return out, nil
}

// rgb565 — упаковать RGB в 565.
func rgb565(r, g, b uint8) uint16 {
	return uint16(r>>3)<<11 | uint16(g>>2)<<5 | uint16(b>>3)
}

func encodeBlock(img *image.RGBA, ox, oy int, out []byte) {
	// собрать 16 пикселей, найти bbox в 565
	var minR, minG, minB uint8 = 255, 255, 255
	var maxR, maxG, maxB uint8
	px := make([]color.RGBA, 0, 16)
	for dy := 0; dy < 4; dy++ {
		for dx := 0; dx < 4; dx++ {
			c := img.RGBAAt(ox+dx, oy+dy)
			px = append(px, c)
			if c.R < minR {
				minR = c.R
			}
			if c.G < minG {
				minG = c.G
			}
			if c.B < minB {
				minB = c.B
			}
			if c.R > maxR {
				maxR = c.R
			}
			if c.G > maxG {
				maxG = c.G
			}
			if c.B > maxB {
				maxB = c.B
			}
		}
	}
	// расширить bbox в стороны (улучшает качество)
	expand := func(v *uint8, lo bool) {
		if lo {
			if *v > 8 {
				*v -= 8
			} else {
				*v = 0
			}
		} else {
			if *v < 248 {
				*v += 8
			} else {
				*v = 255
			}
		}
	}
	expand(&minR, true)
	expand(&minG, true)
	expand(&minB, true)
	expand(&maxR, false)
	expand(&maxG, false)
	expand(&maxB, false)

	c0 := rgb565(maxR, maxG, maxB)
	c1 := rgb565(minR, minG, minB)
	if c1 >= c0 { // гарантируем c0 > c1 (иначе DXT1 интерпретирует как прозрачный блок)
		t := c0
		if t == c1 {
			c1 = t - 1
		} else {
			c0, c1 = c1, t
		}
	}
	// интерполянты
	r0, g0, b0 := unpack565(c0)
	r1, g1, b1 := unpack565(c1)
	r2, g2, b2 := lerp3(r0, g0, b0, r1, g1, b1, 0)
	r3, g3, b3 := lerp3(r0, g0, b0, r1, g1, b1, 1)

	var bits uint32
	for i, c := range px {
		d0 := dist(c, r0, g0, b0)
		d1 := dist(c, r1, g1, b1)
		d2 := dist(c, r2, g2, b2)
		d3 := dist(c, r3, g3, b3)
		var idx uint32
		if d2 <= d0 && d2 <= d1 && d2 <= d3 {
			idx = 2
		} else if d3 <= d0 && d3 <= d1 {
			idx = 3
		} else if d1 <= d0 {
			idx = 1
		}
		bits |= idx << (2 * uint(i))
	}
	out[0] = byte(c0)
	out[1] = byte(c0 >> 8)
	out[2] = byte(c1)
	out[3] = byte(c1 >> 8)
	out[4] = byte(bits)
	out[5] = byte(bits >> 8)
	out[6] = byte(bits >> 16)
	out[7] = byte(bits >> 24)
}

func unpack565(c uint16) (r, g, b uint8) {
	r = uint8((c >> 11) & 0x1F)
	g = uint8((c >> 5) & 0x3F)
	b = uint8(c & 0x1F)
	r = r<<3 | r>>2
	g = g<<2 | g>>4
	b = b<<3 | b>>2
	return
}

func lerp3(r0, g0, b0, r1, g1, b1 uint8, which int) (r, g, b uint8) {
	if which == 0 { // (2*a + b) / 3
		r = uint8((2*int(r0) + int(r1)) / 3)
		g = uint8((2*int(g0) + int(g1)) / 3)
		b = uint8((2*int(b0) + int(b1)) / 3)
	} else { // (a + 2*b) / 3
		r = uint8((int(r0) + 2*int(r1)) / 3)
		g = uint8((int(g0) + 2*int(g1)) / 3)
		b = uint8((int(b0) + 2*int(b1)) / 3)
	}
	return
}

func dist(c color.RGBA, r, g, b uint8) int {
	dr := int(c.R) - int(r)
	dg := int(c.G) - int(g)
	db := int(c.B) - int(b)
	return dr*dr + dg*dg + db*db
}

// ddsHeader128 — fixture header оригинала (снят байт-в-байт с прод-обмена).
var ddsHeader128 = mustParseDDS()

// mustParseDDS — DDS header 128Б, снятый с оригинала (fixture testdata/dds-header.hex):
// dwSize=124, dwFlags=0x1007, 32x128, PF{size=32, flags=FOURCC, fourcc="DXT1"}.
// Строится программно; byte-точность fixture проверяется тестом.
func mustParseDDS() []byte {
	b := make([]byte, 128)
	copy(b[0:4], "DDS ")
	binary.LittleEndian.PutUint32(b[4:], 124)    // dwSize
	binary.LittleEndian.PutUint32(b[8:], 0x1007) // dwFlags (как у оригинала)
	binary.LittleEndian.PutUint32(b[12:], 32)    // dwHeight
	binary.LittleEndian.PutUint32(b[16:], 128)   // dwWidth
	// [20:76] pitch/depth/mipMapCount/reserved = 0
	binary.LittleEndian.PutUint32(b[76:], 32) // PF dwSize
	binary.LittleEndian.PutUint32(b[80:], 4)  // PF flags = DDPF_FOURCC
	copy(b[84:88], "DXT1")                    // fourcc
	// [88:104] rgbCount/masks = 0
	// [104:108] dwCaps1 = 0 (как у оригинала)
	binary.LittleEndian.PutUint32(b[108:], 0x1000) // dwCaps2 (fixture)
	// [112:128] reserved = 0
	return b
}


// DDSBlobOf — header + DXT1 data (2176 = 128 + 2048).
func DDSBlobOf(data []byte) ([]byte, error) {
	if len(data) != 2048 {
		return nil, fmt.Errorf("render: dxt1 data=%d want 2048", len(data))
	}
	out := make([]byte, 0, 128+len(data))
	out = append(out, ddsHeader128...)
	out = append(out, data...)
	return out, nil
}

// UTF16LE — ASCII-текст в UTF-16LE (12 байт для 6 символов).
func UTF16LE(text string) ([]byte, error) {
	for _, ch := range text {
		if ch > 0x7F {
			return nil, fmt.Errorf("render: не-ASCII символ %q", ch)
		}
	}
	out := make([]byte, len(text)*2)
	for i := 0; i < len(text); i++ {
		out[i*2] = text[i]
		out[i*2+1] = 0
	}
	return out, nil
}
