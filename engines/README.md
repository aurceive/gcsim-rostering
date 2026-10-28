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

Чтобы обновить рабочие копии до веток, настроенных в `.gitmodules`, используйте `scripts/engines/update-submodules.ps1`. Например, движок может быть подключён с ветки `develop` или `main`; актуальная настройка находится в `.gitmodules`.

## Go-зависимости и сборка CLI

Скрипты поддерживают имя любого настроенного submodule в `engines/`, а значение `all` выбирает все такие движки. Примеры:

```powershell
# Инициализировать/обновить submodules, загрузить Go-модули и собрать gcsim CLI выбранного движка
scripts/engines/bootstrap.ps1 -Engine kqmsim

# Собрать стандартные цели для всех настроенных движков
scripts/engines/build-engine-clis.ps1 -Engine all

# Собрать только CLI и server для одного движка
scripts/engines/build-engine-clis.ps1 -Engine wfpsim -Targets gcsim,server

# Загрузить Go-модули и собрать несколько целей одного движка
scripts/engines/bootstrap.ps1 -Engine gcsim -Targets gcsim,repl,server
```

`bootstrap.ps1` по умолчанию обновляет все submodule, но загружает зависимости и собирает CLI только для выбранного движка (`gcsim` по умолчанию). Используйте `-Engine all`, чтобы выбрать все настроенные движки.

Результат сборки сохраняется вне каталогов submodule:

```text
engines/bins/<имя-движка>/<цель>.exe
```

Например, для `kqmsim` CLI будет расположен в `engines/bins/kqmsim/gcsim.exe`.

## Server mode

Запустите сервер для выбранного движка:

```powershell
scripts/engines/launch-server.ps1 -Engine kqmsim
scripts/engines/launch-server.ps1 -Engine gcsim -Port 54322 -Workers 4
```

Параметры подключения (адрес, порт, число workers, timeout и share key) настраиваются параметрами `launch-server.ps1`. Для уже собранного сервера можно указать `-NoBuild`; дополнительные опции обновления описаны в `Get-Help scripts/engines/launch-server.ps1 -Full`.

## Совместимость с приложениями

Общий пакет `libs/enginebridge` нормализует расположение корня движка, CLI и игровые локализации. Он читает как `names.generated.json`, так и пару `names.dm.json` + `names.override.json`, применяя override поверх базовых имён. `weapon_roster` и приложения, использующие имена персонажей в архиваторе, используют этот общий слой.

Поддержка отдельных игровых справочников зависит от приложения. Ветки различаются регистрацией оружия/персонажей и структурой метаданных: например, KQM размещает часть данных в `ui/packages/db/src/Data` и использует pipeline-конфиги, в то время как некоторые загрузчики ожидают `data_gen.textproto` или JSON-файлы из UI-пакета. Сборка CLI не означает автоматическую поддержку всех таких форматов; приложения должны явно сообщать об отсутствующих или неподдерживаемых данных.

Текущий адаптер KQM покрывает локализации и generated-файл псевдонимов персонажей. Это ещё не означает полной поддержки KQM в приложениях: например, `weapon_roster` пока требует метаданные оружия в `data_gen.textproto`, которых pipeline-конфиги KQM не предоставляют, а загрузчики `enka_import`, `grow_roster` и `talent_comparator` пока ожидают прежние файлы игровых данных.
