# talent_comparator

Запускает один и тот же gcsim-конфиг несколько раз, **меняя уровни талантов** только у выбранного персонажа.

Особенности:

- Запуск идёт **без** флага оптимизации сабстатов (нет `-substatOptimFull`).
- Результат сохраняется в `output/talent_comparator/`.
- Имя файла: `YYYYMMDD_<char>_<name>.xlsx`.


## Входные файлы

- `input/talent_comparator/config.txt` — gcsim-конфиг симуляции.
- `input/talent_comparator/talent_config.yaml` — настройки приложения.

## Сборка и запуск (Windows)

1. Собрать движок: `scripts/engines/bootstrap.ps1`
2. Собрать приложение и создать локальные конфиги: `scripts/talent_comparator/bootstrap.ps1`
3. Запуск: `apps/talent_comparator/talent_comparator.exe`

Можно проверить на примерах: `apps/talent_comparator/talent_comparator.exe -useExamples`
