package proto

import "encoding/binary"

// Welcome (2106): 194B = [u16 LE len=0xC2][ECB(key1) 192B].
// РАСКЛАДКА 100% (ночь-4, верифицирована фреш-дампом оригинала + xrefs):
//	[0:4]   = fc = SID = rand32 (+0x43c184-база) @0x4041b8 (dword0 НЕ скрамблится!)
//	[4:8]   = V  = authd [03]-payload ([global+0xa0], пишет 0x405da4)
//	[8:136] = scrambleModulus(RSA-1024 модуль) 128Б
//	[136:152] = GG-блоб 16Б (нули при useGameGuard=false; 0x407a10)
//	[152:168] = key2 16Б ([esi+0x21a4], клиент узнаёт key2 из welcome)
//	[168:172] = image-статики: S-byte(0x65), B0(0x65), B1(0x00), B2(0x72)
//	[172]   = 'c'-байт va[0]=0x0 (push 0x0 @0x407e01)
// welcomeExtra4 = add eax,0x4 (@0x407e16): 173+4=177 → roundup8=184 →
//	csum(финальный cumsum)@184 → ECB 192 → wire 194.
// Старики-интерпретации: «plaintext[0]=0x23» = fc&0xFF (fc=0x7d521423),
// «sid 0x7d5214» = fc>>8, «PlainByte/B0/B1/B2 из конфига» — image-статики.

const (
	WelcomeLen      = 194
	WelcomePlainLen = 173
	// welcomeExtra4 — удлинение длины на 4 (asm 0x407e16), содержимое = нули.
	welcomeExtra4 = 4
)

// WelcomeArgs — аргументы сборки welcome (asm 0x407d50, ночь-4 финал).
type WelcomeArgs struct {
	SessionID    uint32    // fc = rand32 (+0x43c184) @0x4041b8
	AuthdSession uint32    // V = authd [03]-payload @0x405da4
	Modulus      [128]byte // RAW RSA-1024 модуль (скрамблится внутри)
	GGQuery      [16]byte  // нули при useGameGuard=false (0x407a10)
	Key2         [16]byte  // [esi+0x21a4] — key2 для клиента
}

func (a *WelcomeArgs) plain() []byte {
	mod := a.Modulus
	ScrambleModulus(&mod)
	b := make([]byte, 0, 173)
	b = binary.LittleEndian.AppendUint32(b, a.SessionID)    // pt[0:4] = fc
	b = binary.LittleEndian.AppendUint32(b, a.AuthdSession) // pt[4:8] = V
	b = append(b, mod[:]...)      // 128
	b = append(b, a.GGQuery[:]...) // 16
	b = append(b, a.Key2[:]...)    // 16
	b = append(b, 0x65, 0x65, 0x00, 0x72) // S,B0,B1,B2 (image-статики)
	b = append(b, 0x00)           // 'c'-байт va[0]=0x0
	return b                      // 173
}

// BuildWelcome собирает welcome-пакет целиком (194B), шифрованный key1.
func BuildWelcome(a *WelcomeArgs, bf *Blowfish) []byte {
	plain := append(a.plain(), make([]byte, welcomeExtra4)...) // 173+4=177
	return WriteFrame(EncryptPrimary(bf, plain))
}

// BuildWelcomeVariant — пробная сборка для живого перебора раскладки через оракул
// (frame-32 от клиента = вариант принят). Варианты:
//	0 = базовый (fc,V,mod-scramble,gg-нули,key2,статки 65650072,хвост 0x00,+4)
//	1 = хвост-байт 0x23 (PlainByte-гипотеза)
//	2 = статики [168:172] = нули (без 65650072)
//	3 = GG-зона = 16×0xFF (проверка валидации зоны)
//	4 = БЕЗ скрамбла модуля (raw)
//	5 = csum = XOR дворов (как DecryptSecondary) вместо cumsum
//	6 = fc фикс. 0x7d521423 (из capture)
//	7 = V фикс. 0x634692d8 (из capture)
//	9 = без welcomeExtra4 (pt 172 → wire 186, тест длины)
func BuildWelcomeVariant(a *WelcomeArgs, bf *Blowfish, variant int) []byte {
	mod := a.Modulus
	if variant != 4 {
		ScrambleModulus(&mod)
	}
	b := make([]byte, 0, 176)
	fc := a.SessionID
	if variant == 6 {
		fc = 0x7d521423
	}
	b = binary.LittleEndian.AppendUint32(b, fc)
	v := a.AuthdSession
	if variant == 7 {
		v = 0x634692d8
	}
	b = binary.LittleEndian.AppendUint32(b, v)
	b = append(b, mod[:]...) // 128
	if variant == 3 {
		for i := 0; i < 16; i++ {
			b = append(b, 0xFF)
		}
	} else {
		b = append(b, a.GGQuery[:]...) // 16
	}
	b = append(b, a.Key2[:]...) // 16
	if variant == 2 {
		b = append(b, 0, 0, 0, 0)
	} else {
		b = append(b, 0x65, 0x65, 0x00, 0x72) // S,B0,B1,B2 (image-статики)
	}
	if variant == 1 {
		b = append(b, 0x23)
	} else {
		b = append(b, 0x00)
	}
	if variant == 9 {
		return WriteFrame(EncryptPrimaryXor(bf, b)) // 172 → 184 → wire 186
	}
	b = append(b, make([]byte, welcomeExtra4)...) // +4 → 177
	if variant == 5 {
		return WriteFrame(EncryptPrimaryXor(bf, b))
	}
	return WriteFrame(EncryptPrimary(bf, b))
}

// EncryptPrimaryXor — как EncryptPrimary, но чек-сумма = XOR дворов вместо cumsum
// (проба альтернативной модели чек-суммы).
func EncryptPrimaryXor(bf *Blowfish, data []byte) []byte {
	n := (len(data) + 7) &^ 7
	buf := make([]byte, n+8)
	copy(buf, data)
	var x uint32
	for k := 0; k < n/4; k++ {
		x ^= binary.LittleEndian.Uint32(buf[k*4:])
	}
	binary.LittleEndian.PutUint32(buf[n:], x)
	out := make([]byte, n+8)
	for off := 0; off < n+8; off += 8 {
		bf.Encrypt(out[off:off+8], buf[off:off+8])
	}
	return out
}