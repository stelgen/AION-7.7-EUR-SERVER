// Package proto — wire-протокол CAPTCHAImageServer (TCP).
//
// Фрейм: [u16 LE totalLen][u16 LE type][payload]; totalLen — полный размер пакета
// (включая поля len и type). seq — u16 счётчик ПЕР-ТИП, с 1 (зеркалим в ответе).
//
// Типы (capture 05.10.2026, 10001+10001 пакетов, docs/captcha-protocol-20261005.md):
//   101  CAPTCHA_LANGUAGE_CHECK      C→S  total 8  [seq u16][langCode u16 (bytes 6e 65 = "en")]
//   102  CAPTCHA_LANGUAGE_CHECK_REPLY S→C total 8  эхо seq+langCode
//   1001 CAPTCHA_REQUEST            C→S  total 18 [seq u16][u32 0][u32 codepage=1200][langCode u16]
//   1002 CAPTCHA_REPLY              S→C total 2212 [seq u16][u32 0][u32 0][u16 0][u16 imageSize=2176][u16 3074 const][DDS blob 2176][text UTF-16LE 12B][u32 0]
package proto

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

const (
	TypeLangCheck       uint16 = 101
	TypeLangCheckReply  uint16 = 102
	TypeCaptchaRequest  uint16 = 1001
	TypeCaptchaReply    uint16 = 1002
)

// Константы wire-протокола (фиксированы на 10000 живых пакетах).
const (
	CodepageUTF16       uint32 = 1200 // <character_codepage> в XML
	DDSImageSize        uint16 = 2176 // 128 header + 2048 DXT1 (128×32)
	ReplyFieldU16       uint16 = 3074 // поле [18:20] = 0x0C02; смысл не раскрыт, копируем const
	LangCodeBytes              = 2    // «en» в BE-порядке: 6e 65
	LangCheckTotal             = 8
	LangCheckReplyTotal        = 8
	ReqTotal                   = 18
	// ReplyTotal вычисляется: 20 header + imageSize + text 12 + 4 = 2212
	ReplyHeaderLen = 16 // body-head до DDS (в body; wire-заголовок [len][type] — ещё 4)
	ReplyTailLen   = 16 // 12Б текст (6 chars UTF-16LE) + 4Б NUL
)

var ErrShort = errors.New("proto: frame short")
var ErrType = errors.New("proto: unsupported type")

// Frame — разобранный пакет.
type Frame struct {
	Type uint16
	Body []byte // всё после [len][type]
}

// En — кодировка языка «en» как u16 BE-упаковка (байты 6e 65).
func En(lang string) uint16 {
	if len(lang) != 2 {
		return 0
	}
	return uint16(lang[0])<<8 | uint16(lang[1])
}

// UnEn — обратно в 2-символьную строку.
func UnEn(v uint16) string {
	return string([]byte{byte(v >> 8), byte(v)})
}

// ReadFrame читает один фрейм из потока (блокируя до полного).
func ReadFrame(r io.Reader, buf []byte) (Frame, []byte, error) {
	if len(buf) < 4 {
		return Frame{}, buf, errors.New("proto: small read buffer")
	}
	if _, err := io.ReadFull(r, buf[:4]); err != nil {
		return Frame{}, buf, err
	}
	total := int(binary.LittleEndian.Uint16(buf[:2]))
	if total < 4 || total > 0x2000 {
		return Frame{}, buf, fmt.Errorf("%w: total=%d", ErrShort, total)
	}
	if _, err := io.ReadFull(r, buf[4:total]); err != nil {
		return Frame{}, buf, err
	}
	return Frame{Type: binary.LittleEndian.Uint16(buf[2:4]), Body: buf[4:total]}, buf, nil
}

// Build — собрать пакет [len][type][body].
func Build(t uint16, body []byte) []byte {
	total := 4 + len(body)
	out := make([]byte, total)
	binary.LittleEndian.PutUint16(out[0:], uint16(total))
	binary.LittleEndian.PutUint16(out[2:], t)
	copy(out[4:], body)
	return out
}

// BuildLangCheckReply: [seq u16][langCode u16] = 4Б body.
func BuildLangCheckReply(seq, lang uint16) []byte {
	body := make([]byte, 4)
	binary.LittleEndian.PutUint16(body[0:], seq)
	binary.LittleEndian.PutUint16(body[2:], lang)
	return Build(TypeLangCheckReply, body)
}

