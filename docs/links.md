# 🔗 Легальные ссылки на всё, что нужно скачать

Ничего ворованного в этом репо нет. Всё, что нужно скачать из интернета, — либо официальные загрузки Microsoft, либо public-архивы (archive.org), либо community-источники (RaGEZONE).

## 💾 Операционная система VM

| Что | Где | Заметки |
|---|---|---|
| Windows Server 2022 ISO (EN eval) | [archive.org/windows-server-2022-evaluation](https://archive.org/download/windows-server-2022-evaluation/SERVER_EVAL_x64FRE_en-us.iso) | 180-дневная eval, официально от MS (сборка 20348.1787) |
| Windows Server 2022 ISO (RU retail Mar2024) | у тебя на NAS (искать `ru-ru_windows_server_2022_updated_march_2024_x64_dvd_f6700d18.iso`) | полноценный retail + CU |
| SQL Server 2022 Developer ISO (EN) | [archive.org/sqlserver-2022-x-64-eng](https://archive.org/download/sqlserver-2022-x-64-eng/SQLServer2022-x64-ENG.iso) | официальный MSDN-образ (1.44 ГБ) |
| SQL Server 2022 eval SSEI | [Microsoft Eval Center](https://www.microsoft.com/en-us/evalcenter/download-sql-server-2022) | крошечный `SQLServer2022-SSEI-Eval.exe`, сам скачает ISO |

## 🧩 Рантаймы Microsoft (все обязательны)

| Что | Прямая ссылка | Размер |
|---|---|---|
| VC++ 2010 SP1 x64 | `https://download.microsoft.com/download/1/6/5/165255E7-1014-4D0A-B094-B6A430A6BFFC/vcredist_x64.exe` | 10.3 МБ |
| VC++ 2010 SP1 x86 | `https://download.microsoft.com/download/1/6/5/165255E7-1014-4D0A-B094-B6A430A6BFFC/vcredist_x86.exe` | 9 МБ |
| VC++ 2012 U4 x64 | `https://download.microsoft.com/download/1/6/B/16B06F60-3B20-4FF2-B699-5E9B7962F9AE/VSU_4/vcredist_x64.exe` | 7.2 МБ |
| VC++ 2012 U4 x86 | `https://download.microsoft.com/download/1/6/B/16B06F60-3B20-4FF2-B699-5E9B7962F9AE/VSU_4/vcredist_x86.exe` | 6.6 МБ |
| VC++ 2013 x64 | `https://download.microsoft.com/download/5/D/8/5D8C65CB-C849-4025-8E95-C3966CAFD8AE/vcredist_x64.exe` | 5.2 МБ |
| VC++ 2013 x86 | `https://download.microsoft.com/download/5/D/8/5D8C65CB-C849-4025-8E95-C3966CAFD8AE/vcredist_x86.exe` | 4.5 МБ |
| VC++ 2015–2022 x64 | `https://aka.ms/vs/17/release/vc_redist.x64.exe` | 24.4 МБ |
| VC++ 2015–2022 x86 | `https://aka.ms/vs/17/release/vc_redist.x86.exe` | 13.3 МБ |
| **SQL Server Native Client 11.0** (QFE, 2024) | `https://download.microsoft.com/download/b/e/d/bed73aac-3c8a-43f5-af4f-eb4fea6c8f3a/ENU/x64/sqlncli.msi` | 4.8 МБ |
| SQL Server Native Client 11.0 (x86) | `https://download.microsoft.com/download/b/e/d/bed73aac-3c8a-43f5-af4f-eb4fea6c8f3a/ENU/x86/sqlncli.msi` | 3.0 МБ |
| SSMS 21 | `https://aka.ms/ssmsfullsetup` | ~500 МБ (опционально, для GUI-администрирования SQL) |
| 7-Zip | `https://www.7-zip.org/a/7z2501-x64.exe` | 1.6 МБ |

⚠️ **SNAC11 обязателен**: все DSN-файлы в сборке написаны под драйвер `SQL Server Native Client 11.0`. Без него любой коннект падает `HY000/556 «недопустимый файл DSN ""»`.

⚠️ При установке msiexec требует параметр `IACCEPTSQLNCLILICENSETERMS=YES` (с регистрацией букв!) — иначе exit 1603.

## 🔧 Драйверы VirtIO (Proxmox → Windows VM)

| Что | Где |
|---|---|
| virtio-win-0.1.302.iso | [GitHub qemus/virtiso-x86](https://github.com/qemus/virtiso-x86/releases/download/v0.1.302-1/virtio-win-0.1.302.iso) (877 МБ) — универсальный (x64+x86), включает `virtio-win-guest-tools.exe` (qemu-ga + spice-vdagent + все драйверы) |
| Свежий virtio-win | [fedorapeople stable](https://fedorapeople.org/groups/virt/virtio-win/direct-downloads/stable-virtio/virtio-win.iso) |

## 🎮 Клиент AION 7.7 (для твоего десктопа)

| Что | Где |
|---|---|
| `euro_aion 7.7.zip` (34.2 ГБ) | у тебя на NAS (`/media/NAS/TG/DVM Software File Centre/Aion/`) |
| Полный гайд клиента | RaGEZONE (тред ниже) |

## 📚 RaGEZONE-гайды (community)

| Гайд | Ссылка |
|---|---|
| **AION7.7pts (Europe) set up tutorial** (dAIdoom, март 2023) — основной | [forum.ragezone.com/threads/1211744](https://forum.ragezone.com/threads/aion7-7pts-europe-set-up-tutorial-and-server-download-address.1211744/) |
| AION 4.6 retail server file (Re-post) — основа туториала | [forum.ragezone.com/threads/1197401](https://forum.ragezone.com/threads/aion-4-6-retail-server-file-re-post.1197401/) |
| Раздел Aion Releases | [forum.ragezone.com/community/aion-releases.587](https://forum.ragezone.com/community/aion-releases.587/) |

## 📦 Файлы сервера (вне этого репо)

`AION7.7SERVER(eu).rar` (4.2 ГБ) — лежит на твоём NAS (`/media/NAS/TG/DVM Software File Centre/Aion/`). В сеть выкладывать не стоит (права NC Soft); внутри репо — только инструкции и скрипты для его установки.

Публичные ссылки на серверные файлы в тредах RaGEZONE периодически умирают (Baidu/Mega); если не нашёл — есть альтернативы:
- Google Drive ссылка в том же треде [1211744](https://forum.ragezone.com/threads/aion7-7pts-europe-set-up-tutorial-and-server-download-address.1211744/) (пост #10).
- Более свежая сборка «fixed» обсуждается в [постах 120-130](https://forum.ragezone.com/threads/aion7-7pts-europe-set-up-tutorial-and-server-download-address.1211744/page-7/) (та же автор).
