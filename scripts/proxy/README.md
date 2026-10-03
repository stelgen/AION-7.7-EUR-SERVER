# 🔄 MITM-прокси перехватчик AuthGateD — логгер и анализатор трафика

Python-прокси между клиентом и AuthGateD: транслирует весь трафик байт-в-байт
и логирует каждый пакет в hex. Используется для:
1. Реверса протокола логина (поиск полей sessionId/пароля).
2. Отладки проблем клиентов (что дошло, что вернул сервер).
3. Будущего MITM-переписывания полей (после реверса Blowfish-ключа из welcome).

## Запуск на VM

```powershell
# python 3.12.8 embed уже установлен: C:\Temp\py\python.exe
schtasks /Run /TN AionProxy
# лог: C:\Temp\proxylog.txt
```

Гейт должен слушать **2107** (прокси занимает 2106):
```
AuthGateD\etc\config.txt → serverPort = 2107
```

## Схема

```
[Клиент] → 2106 (прокси) → 2107 (AuthGateD) → 2110 (L2Authd) → 2220 (AccCache) / 10057×2 (PA) / SQL
```

## Формат лога

```
HH:MM:SS --- new client ('IP', port)
HH:MM:SS G->C len=194 hex=c2007ad70ab5...
HH:MM:SS C->G len=34 hex=2200...
HH:MM:SS C->G closed
```

Фрейминг: первые 2 байта LE = длина пакета. Пакеты после handshake зашифрованы
(Blowfish-ECB, ключ из welcome/SM_INIT) — расшифровка требует извлечения ключа
(см. `docs/server-internals.md` раздел протокола).

## Перехваченные размеры пакетов (сессия логина ru-клиента)

| Направление | Размер | Что |
|---|---|---|
| G→C | 194 | welcome: session id + RSA-модуль (скремблированный) + Blowfish-ключ сессии |
| C→G | 34 | RSA-шифрованный ответ клиента (256-бит RSA — клиент шифрует ключ сессии публичником гейта) |
| G→C | 42 | подтверждение handshake (блочный шифр активирован) |
| C→G | 186 | логин-пакет (account + password-hash, зашифрован Blowfish) |
| G→C | 74 | auth-ответ (server list / session confirm) |
| C↔G | 26 | ping/keepalive |
