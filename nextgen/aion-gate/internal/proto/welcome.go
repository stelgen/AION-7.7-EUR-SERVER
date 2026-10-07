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
	// П1 байон-48: в welcome идёт СЕРВЕРНЫЙ скрамбл (клиент снимает его своим unscramble).
	ScrambleModulusServer(&mod)
	b := make([]byte, 0, 173)
	b = append(b, 0x00)                                     // [0] = opcode SM_INIT (0x00, "c" va0)
	b = binary.LittleEndian.AppendUint32(b, a.SessionID)    // [1:5] = sid (LE)
	b = binary.LittleEndian.AppendUint32(b, a.AuthdSession) // [5:9] = V (authd [03])
	b = append(b, mod[:]...)                                // 128
	b = append(b, a.GGQuery[:]...)                          // 16
	b = append(b, a.Key2[:]...)                             // 16
	b = append(b, 0x65, 0x65, 0x00, 0x72)                   // S,B0,B1,B2 (image-статики)
	b = append(b, 0x00)                                     // 'c'-байт va[0]=0x0
	return b                                                // 173
}

// BuildWelcome собирает welcome-пакет целиком (194B), шифрованный key1.
func BuildWelcome(a *WelcomeArgs, bf *Blowfish) []byte {
	plain := append(a.plain(), make([]byte, welcomeExtra4)...) // 173+4=177
	return WriteFrame(EncryptPrimary(bf, plain))
}

// Примечание (07.10 релиз): эпоха пробных вариантов welcome (1–9: PlainByte-хвост,
// нули-статики, GG=FF, raw-модуль, XOR-csum, фиксированные fc/V, без +4) ЗАВЕРШЕНА —
// variant 0 (BuildWelcome, серверный скрамбл) live-принят клиентом; machinery удалена
// (история — в git и docs/fork-classic-deploy-20261006.md).
