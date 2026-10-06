package proto

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"math/big"
)

// RSA-1024 (оригинал: beecrypt rsakpMake/rsapricrt/i2osp; пул 5 ключей ×0x4c,
// GetKey = counter++ % 5). LIVE-ФАКТ 06.10: RSA-256 давал 96 нулей в модуле →
// скрамбл-константы в welcome (повторяющиеся ECB-блоки) → клиент молча зависал;
// capture-модуль случайный на все 128Б ⇒ оригинал = RSA-1024.

const RSAPoolSize = 5

var ErrBadBlockLen = errors.New("proto: rsa block must be 128 bytes")
var ErrPlainTooBig = errors.New("proto: rsa plaintext > 32 bytes (i2osp 0x20)")

type RSAKey struct {
	Priv *rsa.PrivateKey
}

// GenerateRSAKey — 1024-битный ключ (модуль = ровно 128Б BE).
func GenerateRSAKey() (*RSAKey, error) {
	k, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		return nil, err
	}
	return &RSAKey{Priv: k}, nil
}

// Modulus128 — модуль как 128-байтный буфер (BE; 1024-бит модуль занимает
// ровно 128Б — старший бит установлен, ведущих нулей нет).
func (k *RSAKey) Modulus128() [128]byte {
	var out [128]byte
	m := k.Priv.PublicKey.N.Bytes()
	copy(out[128-len(m):], m)
	return out
}

// DecryptBlock — приватная операция (аналог beecrypt rsapricrt @0x417b60:
// mpnsetbin(in,0x80) → rsapricrt → i2osp(out,0x20)): m = ct^d mod n,
// результат BE-число; asm возвращает decbuf = i2osp(len 0x20) ⇒ ≤32Б.
// Вход — ровно 128Б (RSA-блок логина против модуля из welcome).
func (k *RSAKey) DecryptBlock(ct []byte) ([]byte, error) {
	if len(ct) != 128 {
		return nil, ErrBadBlockLen
	}
	c := new(big.Int).SetBytes(ct)
	m := new(big.Int).Exp(c, k.Priv.D, k.Priv.PublicKey.N)
	if m.BitLen() > 256 {
		return nil, ErrPlainTooBig
	}
	return m.FillBytes(make([]byte, 32)), nil
}

// ScrambleModulus @0x417c50 (дизasm 05.10):
// 1) swap байтов 0..3 ↔ 0x4d..0x50; 2) m[0..63] ^= m[0x40..0x7f];
// 3) dword @0x0d ^= dword @0x34; 4) m[0x40..0x7f] ^= m[0..63] (обновлённые).
// keyIdx вне [0,128] → нулевой буфер (ошибка пула).
func ScrambleModulus(m *[128]byte) {
	for i := 0; i < 4; i++ {
		m[i], m[0x4d+i] = m[0x4d+i], m[i]
	}
	for i := 0; i < 0x40; i++ {
		m[i] ^= m[0x40+i]
	}
	for i := 0; i < 4; i++ {
		m[0x0d+i] ^= m[0x34+i]
	}
	for i := 0x40; i < 0x80; i++ {
		m[i] ^= m[i-0x40]
	}
}

// KeyPool — пул 5 RSA-ключей, выдача по кругу (GetKey = counter++ % 5).
type KeyPool struct {
	keys    [RSAPoolSize]*RSAKey
	counter uint32
}

func NewKeyPool() (*KeyPool, error) {
	p := &KeyPool{}
	for i := range p.keys {
		k, err := GenerateRSAKey()
		if err != nil {
			return nil, err
		}
		p.keys[i] = k
	}
	return p, nil
}

func (p *KeyPool) Get() *RSAKey {
	k := p.keys[p.counter%RSAPoolSize]
	p.counter++
	return k
}