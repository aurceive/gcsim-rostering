# enka_import

CLI-утилита для импорта персонажей из Enka.Network по UID и экспорта в текстовый файл в формате gcsim config (`char`/`add weapon`/`add set`/`add stats`).

## Сборка (Windows)

Из корня репозитория:

```powershell
scripts/enka_import/bootstrap.ps1
```

## Запуск

Пример:

```powershell
apps/enka_import/enka_import.exe
```

По умолчанию утилита читает настройки из `input/enka_import/config.yaml`.

Также можно переопределять любые поля через флаги:

```powershell
apps/enka_import/enka_import.exe -engine wfpsim-custom -uid 123456789
```

Флаги:

- `-config` — путь до YAML-конфига (по умолчанию `input/enka_import/config.yaml`).
- `-useExamples` — использовать пример конфига из `input/enka_import/examples/`.
- `-engine` — имя движка из папки `engines/` (например: `gcsim`, `wfpsim`, `wfpsim-custom`, `custom`).
- `-engine-path` — явный путь до `engines/<engine>` (если не стандартное расположение).
- `-uid` — UID игрока (9 цифр).
- `-out` — полный путь к результирующему `.txt` (перекрывает `-out-dir`).
- `-out-dir` — папка для результата (по умолчанию `output/enka_import`).
- `-include-builds` — дополнительно подтягивать builds профиля Enka (по умолчанию `true`).

Примечания:

- Статы берутся **только из артефактов** (main+sub), как в UI-импорте gcsim.
- В итоговом файле `add stats` пишется двумя строками: `#main` и субстаты.

Вывод:

- Если `outPath`/`-out` не заданы, файл создаётся в `output/enka_import/` по шаблону: `<YYYYMMDD>_<profileName>.txt`.
- Если имя профиля получить не удалось, используется UID: `<YYYYMMDD>_<uid>.txt`.
