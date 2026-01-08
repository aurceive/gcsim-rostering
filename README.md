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

Скрипт приложения:

- создаёт `input/<app>/config.txt` и `input/<app>/roster_config.yaml` из `input/<app>/examples/`, если файлов ещё нет
- скачивает Go-пакеты только для приложения
- собирает бинарник приложения в `apps/<app>/`

### 3 Настройка конфигов приложения

- Заполните/отредактируйте `input/<app>/config.txt`
- Проверьте/отредактируйте `input/<app>/roster_config.yaml`

### 4 Запуск интересующего приложения

- weapon_roster:

  - основной: `apps/weapon_roster/roster.exe`
  - на примерах: `apps/weapon_roster/roster.exe -useExamples`
- grow_roster:

  - основной: `apps/grow_roster/grow_roster.exe`
  - на примерах: `apps/grow_roster/grow_roster.exe -useExamples`
