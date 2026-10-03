# 🚀 Флаги запуска клиента AION — полный разбор

Каждый флаг: что делает, зачем нужен, что будет без него.
Рабочая строка (ru-клиент Innova 4game, проверено 03.10.2026):

```bat
@echo off
cd /d "%~dp0\bin64\Frost"
start Aion.exe -ip:81.25.59.194 -port:2106 -cc:7 -noauthgg -noweb -nobs -ls ^
  -charnamemenu -lbox -nologout -customizing -megaphone -nwp -ncping -f2p ^
  -localtime -rcdelay:5 -loginex -pwd16 -multithread ^
  -frostGame "..\..\bin64\aion.bin" -frostOptions 7 -frostGameNameType aion_live
```

## 🔑 Критические флаги (без них логин НЕ работает)

| Флаг | Что делает | Что будет БЕЗ него |
|---|---|---|
| **-loginex** | Включает **классический протокол LoginEx**: клиент шлёт логин+пароль напрямую в AuthGateD, эхоит sessionId из welcome-пакета | Клиент использует **портал-сессию** (sessionId=0) → гейт: `Session id mismatched` → RST → «вы были отключены» |
| **-pwd16** | Пароль передаётся как **MD5-хеш (16 байт)** — точно совпадает с БД `user_auth.password binary(16)` | Клиент шлёт пароль в другом формате → AuthD не свериет → логин отклонён |
| **-noauthgg** | Пропускает инициализацию GameGuard (nProtect) — на приватном сервере GG-инфраструктуры нет | GameGuard error (код 1xx), клиент не стартует |
| **-ip:<адрес>** | Адрес AuthGateD (логин-сервера) | Клиент пойдёт на официальный сервер |
| **-port:2106** | Порт AuthGateD | По умолчанию 2105 (нерабочий на нашем сервере!) |

## 🌐 Сетевые/регион флаги

| Флаг | Что делает |
|---|---|
| **-cc:7** | Код страны клиента: 0=KR, 2=EURO/F2P, 5=CN, 7=RUS. Должен совпадать с регионом сервера (Server.region) — иначе «региональный код не совместим» |
| **-localtime** | Использовать локальное время клиента для отображения серверного времени |
| **-rcdelay:5** | Reconnect delay — 5 секунд между попытками переподключения |
| **-ncping** | Network connectivity ping — клиент пингует auth для проверки живости |

## 🖥 UI/UX флаги (косметика, на логин не влияют)

| Флаг | Что делает |
|---|---|
| -megaphone | Иконка мегафона (объявления) |
| -webpetition | Веб-петиции (саппорт) |
| -charnamemenu | Меню имени персонажа на выборе |
| -lbox | Стиль логин-бокса |
| -nologout | Убрать кнопку Logout |
| -customizing | Разрешить кастомизацию персонажа |
| -f2p | Free-to-Play режим UI |
| -noweb | Отключить встроенный веб-браузер |
| -nobs | Отключить BlackScreen-заставку |
| -nwp | Отключить web-плаши (web popups) |
| -multithread | Многопоточный рендеринг |
| -ls | Login Server direct mode |

## ❄️ Только для Innova/4game клиента (RU)

| Флаг | Что делает |
|---|---|
| **-frostGame "..\..\bin64\aion.bin"** | Указывает Frost-обёртке (`bin64\Frost\Aion.exe`) какой бинарник запускать. Рабочая папка ОБЯЗАТЕЛЬНО = `bin64\Frost` (иначе относительный путь сломается → «Не удалось найти файл») |
| **-frostOptions 7** | Битмаска опций Frost |
| **-frostGameNameType aion_live** | Идентификатор игры для Frost |

⚠️ **Frost активируется ТОЛЬКО при запуске через `bin64\Frost\Aion.exe`**. Прямой запуск `bin64\aion.bin` обходит Frost полностью. На приватном сервере Frost не нужен — но и не мешает (проверено).

## 📝 Для euro-клиента (AionLauncher)

EU-клиент запускается через `AionLauncher.exe`, который читает **`launcher.config`** (одна строка):
```
-ip:81.25.59.194 -port:2106 -cc:2 -win10-mouse-fix -noweb -nowebshop -nokicks -ncg -noauthgg -ls -charnamemenu -ingameshop -loginex -pwd16
```
⚠️ В ките был порт 2105 (нерабочий!) — обязательно 2106. Флаги `-loginex -pwd16` — критичны.

## 📝 PowerShell нюанс
В PowerShell относительные пути требуют `.\` префикс: `.\bin64\Aion.bin` — иначе «module bin64 could not be loaded». Проще использовать .bat файл.
