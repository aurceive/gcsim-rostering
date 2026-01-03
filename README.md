# Rostering Application

## Weapon roster (Windows / PowerShell)

### 1 Актуализировать + подгрузить зависимости + собрать

- `scripts/weapon_roster/bootstrap.ps1`

Скрипт:

- обновляет сабмодули
- скачивает Go-пакеты во всех Go-модулях
- собирает `apps/weapon_roster/roster.exe`
- собирает CLI движков

### 2 Настроить

- Заполните `input/weapon_roster/config.txt`
- Проверьте настройки в `input/weapon_roster/roster_config.yaml`.

### 3 Запустить

- Основной: `apps/weapon_roster/roster.exe`
- На примерах: `apps/weapon_roster/roster.exe -useExamples`
