# Rostering Application

Правила организации структуры репозитория: [docs/repo-structure.md](docs/repo-structure.md)

## Пайплайн (Windows / PowerShell)

### 1 Обновление и сборка движков (если требуется)

- `scripts/engines/bootstrap.ps1`

Скрипт:

- обновляет сабмодули
- скачивает Go-пакеты для движков
- собирает CLI движков в `engines/bins/<engine>/`

### 2 Сборка интересующего приложения

- weapon_roster: `scripts/weapon_roster/bootstrap.ps1`
- grow_roster: `scripts/grow_roster/bootstrap.ps1`
- enka_import: `scripts/enka_import/bootstrap.ps1`
- wfpsim_discord_archiver: `scripts/wfpsim_discord_archiver/bootstrap.ps1`

Скрипт приложения:

- создаёт `input/<app>/config.txt` и `input/<app>/roster_config.yaml` из `input/<app>/examples/`, если файлов ещё нет
- скачивает Go-пакеты только для приложения
- собирает бинарник приложения в `apps/<app>/`

### 3 Настройка конфигов приложения

- Заполните/отредактируйте `input/<app>/config.txt`
- Проверьте/отредактируйте `input/<app>/roster_config.yaml`
- Для wfpsim_discord_archiver: отредактируйте `input/wfpsim_discord_archiver/config.yaml`

### 4 Запуск интересующего приложения

- weapon_roster:

  - основной: `apps/weapon_roster/roster.exe`
  - на примерах: `apps/weapon_roster/roster.exe -useExamples`

- grow_roster:

  - основной: `apps/grow_roster/grow_roster.exe`
  - на примерах: `apps/grow_roster/grow_roster.exe -useExamples`

- wfpsim_discord_archiver:

  - основной: `apps/wfpsim_discord_archiver/wfpsim_discord_archiver.exe`

- enka_import:

  - основной: `apps/enka_import/enka_import.exe`
  - на примерах: `apps/enka_import/enka_import.exe -useExamples`

## Запуск server mode для движков

### 1 Обновление и сборка движков для server mode (если требуется)

- `scripts/engines/bootstrap.ps1`

### 2 Запуск

Скрипт соберёт `server.exe` в `engines/bins/<engine>/` и запустит его.

- `scripts/engines/launch-server.ps1 -Engine gcsim`
- `scripts/engines/launch-server.ps1 -Engine wfpsim`
- `scripts/engines/launch-server.ps1 -Engine custom`
- `scripts/engines/launch-server.ps1 -Engine wfpsim-custom`

### 3 Подключение из браузера

- Откройте UI (например <https://gcsim.app>)
- Включите “server mode”
- Укажите URL `http://127.0.0.1:54321`
