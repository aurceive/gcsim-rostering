# Движки-симуляторы

Каждый движок подключается как Git submodule непосредственно в `engines/<имя>`. URL, ветка и параметры получения submodule задаются в корневом `.gitmodules`. Скрипты сборки читают имена движков из этой конфигурации, поэтому новые submodule в `engines/` доступны без обновления списков в скриптах.

Каталог `engines/bins/` используется для локальных бинарников и не является submodule.

## Инициализация и обновление

PowerShell-скрипт синхронизирует URL, инициализирует submodule и обновляет их до указанных в `.gitmodules` веток:

```powershell
scripts/engines/update-submodules.ps1
```

Стандартные команды Git:

```powershell
git submodule sync --recursive
git submodule update --init --recursive --depth 1 --recommend-shallow
```

Чтобы обновить только один движок до ветки, настроенной в `.gitmodules`, передайте его имя:

```powershell
scripts/engines/update-submodules.ps1 -Engine kqmsim
```

Без `-Engine` обновляются все настроенные движки. Например, движок может быть подключён с ветки `develop` или `main`; актуальная настройка находится в `.gitmodules`.

`kqmsim` отслеживает upstream-ветку `develop` репозитория KQM, а `kqmsim-custom` — ветку `kqmsim-custom` в форке `aurceive/gcsim`.

## Go-зависимости и сборка CLI

Скрипты поддерживают имя любого настроенного submodule в `engines/`, а значение `all` выбирает все такие движки. Примеры:

```powershell
# Инициализировать/обновить submodules, загрузить Go-модули и собрать gcsim CLI выбранного движка
scripts/engines/bootstrap.ps1 -Engine kqmsim

# То же для пользовательской ветки KQM
scripts/engines/bootstrap.ps1 -Engine kqmsim-custom

# Собрать стандартные цели для всех настроенных движков
scripts/engines/build-engine-clis.ps1 -Engine all

# Собрать только CLI и server для одного движка
scripts/engines/build-engine-clis.ps1 -Engine wfpsim -Targets gcsim,server

# Загрузить Go-модули и собрать несколько целей одного движка
scripts/engines/bootstrap.ps1 -Engine gcsim -Targets gcsim,repl,server
```

`-Engine` применяется одинаково к обновлению submodule, загрузке Go-зависимостей и сборке CLI. По умолчанию используется `all`, поэтому запуск без параметров обрабатывает все настроенные движки. Чтобы ограничить все этапы одним движком, например, используйте `-Engine kqmsim`.

Результат сборки сохраняется вне каталогов submodule:

```text
engines/bins/<имя-движка>/<цель>.exe
```

Например, для `kqmsim` CLI будет расположен в `engines/bins/kqmsim/gcsim.exe`.

## Server mode

Запустите сервер для выбранного движка:

```powershell
scripts/engines/launch-server.ps1 -Engine kqmsim
scripts/engines/launch-server.ps1 -Engine kqmsim-custom
scripts/engines/launch-server.ps1 -Engine gcsim -Port 54322 -Workers 4
```

Параметры подключения (адрес, порт, число workers, timeout и share key) настраиваются параметрами `launch-server.ps1`. Для уже собранного сервера можно указать `-NoBuild`; дополнительные опции обновления описаны в `Get-Help scripts/engines/launch-server.ps1 -Full`.

## Совместимость с приложениями

Общий пакет `libs/enginebridge` нормализует расположение корня движка, CLI и игровые локализации. Он читает как `names.generated.json`, так и пару `names.dm.json` + `names.override.json`, применяя override поверх базовых имён. `weapon_roster` и приложения, использующие имена персонажей в архиваторе, используют этот общий слой.

Поддержка отдельных игровых справочников зависит от приложения. Ветки различаются регистрацией оружия/персонажей и структурой метаданных: например, KQM использует pipeline-конфиги и DM JSON (`character.dm.json`, `weapon.dm.json`, `artifact.dm.json`), в то время как более старые загрузчики могли ожидать `data_gen.textproto` или сгенерированные JSON-файлы. `weapon_roster` и `talent_comparator` адаптируют регистрации из обоих форматов, а `enka_import` читает оба варианта игровых справочников. Другие приложения могут поддерживать только CLI и отдельные форматы данных; сборка CLI сама по себе не означает полной совместимости.

Поддержка KQM в приложениях по-прежнему не универсальна: адаптированы `weapon_roster`, `talent_comparator` и `enka_import`, а `grow_roster` может обнаружить корень движка и вызвать его CLI. Совместимость с CLI не гарантирует, что каждое приложение понимает все специфичные данные, форматы результатов и параметры запуска движка.
