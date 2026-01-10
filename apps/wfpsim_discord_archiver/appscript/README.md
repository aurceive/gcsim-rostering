# Google Apps Script API

Этот код разворачивается как **Web App** и принимает POST-запросы от локального приложения.

## Что делает

- Принимает записи (строки) и добавляет их в Google Sheet.
- После добавления сортирует данные:
  - по `TeamCharacters` (возрастание)
  - затем по `TeamDpsMean` (убывание)

## Деплой

1. Открой Apps Script (https://script.google.com/) и создай проект.
2. Скопируй содержимое файлов из этой папки:
   - `Code.gs`
   - `appsscript.json`
3. В Script Properties задай `API_KEY` (можно через функцию `setApiKeyForScript_(...)`).
4. Deploy → New deployment → Web app.
   - Execute as: **Me**
   - Who has access: **Anyone** (или ограниченный вариант, если подходит)
5. Сохрани `webAppUrl` в `input/wfpsim_discord_archiver/config.yaml`.

## Формат запроса

POST JSON:

```json
{
  "apiKey": "...",
  "sheetId": "...",
  "sheetName": "wfpsim",
  "record": {
    "row": ["FetchedAt", "DiscordGuildID", "..." ]
  }
}
```
