# aion-op — оператор стека AION 7.7 (Phase 0: observe)

Единый Go-бинарь. **Phase 0 = только наблюдение**: read-only пробы, никакого управления.
План: [../TRACK-A-PLAN.md](../TRACK-A-PLAN.md), постановка: [../PLAN.md](../PLAN.md).

## Запуск

```bash
# демо без VM (mock-стек: всё зелёное)
go build -o aion-op . && ./aion-op -config config.yaml
# → http://127.0.0.1:10200

# демо «краснеет»: погасить сервисы
AIONOP_MOCK_DOWN=main,gate ./aion-op
AIONOP_MOCK_DOWN=main AIONOP_MOCK_CONNS=4 ./aion-op   # + окно загрузки NPC

# прод (read-only по SSH; кнопки всё равно выключены)
# vm.mode: ssh в config.yaml + доступ по ключу
./aion-op -config config.yaml
```

Пробы SSH (только чтение): `tasklist /fo csv /nh`, `netstat -ano -p tcp`, `quser`.

## Что уже по best-practice

- state machine (`RUNNING/LOADING/DEGRADED/STOPPED/UNKNOWN`), health = процесс + порт + conns-маркер;
- пара NPC+MAIN = единая единица: разрыв пары и окно загрузки (conns<16) детектируются и блокируют будущие рестарты;
- наблюдаемая консоль-сессия VM (quser) — без неё /IT-сервисы не поднять;
- режим `observe` жёстко в конфиге; управляющих HTTP-роутов НЕТ вообще (не «disabled», а отсутствуют);
- топология — единственный YAML-источник истины, секретов в нём нет.

## Структура

```
config.yaml            — топология (группы/сервисы/порты/order/пара)
main.go                — сборка
internal/config/       — YAML + валидация
internal/core/         — state machine (чистые функции, тесты)
internal/probe/        — Prober: mock | ssh (Phase 1: агент)
internal/web/          — API + embedded UI (вкладки, кнопки-замки)
```

## Дорожная карта (см. TRACK-A-PLAN.md)

- **0.5**: лог-тейлеры+парсер, метрики (RAM/handles/FreeCommit), SQLite-WAL, алерты.
- **1**: агент ~2 МБ в юзер-сессии VM (единственная инсталляция на прод, по «го») → режим operate.
- **1.5**: watchdog-автопилот (ночной рестарт пары, эскалации).
