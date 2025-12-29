# weapon_roster

Приложение для работы с ростерами оружия. Корневой README описывает только основной пайплайн; здесь — детали.

## Сборка

Из корня репозитория:

- `go -C apps/weapon_roster build -o roster.exe ./cmd/weapon_roster`

## Запуск

- `apps/weapon_roster/roster.exe`

## Выбор активного движка

В `roster_config.yaml`:

- `engine: gcsim` или `engine: wfpsim` или `engine: custom`
- либо `engine_path: <путь>` для явного указания пути к репо движка

Примечания:

- Переключение `engine` влияет на чтение данных/локализаций и на то, какой `engines/bins/<engine>/gcsim.exe` будет запущен.
- Переключение `engine` не требует пересборки `roster.exe`, но требует наличие CLI (`engines/bins/<engine>/gcsim.exe`), собранного из соответствующего сабмодуля.

## Сборка движков

См. `engines/README.md` (есть варианты через скрипт и ручные команды).
