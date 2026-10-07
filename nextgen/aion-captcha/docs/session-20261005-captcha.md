# 🧵 Сессия 05.10.2026 — замена CAPTCHAImageServer → aion-captcha (Трек B, шаг 2)

> Прод-замена ВЫПОЛНЕНА И ВЕРИФИЦИРОВАНА. Коммиты: `3bfe87d` (recon) → `297b716` (протокол) →
> `179a808` (код) → см. git log. Статус-снимок: ../SNAPSHOT.md.

## Хронология

1. **Разведка read-only** (док ./captcha-recon-20261005.md): единственный клиент :22206 = Server64
   (постоянная сессия); authd/gate чисты; XML-генератор капчи раскрыт (6 цифр, dds/dxt1 128×32);
   Server64 ретраит и переподключается сам — замена безопасна; конфиг капчи в common.xml L37-40
   (useCaptcha=true, buffer 10000, addr 127.0.0.1:22206); aion-op задача AionCAPTCHA.
2. **Live capture по «го»** (окно 13:02–13:13): копия капчи на :22207 (config.ini байтово) +
   mirror-прокси capmirror.exe (Go, C2S/S2C hexdump). Server64 переподключился сам. Снято:
   101×1 + 1001×10000 → 102×1 + 1002×10000 (0 bad). Реставрация: /run AionCAPTCHA → оригинал PID 1940,
   Server64 reconnected. Дамп 77МБ (conn-1.hex) в песочницу + на VM.
3. **Протокол закрыт** (док ./captcha-protocol-20261005.md): фрейм [u16 len][u16 type], seq пер-тип;
   handshake 101/102; запрос 1001 (18Б); ответ 1002 (2212Б: DDS+DXT1+текст). Секрет: Server64 наливает
   буфер 10000 капч после каждого reconnect (26/с), потом тишина; при логине — из буфера. Валидация в
   Server64 по тексту из 1002. Rate оригинала 26/с (38мс/картинку GDI+).
4. **Реализация** aion-captcha (Go): proto/render(DXT1)/server/config/ship(из logd). Тесты
   зелёные на fixture. Локальный smoke: 10000 запросов за 4.14с = 2415/с (92× быстрее оригинала).
5. **Деплой-свитч по «го»**: D:\SAION\aion-captcha\ (exe MD5 5394aab1 + config.yaml verbose=true +
   run.cmd); оригинал убит через одноразовую SYSTEM-задачу taskkill (schtasks /end не убивает
   start-детачнутый процесс!); задача AionCAPTCHA пере-таргечена на run.cmd.
6. **Верификация**: 22206 LISTENING (PID 5572), Server64 ESTABLISHED за ~5с, полный налив 10000
   (captcha.log seq=1..9901 rate 1/100), тишина после; Server64.err — 0 captcha-ошибок после свитча.

## Готчи сессии

- schtasks /end AionCAPTCHA НЕ убивает CAPTCHAImageServer.exe (оригинал стартует `start`-ом из
  captcha.bat — процесс детачнут от задачи) → taskkill через одноразовую SYSTEM-задачу.
- python на VM НЕТ (where python пуст) — все парсеры только локально; type через cmd качает
  большие файлы нормально (77МБ).
- scp push в несуществующий каталог — fail («dest open: No such file»): сначала mkdir.
- Mirror-окно капчи безопасно: Server64 ретраит, live-прод не страдает; полный on-board налив буфера
  — идеальный capture-провокатор (не нужен юзер!).

## Финальное состояние прода (05.10 ~13:55)

- aion-captcha PID 5572 (:22206, задача AionCAPTCHA → D:\SAION\aion-captcha\run.cmd), Server64 7348 подключён
- Оригинал НЕ удалён (D:\AION_LIVE_SERVER\CAPTCHAImageServer\ нетронут), откат = 4 строки (см. SNAPSHOT)
- Трек B: logd ✅ → CAPTCHA ✅ → следующий = .NET-мелочь / AuthGateD

## Инструменты сессии

- ~/STELGEN/tmp/aion-vm/capmirror/main.go (+capmirror.exe) — mirror-прокси
- ~/STELGEN/tmp/aion-vm/cap-stage.ps1 / cap-start.ps1 / cap-restore.ps1 / cap-switch.ps1 / cap-s64check.ps1
- ~/STELGEN/tmp/aion-vm/fake_captcha_client.py — фейк-клиент (host port n)
- ~/STELGEN/tmp/aion-vm/captcha-capture/ — дампы (conn-1.hex 77МБ, dds-header.hex, samples)
- /tmp/cap-ascii.txt, /tmp/cap-u16.txt, /tmp/s64-u16.txt и пр. — строки бинарей
