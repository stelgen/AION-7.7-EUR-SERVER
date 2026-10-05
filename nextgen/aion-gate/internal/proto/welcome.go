package proto

// Welcome (2106): 194B = [u16 LE len=0xC2][ECB(key1) 192B].
// plaintext = Assemble("cddbbbcccc") 173B (+4 — §5.1, верифицировать capture'ом)
// → round-up 184 → +8 чексумма → ECB 192 (док §1/§2).

const (
	WelcomeLen      = 194
	WelcomePlainLen = 173
	// welcomeExtra4 — происхождение 4 доп. байт plaintext не вскрыто (§5.1);
	// модель размера сходится только с ними (173+4 → 184 → +8 → 192+2=194).
	welcomeExtra4 = 4
)

// WelcomeArgs — wargs вызова Assemble("cddbbbcccc") (§2).
type WelcomeArgs struct {
	PlainByte    byte      // plaintext[0]: capture 03.10 = 0x23 (§5.1)
	SessionID    uint32    // m_iSessionId
	AuthdSession uint32    // [authd_sock+0xa0]
	Modulus      [128]byte // RAW modulus (скрамблится внутри)
	GGQuery      [16]byte  // нули при GG-off
	Key2         [16]byte
	LoginType    byte
	B0, B1, B2   byte
}

func (a *WelcomeArgs) plain() []byte {
	mod := a.Modulus
	ScrambleModulus(&mod)
	return Assemble("cddbbbcccc",
		a.PlainByte, a.SessionID, a.AuthdSession,
		mod[:], a.GGQuery[:], a.Key2[:],
		a.LoginType, a.B0, a.B1, a.B2)
}

// BuildWelcome собирает welcome-пакет целиком (194B), шифрованный key1.
func BuildWelcome(a *WelcomeArgs, bf *Blowfish) []byte {
	plain := append(a.plain(), make([]byte, welcomeExtra4)...)
	return WriteFrame(EncryptPrimary(bf, plain))
}
