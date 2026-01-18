# wfpsim_discord_archiver

Архиватор ссылок вида `https://wfpsim.com/sh/<uuid>` из Discord-каналов.

- **Источник**: Discord сообщения (через обычного Discord Bot, не user token).
- **Данные**: дергает `https://wfpsim.com/api/share/<uuid>` и сохраняет нормализованные поля в Google Sheets.
- **Инкрементальность**: при первом запуске читает сообщения за последние `sinceDays` (по умолчанию 30), далее — с последнего обработанного messageId (state в `work/`).
- **Сортировка**: записи упорядочиваются по `TeamCharacters` (asc), затем по `TeamDpsMean` (desc). В режиме Apps Script сортирует сам скрипт.

## Требования

- Добавь бота на сервер и выдай ему доступ на чтение нужных каналов.
- Включи Intents в Discord Developer Portal (как минимум Message Content Intent, если хочешь читать полный текст сообщений).
- Подготовь Google Sheet.
- Рекомендуемый способ записи: Apps Script Web App (код лежит в `appscript/`).

## Конфиг и секреты

Секреты и локальные настройки должны лежать в `input/`:

- пример: `input/wfpsim_discord_archiver/examples/config.yaml`
- реальный конфиг: `input/wfpsim_discord_archiver/config.yaml` (в git не попадёт)

## Пайплайн (Windows / PowerShell)

### 1 Сборка приложения

```scripts/wfpsim_discord_archiver/bootstrap.ps1```

Скрипт:

- создаёт `input/wfpsim_discord_archiver/config.yaml` из `input/wfpsim_discord_archiver/examples/config.yaml`, если файла ещё нет
- скачивает Go-пакеты только для приложения
- собирает `wfpsim_discord_archiver.exe` в `apps/wfpsim_discord_archiver/`

### 2 Настройка конфига

Отредактируй `input/wfpsim_discord_archiver/config.yaml`:

- Discord: `discord.token`, `discord.serverIds`, `discord.channelIds`
- Apps Script: `appsScript.webAppUrl`, `appsScript.apiKey`
- Google Sheet: `sheet.id`, `sheet.name`

### 3 Запуск

Запуск из корня репозитория (рекомендуется):

```powershell
./apps/wfpsim_discord_archiver/wfpsim_discord_archiver.exe
```

Если запускаешь из папки приложения:

```powershell
cd apps/wfpsim_discord_archiver
./wfpsim_discord_archiver.exe
```

Dry-run (без записи в Google Sheets):

```powershell
./apps/wfpsim_discord_archiver/wfpsim_discord_archiver.exe --dry-run
```

## Примечания

- Секреты (bot token, api key) не коммить.
- State-файл по умолчанию: `work/wfpsim_discord_archiver_state.json`.
