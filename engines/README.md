# Engines (сабмодули)

Этот каталог содержит сабмодули движков. Корневой README оставляет только основной пайплайн, а здесь — детали/варианты.

## Привязка сабмодулей к веткам

Сабмодули уже привязаны к веткам в `.gitmodules`:

- `engines/gcsim` → `main`
- `engines/wfpsim` → `develop`
- `engines/custom` → `custom`

Примечание: если указанная ветка отсутствует на `origin`, скрипт обновления использует fallback на `origin/HEAD`.

## Инициализация/обновление сабмодулей

### Вариант 1: скриптом (PowerShell)

- `../scripts/submodules/update-submodules.ps1`

### Вариант 2: команды Git (PowerShell)

- `git submodule sync --recursive`
- `git submodule update --init --recursive --depth 1 --recommend-shallow`
- `git submodule update --remote --recursive --depth 1 --recommend-shallow`

## Go-зависимости

Если нужно заранее скачать зависимости во всех движках:

- `go -C gcsim mod download`
- `go -C wfpsim mod download`
- `go -C custom mod download`

## Сборка CLI движков

В сабмодулях игнорируется любой `*.exe`, поэтому бинарники можно оставлять прямо в корне сабмодуля.

### Вариант 1: сборка скриптом (PowerShell)

- Собрать все 3 движка (gcsim/wfpsim/custom) и цели `gcsim`, `repl`, `server`:
  - `../scripts/submodules/build-engine-clis.ps1`

Примеры результата (пути):

- `gcsim/gcsim.exe`
- `wfpsim/gcsim.exe`
- `custom/gcsim.exe`

### Вариант 2: вручную (PowerShell)

- gcsim:
  - `go -C gcsim build -o gcsim.exe ./cmd/gcsim`
- wfpsim:
  - `go -C wfpsim build -o gcsim.exe ./cmd/gcsim`
- custom:
  - `go -C custom build -o gcsim.exe ./cmd/gcsim`
