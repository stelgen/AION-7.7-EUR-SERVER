package proto

// Welcome (2106): 194B = [u16 LE len=0xC2][ECB(key1) 192B].
// Дизasm 0x407d50 (ночь-4, docs/session-20261005-authgate-disasm407d50.md):
// fmt "cddbbbcccc" @0x42cf2c; plaintext 173B =
//	[0]      = 0x00            ('c' ← va[0] = 0x0 — НЕ PlainByte!)
//	[1:5]    = [esi+0xfc]      (d)
//	[5:9]    = V=[ds:0x43b438→obj+0xa0] (d)
//	[9:137]  = scrambleModulus 128B (b: len 0x80)
//	[137:153]= GG-блоб 16B (нули при флаге [0x43c188]==0) (b: len 0x10)
//	[153:169]= key2 16B ([esi+0x21a4]) — клиент узнаёт key2 ИЗ welcome (b: len 0x10)
//	[169:173]= S=[0x43c1ac], B0, B1, B2=[0x43c1b0..2] (cccc)
// welcomeExtra4 = add eax,0x4 к длине форматтера (@0x407e16) — удлинение, НЕ поле:
//	n=177 → EncryptPrimary: roundup8=184 → csum(финальный S)@dword46(184) →
//	ECB 192 → wire 194. Байты 173..183 plaintext = нули буфера.

const (
	WelcomeLen      = 194
	WelcomePlainLen = 173
	// welcomeExtra4 — удлинение длины на 4 (asm 0x407e16: add eax,0x4),
	// содержимое = нули (хвост буфера за выводом форматтера).
	welcomeExtra4 = 4
)

// WelcomeArgs — wargs вызова Assemble("cddbbbcccc") (asm 0x407d50, ночь-4).
type WelcomeArgs struct {
	SessionID    uint32    // [esi+0xfc] (d)
	AuthdSession uint32    // V=[ds:0x43b438→obj+0xa0] (d)
	Modulus      [128]byte // RAW modulus (скрамблится внутри; b len=0x80)
	GGQuery      [16]byte  // нули при GG-off ([0x43c188]==0) (b len=0x10)
	Key2         [16]byte  // [esi+0x21a4] — key2 для клиента (b len=0x10)
	LoginType    byte      // S=[0x43c1ac] (c)
	B0, B1, B2   byte      // [0x43c1b0..2] (ccc)
}

func (a *WelcomeArgs) plain() []byte {
	mod := a.Modulus
	ScrambleModulus(&mod)
	return Assemble("cddbbbcccc",
		byte(0), a.SessionID, a.AuthdSession,
		mod[:], a.GGQuery[:], a.Key2[:],
		a.LoginType, a.B0, a.B1, a.B2)
}

// BuildWelcome собирает welcome-пакет целиком (194B), шифрованный key1.
func BuildWelcome(a *WelcomeArgs, bf *Blowfish) []byte {
	plain := append(a.plain(), make([]byte, welcomeExtra4)...) // 173+4=177
	return WriteFrame(EncryptPrimary(bf, plain))
}