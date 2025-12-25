# Rostering app

## Инициализация сабмодулей (shallow)

- `git submodule update --init --recursive --depth 1 --recommend-shallow`

## Сборка и запуск (из корня репозитория)

- Сборка: `go -C app build -o app/roster.exe .`
- Запуск: `./app/roster.exe`

## Выбор активного движка

- В `roster_config.yaml`:
  - `engine: gcsim` или `engine: wfpsim` или `engine: custom`
  - либо `engine_path: <путь>` для явного указания пути к репо движка

Примечание: переключение `engine` влияет на чтение данных/локализации; выбор Go-реализации `optimization/simulator` определяется при сборке через `replace` в `app/go.mod`.
