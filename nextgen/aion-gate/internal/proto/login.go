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

// Хвост CM_LOGIN у 7.7 клиента ВАРИАТИВНЫЙ (live 06.10: pt=304 = 1+128+128+47;
// rsa-hunt видел 312 = 1+256+55) — фикс-55 из гита НЕ работает. Правило: k = (len-1)/128,
// хвост = остаток (≤64Б). Live-структура хвоста 47Б: [sid LE][нули][0x20][7×0]
// [68ffdab3e2fda892][2d9cc7baa87e0d49][00000000].
const LoginTailMax = 64

// SplitLogin режет plaintext CM_LOGIN на op, RSA-чанки (по 128B, ct) и вариативный хвост.
// К-3 (P1-3): op = pt[0] ВОЗВРАЩАЕТСЯ и логируется вызывающим (эталон 7.7: CM_LOGIN
// = op 0x0B в AUTHED_GG); ct/tail от значения op не зависят (не смещаются).
// ok=false — форма не сходится → вызывающий релеит raw.
func SplitLogin(pt []byte) (op byte, chunks [][]byte, tail []byte, ok bool) {
	if len(pt) < 1+128 {
		return 0, nil, nil, false
	}
	body := len(pt) - 1
	k := body / 128
	rem := body % 128
	if k == 0 || rem > LoginTailMax {
		return 0, nil, nil, false
	}
	ct := pt[1 : 1+k*128]
	tail = pt[1+k*128:]
	for i := 0; i < k; i++ {
		chunks = append(chunks, ct[i*128:(i+1)*128])
	}
	return pt[0], chunks, tail, true
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
	User   string
	Pwd    string
	PwdHex string // RAW-байты поля пароля (hex) — во 2-м блоке 7.7 есть иные данные
	Otp    uint32
	Ex     bool // loginex (k>1)
	// К-4 (P1-4): k=1 — раскладки. Layout = что сработало: "48" (байон-4.8: user@94:108,
	// pwd@108:124, otp LE @124), "77" (эталон 7.7: user=m[64:96](32), pwd=m[96:128](32),
	// otp LE m[124:128]) или "ex" (loginex). 4.8 пробуется ПЕРВОЙ, 7.7 — при не-printable.
	Layout string
	// Alt77 — вторая (не сработавшая/проигравшая) гипотеза 7.7 для k=1: ОБЕ гипотезы
	// возвращаются для лога (первый живой k=1 покажет верную раскладку).
	Alt77User string
	Alt77Pwd  string
}

// DecodeLoginPlain расшифрованные чанки → креды по раскладке гита.
// КАЛИБРОВКА 06.10 (живой ct юзера, e=65537 HIT): loginex-блоки, user =
// combined[78:142] (блок1[78:128]+блок2[0:14]) — ПОДТВЕРЖДЕНО; pwd = combined[206:238],
// otp = combined[238:242] — во 2-м блоке 7.7 есть иные данные (ct2 не сводится к
// простым формам), поэтому pwd/otp — best-effort (authd пароль игнорирует).
// Валидатор: username printable (обязательно); pwd/otp логируются как есть.
func DecodeLoginPlain(ms [][]byte) (DecodedLogin, bool) {
	var d DecodedLogin
	switch {
	case len(ms) == 1:
		m := ms[0]
		// Гипотеза 1 (байон-4.8, прежнее поведение): user@94:108(14), pwd@108:124(16), otp LE @124.
		d.User = ReadCStr(m[94:108])
		d.Pwd = ReadCStr(m[108:124])
		d.PwdHex = hexEncode(m[108:124])
		d.Otp = binary.LittleEndian.Uint32(m[124:128])
		d.Layout = "48"
		if isPrintableASCII(d.User) {
			return d, true
		}
		// К-4 (P1-4): гипотеза 2 — эталон 7.7: user=m[64:96](32), pwd=m[96:128](32),
		// otp = LE m[124:128]. Валидатор тот же: username printable.
		d.Alt77User = ReadCStr(m[64:96])
		d.Alt77Pwd = ReadCStr(m[96:128])
		if isPrintableASCII(d.Alt77User) {
			d.User = d.Alt77User
			d.Pwd = d.Alt77Pwd
			d.PwdHex = hexEncode(m[96:128])
			d.Layout = "77"
			return d, true
		}
		return d, false // ни 4.8, ни 7.7 не дали printable user — обе гипотезы в логе вызывающего
	case len(ms) >= 2:
		d.Ex = true
		buf := make([]byte, 0, len(ms)*128)
		for _, m := range ms {
			buf = append(buf, m...)
		}
		if len(buf) < 242 {
			return d, false
		}
		d.Layout = "ex"
		d.User = ReadCStr(buf[78:142])
		d.Pwd = ReadCStr(buf[206:238])
		d.PwdHex = hexEncode(buf[206:238])
		d.Otp = binary.LittleEndian.Uint32(buf[238:242])
	default:
		return d, false
	}
	return d, isPrintableASCII(d.User)
}

func hexEncode(b []byte) string {
	const h = "0123456789abcdef"
	s := make([]byte, 0, len(b)*2)
	for _, c := range b {
		s = append(s, h[c>>4], h[c&15])
	}
	return string(s)
}

// BuildLoginDecbuf — decbuf для authd-blob "cbdb", собирается из РЕАЛЬНЫХ полей
// (asm оригинала arg3=0x22=34 = user14+pwd16+otp4; работает одинаково для обеих форм —
// не-loginex эти же байты лежат в m[94:128], loginex собирается из склейки чанков).
func BuildLoginDecbuf(user, pwd string, otp uint32, decbufLen int) []byte {
	pad := func(s string, n int) []byte {
		b := []byte(s)
		if len(b) > n {
			b = b[:n]
		}
		out := make([]byte, n)
		copy(out, b)
		return out
	}
	var ob []byte
	switch decbufLen {
	case 32:
		oy := otp
		ob = []byte{byte(oy), byte(oy >> 8)}
		out := append(pad(user, 14), pad(pwd, 16)...)
		return append(out, ob...)
	case 128:
		out := make([]byte, 128)
		copy(out[94:108], pad(user, 14))
		copy(out[108:124], pad(pwd, 16))
		binary.LittleEndian.PutUint32(out[124:128], otp)
		return out
	default: // 34
		out := append(pad(user, 14), pad(pwd, 16)...)
		return binary.LittleEndian.AppendUint32(out, otp)
	}
}
