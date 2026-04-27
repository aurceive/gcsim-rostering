# constellation_comparator

Сравнивает DPS отряда при различных уровнях созвездий для 1–4 персонажей.

## Использование

### Первый запуск

```powershell
# Из корня репозитория
scripts/engines/bootstrap.ps1                       # сборка движка (если не готов)
scripts/constellation_comparator/bootstrap.ps1      # создаёт конфиги и собирает .exe
```

Bootstrap-скрипт:
- создаёт `input/constellation_comparator/config.txt` и `constellation_config.yaml` из `examples/`, если файлов ещё нет
- скачивает Go-модули
- собирает `apps/constellation_comparator/constellation_comparator.exe`

### Запуск

```powershell
apps/constellation_comparator/constellation_comparator.exe            # основной
apps/constellation_comparator/constellation_comparator.exe -useExamples  # на примерах
```

Или напрямую через `go run` (без сборки):

```powershell
cd apps/constellation_comparator
go run ./cmd/constellation_comparator
go run ./cmd/constellation_comparator -useExamples
```

## Настройки (`constellation_config.yaml`)

| Ключ | Тип | Обязательный | Описание |
|---|---|---|---|
| `name` | string | да | Имя прогона (часть имени xlsx) |
| `chars` | list[string] | да | 1–4 персонажа, созвездия которых повышаем |
| `engine` | string | нет | Имя движка из `engines/` (default: `gcsim`) |
| `engine_path` | string | нет | Абсолютный путь к репозиторию движка |
| `max_additional` | int | нет | Лимит суммарных доп. созвездий. Если не задан — без лимита |
| `ignore_existing_results` | bool | нет | Не читать сегодняшний xlsx, начать заново |

## Комбинаторика

| Перс. на C0 | max_additional=∞ | =3 | =6 |
|---|---|---|---|
| 1 | 7 | 4 | 7 |
| 2 | 49 | 10 | 28 |
| 3 | 343 | 20 | 84 |
| 4 | 2401 | 35 | 210 |

## Выходные файлы

Сохраняются в `output/constellation_comparator/` с именем `YYYYMMDD_constellation_comparator_{name}.xlsx`.

### Лист Summary
Лучшая (наивысший Team DPS) вариация на каждый уровень доп. созвездий.

Колонки: `Доп. конст | Team DPS | Team % | [Char1] … [CharN]` + `Sim Config` (через колонку)

### Лист Full
Все вариации, отсортированные по `Доп. конст` ASC → `Team DPS` DESC.

Дополнительная колонка `Best %`: 100% = лучший результат на том же уровне доп. созвездий.

Колонки: `Доп. конст | Team DPS | Team % | Best % | [Char1] … [CharN]` + `Sim Config` (через колонку)

## Защита от сбоев и досчитывание

- **Ctrl+C**: при прерывании экспортирует уже посчитанные результаты.
- **Ошибки движка**: нефатальные — записываются как 0 DPS, прогон продолжается.
- **Resume**: при повторном запуске в тот же день автоматически находит сегодняшний xlsx и досчитывает недостающие комбинации.
