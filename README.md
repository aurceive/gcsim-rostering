# Rostering app

## Инициализация сабмодулей (shallow)

- `git submodule update --init --recursive --depth 1 --recommend-shallow`

## Обновление сабмодулей до актуального состояния (без редактирования файлов)

Сабмодули уже привязаны к веткам в `.gitmodules`:

- `engines/gcsim` → `main`
- `engines/wfpsim` → `develop`
- `engines/custom` → `custom`

Примечание: если указанная ветка отсутствует на `origin`, скрипт обновления использует fallback на `origin/HEAD`.

### Вариант 1: одной командой (PowerShell)

- `./scripts/update-submodules.ps1`

### Вариант 2: команды Git (PowerShell)

- `git submodule sync --recursive`
- `git submodule update --init --recursive --depth 1 --recommend-shallow`
- `git submodule update --remote --recursive --depth 1 --recommend-shallow`

## Сборка и запуск (из корня репозитория)

- Сборка:
  go -C apps/weapon_roster build -o roster.exe ./cmd/weapon_roster
- Запуск:
  apps/weapon_roster/roster.exe

## Сборка CLI движков (внутри сабмодулей)

В сабмодулях игнорируется любой `*.exe`, поэтому бинарники можно оставлять прямо в корне сабмодуля.

### Вариант 1: одной командой (PowerShell)

- Собрать все 3 движка (gcsim/wfpsim/custom) и цели `gcsim`, `repl`, `server`:
  - `./scripts/build-engine-clis.ps1`

Примеры результата (пути):

- `engines/gcsim/gcsim.exe`
- `engines/wfpsim/gcsim.exe`
- `engines/custom/gcsim.exe`

### Вариант 2: вручную (PowerShell)

- gcsim:
  - `go -C engines/gcsim build -o gcsim.exe ./cmd/gcsim`
- wfpsim:
  - `go -C engines/wfpsim build -o gcsim.exe ./cmd/gcsim`
- custom:
  - `go -C engines/custom build -o gcsim.exe ./cmd/gcsim`

## Выбор активного движка

- В `roster_config.yaml`:
  - `engine: gcsim` или `engine: wfpsim` или `engine: custom`
  - либо `engine_path: <путь>` для явного указания пути к репо движка

Примечание: переключение `engine` влияет на чтение данных/локализации и на то, какой `engines/<engine>/gcsim.exe` будет запущен.

Примечание: `apps/weapon_roster` запускает движок через CLI `gcsim.exe` внутри соответствующего сабмодуля, поэтому переключение `engine` не требует пересборки `roster.exe`, но требует наличие `engines/<engine>/gcsim.exe` (см. сборку CLI движков выше).
