# Always-on / автостарт сервисов AION VM

Схема после ребута VM (без батников для лёгких):
- ONSTART от SYSTEM (задачи пересозданы, /rl HIGHEST):
  AionAcc(acc.bat,45s) AionAuth(auth.bat,60s) AionGate(gate.bat,75s)
  AionLog(logsrv.bat,50s) AionICSrv(icserver.bat) AionCAPTCHA(captcha.bat)
  AionPA(pa.bat) AionProxy(proxy.bat)
- L2Authd и AuthGateD живут ТОЛЬКО в интерактивной сессии юзера — их задачи
  созданы с /IT (запуск при залогиненном пользователе). После включения
  автологона в Administrator сессия есть всегда.
- Тяжёлые (CacheD->NPCSvr->Server64) поднимает AION-START-SERVER.bat
  (или руками задачи AionCache/AionNPC/AionMain).
- fix-autonomy.bat: DisableCAD, ShutdownReason off, заставка off, power off,
  event log caps (App/Sys 50MB, Sec 256MB, overwrite), чистка мусорных задач.
- Мусорные задачи удалены/disabled: AionRAD, AionMain2, AionSRV, AionPA2,
  AionPA2105, AionFixLight/Kill, AionIntStart, AionGateTest, AionNetMon.
