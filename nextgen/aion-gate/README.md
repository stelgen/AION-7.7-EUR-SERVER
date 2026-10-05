# aion-gate — перепись AuthGateD (гейт 2106 → authd 2110)

Go-замена оригинального AuthGateD.exe (AION 7.7 PTS EU). Референс:
`docs/authgate-protocol-20261005.md` (криптосхема верифицирована capture 03.10,
wire 2110 дизasm 05.10 байт-в-байт). Образец структуры: `../aion-captcha`.

## Статус (05.10 ночь, скелет)

- ✅ `internal/proto` — Blowfish ECB (0 зависимостей, P/S из exe, векторы python-blowfish),
  EncryptPrimary/DecryptSecondary (скрамбл/чексуммы), Assemble (c/h/d/b/s/S),
  LUT-ключи (key1 static ✓ 6b60cb5b…, key2), welcome-билдер (194B, plaintext[0]=0x23),
  RSA-256 (пул 5 ключей, ScrambleModulus @0x417c50 1-в-1 с дизasmом), фрейминг 2b LE.
- ✅ `internal/authdclient` — wire 2110 1-в-1: [00][sid][IP] / [01][sid] /
  [02][sid][len=blob+2][blob]; inbound state-машина [01]/[02]/[03], type<0x15.
  Реконнект после потери authd — решение внешнего кода (оригинал НЕ реконнектит).
- ⬜ next: `internal/server` (сессии/brute/BlockIPs/cc), `internal/config` (yaml-зеркало
  config.txt), `internal/ship` (копия из aion-logd), main.go, фейк-клиент/фейк-authd.

## Тесты

```bash
go test ./...   # все зелёные; welcome-паритет: plaintext[0]=0x23, dword0=23 14 52 7d (capture)
```

## TODO §5 (добить при тестах)

- welcomeExtra4 (+4 байта plaintext) и выравнивание modulus128 — верифицировать capture'ом.
- §5.1 пуш-порядок 0x407d50; §5.2 type-байт клиента; §5.5 42-ответ ctx; §5.6 sid-генератор.
