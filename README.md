# Rostering Application

## Weapon roster (Windows / PowerShell)

### 1 Актуализировать + подгрузить зависимости + собрать

- `./scripts/weapon_roster/bootstrap.ps1`

Скрипт:

- обновляет сабмодули
- скачивает Go-пакеты во всех Go-модулях
- собирает `apps/weapon_roster/roster.exe`
- собирает CLI движков (в т.ч. `engines/<engine>/gcsim.exe`)

### 2 Запустить

- Заполните `config.txt` и проверьте настройки в `apps/weapon_roster/roster_config.yaml`.
- `apps/weapon_roster/roster.exe`
