package proto

import (
	"crypto/rand"
	"encoding/binary"
)

// Classic-режим (mode: classic) — beyond-aion-совместимый флоу 4.8, раскладки из
// docs/beyond-aion-48-protocol-vs-gate-20261006.md §2/§5/§6 и исходников AC-Login 4.7.5
// (SM_INIT.java: opcode-first, rev 0x0000c621, mod 128B, gg 16B, bfkey 16B;
// SM_AUTH_GG.java: [0b][sid][4×0][19×0] = 40B pt → wire 50).

const (
	ClassicWelcomeWire = 210          // [u16 208][ECB 208]
	ClassicProtocolRev = 0x0000c621   // dword[5:9] SM_INIT
	ClassicTailMagic   = 0x3FCE09ED   // dword @184 SM_INIT
)

// EncryptGitInit — шифрование ПЕРВОГО пакета (CryptEngine.encrypt, updatedKey=false,
// сверено с исходником AC-Login 4.7.5 = семейство beyond-aion):
//   length = len(pt) + 4 + 4; length += 8 - length%8  ← JAVA-семантика: при кратности 8
//   добавляет ЕЩЁ 8 (для SM_INIT pt=192: 200 → 208 → ECB 208 → wire 210); при НЕ-кратности
//   (наш 7.7 welcome pt=177): 185 → 192 — совпадает с asm-моделью EncryptPrimary 1-в-1.
// encXORPass: ecx = РАНДОМ-СИД; chain dwords 1..(stop/4-1), stop = length-8:
//   ecx += edx; edx ^= ecx (dword0 не трогается); финальный ecx → [stop:stop+4];
//   [stop+4:length) остаются нулями («chk-нули»). Отличие от EncryptPrimary: сид Rnd,
//   а не dword[0] (у 7.7 AuthGateD сид = dword[0] — live-верифицировано дизasm'ом).
func EncryptGitInit(bf *Blowfish, pt []byte, rnd uint32) []byte {
	length := len(pt) + 8
	length += 8 - length%8 // Java: 8 - length%8, при %8==0 даёт +8
	buf := make([]byte, length)
	copy(buf, pt)
	stop := length - 8
	S := rnd
	for pos := 4; pos < stop; pos += 4 {
		d := binary.LittleEndian.Uint32(buf[pos:])
		S += d
		binary.LittleEndian.PutUint32(buf[pos:], d^S)
	}
	binary.LittleEndian.PutUint32(buf[stop:], S) // финальный ecx
	// [stop+4:length) — chk-нули (buf создан нулевым)
	out := make([]byte, length)
	for off := 0; off < length; off += 8 {
		bf.Encrypt(out[off:off+8], buf[off:off+8])
	}
	return out
}

// DecryptGitInit — клиентская сторона первого пакета (реверс encXORPass от stored ecx).
func DecryptGitInit(bf *Blowfish, data []byte) ([]byte, error) {
	if len(data) == 0 || len(data)%8 != 0 {
		return nil, ErrBadPacketLen
	}
	dec := make([]byte, len(data))
	for off := 0; off < len(data); off += 8 {
		bf.Decrypt(dec[off:off+8], data[off:off+8])
	}
	n := len(dec) - 8
	S := binary.LittleEndian.Uint32(dec[n:])
	for k := n/4 - 1; k >= 1; k-- {
		old := binary.LittleEndian.Uint32(dec[k*4:]) ^ S
		binary.LittleEndian.PutUint32(dec[k*4:], old)
		S -= old
	}
	return dec[:n], nil
}

// BuildClassicWelcome — SM_INIT (op 0x00) plaintext 192B по §2:
// [0]=00, [1:5]=sid, [5:9]=0x0000c621, [9:137]=ScrambleModulusServer(N) (скрамбл ВНУТРИ,
// вход = RAW модуль — как WelcomeArgs), [137:153]=16×0, [153:169]=sessionKey16,
// [169:176]=7×0, [176]=0, [177:181]=0, [181:183]=0, [183]=0, [184:188]=0x3FCE09ED,
// [188:192]=0 → EncryptGitInit(static) → ECB 208 → wire 210.
func BuildClassicWelcome(sid uint32, modRaw [128]byte, sessionKey [16]byte, staticBF *Blowfish) []byte {
	mod := modRaw
	ScrambleModulusServer(&mod)
	pt := make([]byte, 0, 192)
	pt = append(pt, 0x00)
	pt = binary.LittleEndian.AppendUint32(pt, sid)
	pt = binary.LittleEndian.AppendUint32(pt, ClassicProtocolRev)
	pt = append(pt, mod[:]...)
	pt = append(pt, make([]byte, 16)...) // gg-нули
	pt = append(pt, sessionKey[:]...)
	pt = append(pt, make([]byte, 7)...)
	pt = append(pt, 0x00)
	pt = binary.LittleEndian.AppendUint32(pt, 0)
	pt = binary.LittleEndian.AppendUint16(pt, 0)
	pt = append(pt, 0x00)
	pt = binary.LittleEndian.AppendUint32(pt, ClassicTailMagic)
	pt = binary.LittleEndian.AppendUint32(pt, 0)
	var r [4]byte
	_, _ = rand.Read(r[:])
	return WriteFrame(EncryptGitInit(staticBF, pt, binary.LittleEndian.Uint32(r[:])))
}

