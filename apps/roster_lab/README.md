# roster_lab

`roster_lab` объединяет расчёт оружий, инвестиций, созвездий, уровней и талантов для одного целевого персонажа. Приложение использует один `sim_config.dart` и запускает независимые сценарии из `roster_lab.yaml`. Стандарты оптимизации задаются глобально в `optimizer_variants`; `weapons` и `growth` могут полностью заменить их своим `optimizer_variants`. Случайное декартово произведение сценариев не генерируется.

## Сборка

Из корня репозитория:

```powershell
scripts\roster_lab\bootstrap.ps1
```

Ручная сборка:

```powershell
go -C apps\roster_lab build -o roster_lab.exe .\cmd\roster_lab
```

## Запуск

```powershell
apps\roster_lab\roster_lab.exe
apps\roster_lab\roster_lab.exe -useExamples
```

Входы располагаются в:

```text
input\roster_lab\roster_lab.yaml
input\roster_lab\sim_config.dart
```

Файл `sim_config.dart` выбирается неявно. Поле `project.sim_config` можно указать только для нестандартного имени или абсолютного пути.

`input\roster_lab\examples\roster_lab.example.yaml` является обязательным контрактом настроек: он содержит все поддерживаемые и зарезервированные поля, значения по умолчанию и альтернативные формы. При добавлении новой YAML-возможности этот файл должен обновляться одновременно.

## Resume

По умолчанию приложение ищет ODS-файл в `output\roster_lab`, созданный за последние 24 часа:

```powershell
apps\roster_lab\roster_lab.exe -resume auto
```

Другие режимы:

```powershell
apps\roster_lab\roster_lab.exe -resume off
apps\roster_lab\roster_lab.exe -resume all
apps\roster_lab\roster_lab.exe -resume output\roster_lab\result.ods
```

Завершённые прогоны пропускаются. `failed` считается синонимом «не посчитано» и запускается повторно. При несовпадении исходного YAML или `sim_config.dart` в режимах `auto`/`all` приложение запрашивает подтверждение слияния. Для явного пути подтверждение не требуется.

## Формат вывода

Результат — один автономно созданный ODS-файл:

- `Run Summary`;
- отдельный лист для каждого активного сценария;
- `Technical` с checkpoint и исходными файлами;
- `Errors`, если остались неуспешные прогоны.

Неактивные сценарии не создают листов. Все ячейки имеют сетку.

## Текущие ограничения

- поддерживается один целевой персонаж;
- team scope зарезервирован в формате, но пока отклоняется;
- прогоны выполняются последовательно;
- поддержка XLSX отсутствует;
- внешний редактор или LibreOffice для генерации не требуется.