// ParseLangCheck разбирает 101: body = [seq u16][langCode u16].
func ParseLangCheck(b []byte) (seq, lang uint16, err error) {
	if len(b) != 4 {
		return 0, 0, fmt.Errorf("%w: langcheck body=%d", ErrType, len(b))
	}
	return binary.LittleEndian.Uint16(b), binary.LittleEndian.Uint16(b[2:]), nil
}

// ParseLangCheckReply — то же тело (эхо).
func ParseLangCheckReply(b []byte) (seq, lang uint16, err error) {
	return ParseLangCheck(b)
}

// BuildRequest: [seq u16][u32 0][u16 0][codepage u32][langCode u16] = 14Б body (для фейк-клиента).
func BuildRequest(seq, lang uint16, codepage uint32) []byte {
	body := make([]byte, 14)
	binary.LittleEndian.PutUint16(body[0:], seq)      // [0:2] seq
	// [2:8] = 6Б нулей (u32 0 + u16 0)
	binary.LittleEndian.PutUint32(body[8:], codepage) // [8:12] codepage
	binary.LittleEndian.PutUint16(body[12:], lang)    // [12:14] langCode
	return Build(TypeCaptchaRequest, body)
}

// ParseRequest разбирает 1001: body = [seq u16][u32 0][u16 0][codepage u32][langCode u16].
func ParseRequest(b []byte) (seq uint16, codepage uint32, lang uint16, err error) {
	if len(b) != 14 {
		return 0, 0, 0, fmt.Errorf("%w: req body=%d", ErrType, len(b))
	}
	seq = binary.LittleEndian.Uint16(b)
	codepage = binary.LittleEndian.Uint32(b[8:])
	lang = binary.LittleEndian.Uint16(b[12:])
	return seq, codepage, lang, nil
}

// BuildReply: [seq u16][u32 0][u32 0][u16 0][u16 imageSize][u16 0x0C02][dds blob][text 12B][u32 0].
// body = 16 + len(dds) + 16.
func BuildReply(seq uint16, dds []byte, textUTF16 []byte) []byte {
	body := make([]byte, 16+len(dds)+len(textUTF16)+4)
	binary.LittleEndian.PutUint16(body[0:], seq)              // [0:2]
	// [2:6][6:10] u32 0 (нулевые)
	binary.LittleEndian.PutUint16(body[10:], 0)               // [10:12] u16 0
	binary.LittleEndian.PutUint16(body[12:], uint16(len(dds))) // [12:14] imageSize=2176
	binary.LittleEndian.PutUint16(body[14:], ReplyFieldU16)   // [14:16] 0x0C02
	copy(body[16:], dds)                                      // [16:16+len(dds)] DDS blob
	copy(body[16+len(dds):], textUTF16)                       // текст UTF-16LE
	// хвост [.. +4] u32 0 (нулевые)
	return Build(TypeCaptchaReply, body)
}

// Reply — разобранный 1002.
type Reply struct {
	Seq       uint16
	DDS       []byte // 2176 (header+data)
	TextUTF16 []byte // 12Б (6 chars)
}

// ParseReply разбирает 1002-тело.
func ParseReply(b []byte) (Reply, error) {
	if len(b) < 20 {
		return Reply{}, fmt.Errorf("%w: reply body=%d", ErrType, len(b))
	}
	imgSize := binary.LittleEndian.Uint16(b[12:])
	field := binary.LittleEndian.Uint16(b[14:])
	tail := len(b) - 16 - int(imgSize)
	if tail != ReplyTailLen {
		return Reply{}, fmt.Errorf("%w: reply tail=%d want %d (imgSize=%d field=0x%04x)", ErrType, tail, ReplyTailLen, imgSize, field)
	}
	return Reply{
		Seq:       binary.LittleEndian.Uint16(b),
		DDS:       b[16 : 16+int(imgSize)],
		TextUTF16: b[16+int(imgSize) : 16+int(imgSize)+12],
	}, nil
}

// ReplyTotalOf — полный размер пакета-ответа для данного размера картинки.
func ReplyTotalOf(ddsLen int) int { return 4 + ReplyHeaderLen + ddsLen + ReplyTailLen }
