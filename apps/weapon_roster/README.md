# weapon_roster

Приложение для работы с ростерами оружия. Корневой README описывает только основной пайплайн; здесь — детали.

## Сборка

Из корня репозитория:

- `go -C apps/weapon_roster build -o roster.exe ./cmd/weapon_roster`

## Запуск

- `apps/weapon_roster/roster.exe`

Если `input/weapon_roster/config.txt` или `input/weapon_roster/roster_config.yaml` отсутствуют, запустите `scripts/weapon_roster/bootstrap.ps1` — он создаст их из `input/weapon_roster/examples/`.

## Выбор активного движка

В `input/weapon_roster/roster_config.yaml`:

- `engine: gcsim` или `engine: wfpsim` или `engine: custom`
- либо `engine_path: <путь>` для явного указания пути к репо движка

Примечания:

- Переключение `engine` влияет на чтение данных/локализаций и на то, какой `engines/bins/<engine>/gcsim.exe` будет запущен.
- Переключение `engine` не требует пересборки `roster.exe`, но требует наличие CLI (`engines/bins/<engine>/gcsim.exe`), собранного из соответствующего сабмодуля.

## Сборка движков

См. `engines/README.md`. Основной вариант скриптом:

- `scripts/engines/bootstrap.ps1`
