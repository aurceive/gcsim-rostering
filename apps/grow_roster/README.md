# grow_roster

Минимальное приложение для прогонов одного конфига на разных уровнях инвестиций и вариантах main stats.

## Сборка

Из корня репозитория:

- `go -C apps/grow_roster build -o grow_roster.exe ./cmd/grow_roster`

## Запуск

- `apps/grow_roster/grow_roster.exe`

Для примеров:

- `apps/grow_roster/grow_roster.exe -useExamples`

## Результат

Файл сохраняется в `output/grow_roster/` по шаблону:

- если `char` задан: `output/grow_roster/<YYYYMMDD>_grow_roster_<char>_<roster_name>.xlsx`
- если `char` не задан: `output/grow_roster/<YYYYMMDD>_grow_roster_<roster_name>.xlsx`