// BuildClassicAuthGG — SM_AUTH_GG (0x0b):
//   gitForm=false → живая форма 7.7: [0b][sid][27×0] (32B pt, wire 42);
//   gitForm=true  → форма гита (SM_AUTH_GG.java): [0b][sid][0×4][0xCD5000][0]
//                   [0x0b<<24][sid^0xCD5000][3×0] → 37B pt → ECB 48 → wire 50.
func BuildClassicAuthGG(sid uint32, gitForm bool) []byte {
	pt := []byte{0x0b}
	pt = binary.LittleEndian.AppendUint32(pt, sid)
	if !gitForm {
		pt = append(pt, make([]byte, 27)...)
		return pt
	}
	pt = append(pt, 0, 0, 0, 0)
	pt = binary.LittleEndian.AppendUint32(pt, 0x00CD5000)
	pt = binary.LittleEndian.AppendUint32(pt, 0)
	pt = binary.LittleEndian.AppendUint32(pt, 0x0B000000)
	pt = binary.LittleEndian.AppendUint32(pt, sid^0xCD5000)
	pt = append(pt, make([]byte, 12)...)
	return pt
}

// BuildClassicLoginOK — SM_LOGIN_OK (0x03): [accountId][loginOk][0][0][0x000003ea][7×0][19×0].
func BuildClassicLoginOK(accountID, loginOk uint32) []byte {
	pt := []byte{0x03}
	pt = binary.LittleEndian.AppendUint32(pt, accountID)
	pt = binary.LittleEndian.AppendUint32(pt, loginOk)
	pt = binary.LittleEndian.AppendUint32(pt, 0)
	pt = binary.LittleEndian.AppendUint32(pt, 0)
	pt = binary.LittleEndian.AppendUint32(pt, 0x000003ea)
	pt = append(pt, make([]byte, 7)...)
	pt = append(pt, make([]byte, 19)...)
	return pt
}

// BuildClassicLoginFail — LOGIN_FAIL (0x01): [D responseId], шифруется key2 вызывающим.
func BuildClassicLoginFail(responseID uint32) []byte {
	pt := []byte{0x01}
	return binary.LittleEndian.AppendUint32(pt, responseID)
}

// BuildClassicServerList — SM_SERVER_LIST (0x04) полный формат §6:
// C(count) C(lastServer), на сервер [C id][ip4][H port][H0][C age][C pvp][H cur][H max]
// [C online][C type][C hide][H0][C brackets], затем H(maxId+1), C(1 автолинк),
// C(charCount на сервер), 13×0.
func BuildClassicServerList(ip [4]byte, port uint16, lastServer byte) []byte {
	pt := []byte{0x04, 0x01, lastServer}
	pt = append(pt, 0x01) // id сервера
	pt = append(pt, ip[:]...)
	pt = binary.LittleEndian.AppendUint16(pt, port)
	pt = binary.LittleEndian.AppendUint16(pt, 0)
	pt = append(pt, 0) // age
	pt = append(pt, 0) // pvp
	pt = binary.LittleEndian.AppendUint16(pt, 1)
	pt = binary.LittleEndian.AppendUint16(pt, 1000)
	pt = append(pt, 1) // online
	pt = append(pt, 0) // type
	pt = append(pt, 0) // hide
	pt = binary.LittleEndian.AppendUint16(pt, 0)
	pt = append(pt, 0) // brackets
	pt = binary.LittleEndian.AppendUint16(pt, 2) // maxIdWithChars+1
	pt = append(pt, 0x01)                        // автолинк
	pt = append(pt, 0x00)                        // charCount сервера 1
	pt = append(pt, make([]byte, 13)...)
	return pt
}

// BuildClassicPlayOK — SM_PLAY_OK (0x07): [playOk1][playOk2][servId C][14×0].
func BuildClassicPlayOK(playOk1, playOk2 uint32, servID byte) []byte {
	pt := []byte{0x07}
	pt = binary.LittleEndian.AppendUint32(pt, playOk1)
	pt = binary.LittleEndian.AppendUint32(pt, playOk2)
	pt = append(pt, servID)
	pt = append(pt, make([]byte, 14)...)
	return pt
}