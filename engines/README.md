# Engines (сабмодули)

Этот каталог содержит сабмодули движков. Корневой README оставляет только основной пайплайн, а здесь — детали/варианты.

## Привязка сабмодулей к веткам

Сабмодули уже привязаны к веткам в `.gitmodules`:

- `engines/gcsim` → `main`
- `engines/wfpsim` → `develop`
- `engines/custom` → `custom`
- `engines/wfpsim-custom` → `wfpsim-custom`

Примечание: если указанная ветка отсутствует на `origin`, скрипт обновления использует fallback на `origin/HEAD`.

## Инициализация/обновление сабмодулей

### Вариант 1: скриптом (PowerShell)

- `../scripts/engines/update-submodules.ps1`

### Вариант 2: команды Git (PowerShell)

- `git submodule sync --recursive`
- `git submodule update --init --recursive --depth 1 --recommend-shallow`
- `git submodule update --remote --recursive --depth 1 --recommend-shallow`

## Go-зависимости

Если нужно заранее скачать зависимости во всех движках:

- `go -C gcsim mod download`
- `go -C wfpsim mod download`
- `go -C custom mod download`
- `go -C wfpsim-custom mod download`

## Сборка CLI движков

Бинарники движков складываются в `engines/bins/<engine>/` (это держит сабмодули чистыми).

### Вариант 1: сборка скриптом (PowerShell)

- Собрать все 3 движка (gcsim/wfpsim/custom) и цели `gcsim`, `repl`, `server`:
  - `../scripts/engines/build-engine-clis.ps1`

- Собрать все 4 движка (gcsim/wfpsim/custom/wfpsim-custom):
  - `../scripts/engines/build-engine-clis.ps1 -Engine all`

Также можно использовать общий скрипт (обновление сабмодулей + download + сборка CLI):

- `../scripts/engines/bootstrap.ps1`

Примеры результата (пути):

- `bins/gcsim/gcsim.exe`
- `bins/wfpsim/gcsim.exe`
- `bins/custom/gcsim.exe`
- `bins/wfpsim-custom/gcsim.exe`

### Вариант 2: вручную (PowerShell)

- gcsim:
  - `go -C gcsim build -o ..\bins\gcsim\gcsim.exe ./cmd/gcsim`
- wfpsim:
  - `go -C wfpsim build -o ..\bins\wfpsim\gcsim.exe ./cmd/gcsim`
- custom:
  - `go -C custom build -o ..\bins\custom\gcsim.exe ./cmd/gcsim`
- wfpsim-custom:
  - `go -C wfpsim-custom build -o ..\bins\wfpsim-custom\gcsim.exe ./cmd/gcsim`
