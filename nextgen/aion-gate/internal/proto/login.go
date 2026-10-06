package proto

import "encoding/binary"

// CM_LOGIN (op 0x00, state AUTHED_GG) — раскладка из гита beyond-aion 4.8
// (docs/beyond-aion-48-protocol-vs-gate-20261006.md §4):
//   pt = [op 1B][RSA ct — кратно 128B][tail 55B]
//   tail 55 = [sid D][16×0][7B: 20 00 00 00 00 00 01][16B: 9D DA 47 A7 21 C0 A6 A5
//             4B B7 5E E3 CE C9 26 AA][D 0][D unk][D 0]
//   (для 7.7 EU клиента константы magic другой сборки — сверяем СТРУКТУРУ, не байты)
//   isLoginEx = число чанков k > 1.
// Раскладка расшифрованного чанка (128B BE):
//   не-loginex: username @94:108 (14B, до нуля), password @108:124 (16B, до нуля),
//               otp LE i32 @124:128 (FFFFFFFF = не используется); ведущие нули — паддинг
//   loginex:    username = чанк1[78:142] (64B), password = чанк2[78:110] (32B),
//               otp = чанк2[110:114]

// LoginTailLen — фикс-хвост CM_LOGIN.
const LoginTailLen = 55

// SplitLogin режет plaintext CM_LOGIN на RSA-чанки (по 128B, ct) и хвост 55B.
// ok=false — форма не сходится (клиент другой сборки) → вызывающий релеит raw.
func SplitLogin(pt []byte) (chunks [][]byte, tail []byte, ok bool) {
	if len(pt) < 1+128+LoginTailLen {
		return nil, nil, false
	}
	tail = pt[len(pt)-LoginTailLen:]
	ct := pt[1 : len(pt)-LoginTailLen]
	if len(ct)%128 != 0 || len(ct) == 0 {
		return nil, nil, false
	}
	k := len(ct) / 128
	for i := 0; i < k; i++ {
		chunks = append(chunks, ct[i*128:(i+1)*128])
	}
	return chunks, tail, true
}

// ReadCStr — ASCII до нуля (charset Cp1252; для ASCII-подмножества совпадает).
func ReadCStr(b []byte) string {
	for i, c := range b {
		if c == 0 {
			return string(b[:i])
		}
	}
	return string(b)
}

func isPrintableASCII(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < 0x20 || s[i] > 0x7e {
			return false
		}
	}
	return true
}

// DecodedLogin — результат DecodeLoginPlain.
type DecodedLogin struct {
	User string
	Pwd  string
	Otp  uint32
	Ex   bool // loginex (k>1)
}

// DecodeLoginPlain расшифрованные чанки → креды по раскладке гита.
// Валидатор (готовый оракул для выбора e): username printable, otp == 0xFFFFFFFF.
// loginex (k>1): чанки СКЛЕИВАЮТСЯ в один буфер и режутся по общим смещениям
// (гит CM_LOGIN.decryptLoginData): user = [78:142], pwd = [128+78 : 128+110],
// otp = [128+110 : 128+114] — «password @78 в чанке 2, otp следом».
func DecodeLoginPlain(ms [][]byte) (DecodedLogin, bool) {
	var d DecodedLogin
	switch {
	case len(ms) == 1:
		m := ms[0]
		d.User = ReadCStr(m[94:108])
		d.Pwd = ReadCStr(m[108:124])
		d.Otp = binary.LittleEndian.Uint32(m[124:128])
	case len(ms) >= 2:
		d.Ex = true
		buf := make([]byte, 0, len(ms)*128)
		for _, m := range ms {
			buf = append(buf, m...)
		}
		if len(buf) < 242 {
			return d, false
		}
		d.User = ReadCStr(buf[78:142])
		d.Pwd = ReadCStr(buf[206:238])
		d.Otp = binary.LittleEndian.Uint32(buf[238:242])
	default:
		return d, false
	}
	ok := isPrintableASCII(d.User) && isPrintableASCII(d.Pwd) && d.Otp == 0xFFFFFFFF
	return d, ok
}

// LoginDecbuf — decbuf для authd-blob "cbdb": срез полного m по loginDecbufLen
// (34 = user14+pwd16+otp4 ровно по раскладке; 32 = i2osp 0x20 asm-модели; 128 = полный m).
func LoginDecbuf(m []byte, decbufLen int) []byte {
	if decbufLen <= 0 || decbufLen > len(m) {
		decbufLen = 34
	}
	return m[len(m)-decbufLen:]
}