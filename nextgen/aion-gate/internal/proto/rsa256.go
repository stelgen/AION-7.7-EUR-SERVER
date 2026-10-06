package proto

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"math/big"
	"sync"
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

// GenerateRSAKey — 1024-битный ключ с экспонентой e (конфиг rsaExponent).
// Гипотеза «клиент шифрует m^17»: e=17 (Pub.key[1]=0x11). Гит beyond-aion 4.8:
// e = RSAKeyGenParameterSpec.F4 = 65537. Оракул — живой логин с валидной раскладкой
// user@94/pwd@108/otp=FFFFFFFF; калибровка через fork-режим (ct + N_orig + известные креды).
// rsa.GenerateKey жёстко 65537 — генерим p/q вручную, d = e⁻¹ mod φ(n), gcd(e,φ)=1.
func GenerateRSAKey(exponent int64) (*RSAKey, error) {
	if exponent < 3 || exponent%2 == 0 {
		return nil, errors.New("proto: rsa exponent must be odd and >= 3")
	}
	eBig := big.NewInt(exponent)
	one := big.NewInt(1)
	for i := 0; i < 64; i++ {
		p, err := rand.Prime(rand.Reader, 512)
		if err != nil {
			return nil, err
		}
		q, err := rand.Prime(rand.Reader, 512)
		if err != nil {
			return nil, err
		}
		if p.Cmp(q) == 0 {
			continue
		}
		p1 := new(big.Int).Sub(p, one)
		q1 := new(big.Int).Sub(q, one)
		phi := new(big.Int).Mul(p1, q1)
		if new(big.Int).GCD(nil, nil, eBig, phi).Cmp(one) != 0 {
			continue // gcd(e, φ) ≠ 1 — пробуем другие простые
		}
		n := new(big.Int).Mul(p, q)
		d := new(big.Int).ModInverse(eBig, phi)
		priv := &rsa.PrivateKey{
			PublicKey: rsa.PublicKey{N: n, E: int(exponent)},
			D:         d,
			Primes:    []*big.Int{p, q},
		}
		priv.Precompute()
		return &RSAKey{Priv: priv}, nil
	}
	return nil, errors.New("proto: rsa keygen failed after 64 tries")
}

// Modulus128 — модуль как 128-байтный буфер (BE; 1024-бит модуль занимает
// ровно 128Б — старший бит установлен, ведущих нулей нет).
func (k *RSAKey) Modulus128() [128]byte {
	var out [128]byte
	m := k.Priv.PublicKey.N.Bytes()
	copy(out[128-len(m):], m)
	return out
}

// DecryptBlock — приватная операция (аналог beecrypt rsapricrt @0x417b60):
// m = ct^d mod n, всегда ПОЛНЫЙ 128Б BE (П3 байон-48: раскладка читается из полного
// m; decbuf для authd отрезается по конфигу loginDecbufLen — asm оригинала arg3=0x22=34).
// Вход — ровно 128Б (RSA-блок логина против модуля из welcome).
func (k *RSAKey) DecryptBlock(ct []byte) ([]byte, error) {
	if len(ct) != 128 {
		return nil, ErrBadBlockLen
	}
	c := new(big.Int).SetBytes(ct)
	m := new(big.Int).Exp(c, k.Priv.D, k.Priv.PublicKey.N)
	return m.FillBytes(make([]byte, 128)), nil
}

// ScrambleModulus — КЛИЕНТСКИЙ unscramble (то, что делает клиент с модулем из welcome):
// xor-upper → dword 0x0d^0x34 → xor-lower → swap.
// ⚠ ИСТОРИЯ: до 06.10 (commit 3b78928) она слалась клиенту как «серверный скрамбл» —
// клиент анскрамблил ЕЩЁ РАЗ → N_client ≠ N_our → decbuf = мусор (root-cause найден
// по гиту beyond-aion 4.8, см. docs/beyond-aion-48-protocol-vs-gate-20261006.md §3).
// Теперь используется ТОЛЬКО для анскрамбла чужих (оригинальных) модулей и тестов.
func ScrambleModulus(m *[128]byte) {
	for i := 0x40; i < 0x80; i++ {
		m[i] ^= m[i-0x40]
	}
	for i := 0; i < 4; i++ {
		m[0x0d+i] ^= m[0x34+i]
	}
	for i := 0; i < 0x40; i++ {
		m[i] ^= m[0x40+i]
	}
	for i := 0; i < 4; i++ {
		m[i], m[0x4d+i] = m[0x4d+i], m[i]
	}
}

// ScrambleModulusServer — СЕРВЕРНЫЙ скрамбл, который обязан слать гейт
// (= EncryptedRSAKeyPair.encryptModulus из гита beyond-aion 4.8, порядок инверсный
// клиентскому unscramble): 1) swap m[0..4) ↔ m[0x4d..0x51); 2) m[i] ^= m[0x40+i]
// i<0x40 (lower ^= upper); 3) dword @0x0d ^= dword @0x34; 4) m[0x40+i] ^= m[i]
// i<0x40 (upper ^= lower, обновлённый). Клиент своим unscramble (ScrambleModulus)
// восстанавливает ровно N: UnscrambleClient(ServerScramble(N)) == N (инверс точный,
// верифицировано 500/500 в доке §3; обе функции НЕ инволюции — RAW/двойной скрамбл сломан).
func ScrambleModulusServer(m *[128]byte) {
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

// RSAKeyFromHex — фиксированная пара (N, D hex BE) для экспериментов Pub.key:
// клиент может шифровать логин против фиксированного ключа (не из welcome).
func RSAKeyFromHex(nHex, dHex string, exponent int64) (*RSAKey, error) {
	n, ok := new(big.Int).SetString(nHex, 16)
	if !ok || n.BitLen() < 1020 {
		return nil, errors.New("rsa256: bad N hex")
	}
	d, ok := new(big.Int).SetString(dHex, 16)
	if !ok {
		return nil, errors.New("rsa256: bad D hex")
	}
	return &RSAKey{Priv: &rsa.PrivateKey{PublicKey: rsa.PublicKey{N: n, E: int(exponent)}, D: d}}, nil
}

// KeyPool — пул 5 RSA-ключей, выдача по кругу (GetKey = counter++ % 5).
type KeyPool struct {
	mu      sync.Mutex
	keys    [RSAPoolSize]*RSAKey
	counter uint32
}

// NewKeyPoolFromKey — пул из одной фиксированной пары (эксперимент Pub.key).
func NewKeyPoolFromKey(k *RSAKey) *KeyPool {
	p := &KeyPool{}
	for i := range p.keys {
		p.keys[i] = k
	}
	return p
}
func NewKeyPool(exponent int64) (*KeyPool, error) {
	p := &KeyPool{}
	for i := range p.keys {
		k, err := GenerateRSAKey(exponent)
		if err != nil {
			return nil, err
		}
		p.keys[i] = k
	}
	return p, nil
}

func (p *KeyPool) Get() *RSAKey {
	p.mu.Lock()
	defer p.mu.Unlock()
	k := p.keys[p.counter%RSAPoolSize]
	p.counter++
	return k
}