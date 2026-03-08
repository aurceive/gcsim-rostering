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

- `engine: gcsim` или `engine: wfpsim` или `engine: wfpsim-custom` или `engine: custom`
- либо `engine_path: <путь>` для явного указания пути к репо движка

Примечания:

- Переключение `engine` влияет на чтение данных/локализаций и на то, какой `engines/bins/<engine>/gcsim.exe` будет запущен.
- Переключение `engine` не требует пересборки `roster.exe`, но требует наличие CLI (`engines/bins/<engine>/gcsim.exe`), собранного из соответствующего сабмодуля.

## Сборка движков

См. `engines/README.md`. Основной вариант скриптом:

- `scripts/engines/bootstrap.ps1`

## Инкрементальная запись таблицы

По умолчанию результат сохраняется в `output/weapon_roster/<YYYYMMDD>_weapon_roster_<char>_<roster>.xlsx`.

Если файл с таким именем уже существует, он будет прочитан и обновлён (добавляются новые строки; совпадающие `оружие+refine+variant` перезаписываются в своём блоке variant).

Если `base_table_path` и `output_table_path` не заданы, приложение попробует найти существующую таблицу для этого `<char>_<roster>` в `output/weapon_roster/` с префиксом текущей даты (`YYYYMMDD_...`) и использовать её как базу для merge; файл результата при этом всё равно будет выбран по умолчанию (по текущей дате).

В `input/weapon_roster/roster_config.yaml` можно управлять тем, как формируется таблица:

- `output_table_path`: путь к XLSX, куда писать результат. Если файл уже существует, он будет прочитан и обновлён (добавляются новые строки; совпадающие `оружие+refine+variant` перезаписываются в своём блоке variant).
- `base_table_path`: путь к XLSX, который будет использован как база для merge (например, чтобы продолжать работу поверх другой таблицы).
- `trust_existing_results`: если `true`, то при merge существующий результат сохраняется, **если новый не лучше** (по выбранному `target`) для того же `оружие+refine+variant`.
- `ignore_existing_results`: если `true`, полностью игнорировать найденные ранее результаты (автопоиск и существующий `output_table_path`); несовместимо с `base_table_path`.
- `weapons`: список оружий для расчёта только выбранных (удобно для пересчёта пары строк). Каждый элемент — либо ключ из данных движка (например, `skywardharp`), либо точное русское имя (строгое полное совпадение). При заданном списке `minimum_weapon_rarity` не применяется, но несовместимые по классу оружия будут отклонены.

`skip_existing_results` и частичное сохранение работают на уровне `оружие+refine+optimizer variant`, а не на уровне целого оружия.

При досрочном завершении (Ctrl+C) приложение экспортирует только те записи `оружие+refine+optimizer variant`, которые были полностью посчитаны.
