# Google Apps Script API

Этот код разворачивается как **Web App** и принимает POST-запросы от локального приложения.

## Что делает

- Принимает записи (строки) и добавляет их в Google Sheet.
- После добавления приводит таблицу к "правилам таблицы":
  - сортировка: `TeamCharacters` ASC → блоки `TeamConstellations` по max `TeamDpsMean` DESC → внутри блока `TeamDpsMean` DESC
  - заполнение/вычисление `TeamCharactersUI` (персонажи + созвездия в одном поле)
  - вместо merge: `TeamCharactersUI` заполняется только в первой строке каждой непрерывной группы

## Деплой

1. Открой Apps Script (<https://script.google.com/>) и создай проект.
2. Скопируй содержимое файлов из этой папки:
   - `Code.gs`
   - `appsscript.json`
3. В Script Properties задай `API_KEY` (можно через функцию `setApiKeyForScript_(...)`).
4. Deploy → New deployment → Web app.
   - Execute as: **Me**
   - Who has access: **Anyone** (или ограниченный вариант, если подходит)
5. Сохрани `webAppUrl` в `input/wfpsim_discord_archiver/config.yaml`.

## Что такое `sheetId` / SPREADSHEET_ID

`sheetId` — это идентификатор *документа* Google Spreadsheet.
Его можно взять из URL таблицы:

`https://docs.google.com/spreadsheets/d/<SPREADSHEET_ID>/edit#gid=...`

`sheetName` — это имя вкладки (tab) внутри документа (например `wfpsim`).

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
