# Plan: Автообновление терминала (проверка релизов GitHub + self-update с перезапуском)

## Overview
Новый пакет `updater/` на стандартной библиотеке: фоновая проверка GitHub Releases раз в сутки с кэшем в `~/.finam-cli/update.json`, значок `⚡` в шапке TUI, диалог «Обновить и перезапустить / Продолжить» при запуске и по клавише `U`, self-update скачиванием релизного ассета со сверкой SHA256 и атомарной подменой бинарника. Разработка по TDD; фазы 1–3 дают полностью протестированное ядро без единой правки в UI, фазы 4–5 подключают его к `main.go` и интерфейсу, фазы 6–7 закрывают CI, документацию и ручную проверку.

## Phase 1: Ядро — версии и файл состояния
- [x] Task: (Red) Тесты `updater/semver_test.go`: `IsRelease` (`v0.14.0`/`0.14.0` → true; `dev`, `dev (a1b2c3d)`, `""`, мусор → false), `Compare` (мажор/минор/патч, префикс `v` опционален, `v1.0.0-rc1` < `v1.0.0`), `IsNewer` (новее → true; равные, downgrade, нерелизная сторона → false)
  - Acceptance: тесты компилируются и падают, фиксируя таблицу сравнений (5cbc6f0)
- [ ] Task: (Green) `updater/semver.go` — парсер semver без внешних зависимостей + `IsRelease`/`Compare`/`IsNewer` с GoDoc
  - Acceptance: тесты фазы зелёные; `go vet ./updater/...` чист
- [ ] Task: (Red→Green) `config.UserConfigDir()` (экспортированный каталог `~/.finam-cli`, переиспользован в `saveTokenInternal`) + `updater/state.go`: `LoadState`/`SaveState`; тесты через подменяемый каталог: отсутствующий файл → нулевое состояние без ошибки, битый JSON → нулевое состояние + `[WARN]`, круговая запись/чтение, атомарность (temp+rename, после записи в каталоге нет `.tmp`)
  - Acceptance: тесты зелёные; существующие тесты `config` не сломаны; путь совпадает с каталогом `.env`

## Phase 2: Проверка обновлений (GitHub API + планировщик)
- [ ] Task: (Red) `updater/github_test.go` на `httptest`: разбор `tag_name`/`html_url`/`published_at`/`assets`, выбор ассета по имени, 404/500/невалидный JSON → ошибка, соблюдение таймаута и заголовков `Accept`/`User-Agent`
  - Acceptance: тесты компилируются и падают
- [ ] Task: (Green) `updater/github.go` — `Release`/`Asset`, `FetchLatestRelease(ctx)`, подменяемый `apiBaseURL`, таймаут 10 с, единый `[WARN]`-лог ошибок
  - Acceptance: тесты фазы зелёные; сеть в тестах не используется
- [ ] Task: (Red→Green) `updater/checker.go` — `ShouldCheck(state, now)` (сутки, нулевое время → true) и `Run(ctx, current, onNewVersion)`; тесты: свежее состояние → запросов нет, просроченное → один запрос + сохранение состояния, найденная новая версия → ровно один вызов колбэка (повторная проверка той же версии — без дубля), `!IsRelease(current)` → мгновенный возврат без запросов и без файла, `ctx.Done()` останавливает цикл
  - Acceptance: тесты зелёные под `-race`, без `time.Sleep` в ожиданиях (каналы/подменяемые часы)

## Phase 3: Self-update — скачивание, целостность, подмена
- [ ] Task: (Red→Green) `updater/asset.go` — `AssetName(goos, goarch)` для 4 поддерживаемых комбинаций + ошибка с текстом платформы для остальных; табличный тест, включая `linux/arm64` и `windows/arm64`
  - Acceptance: имена совпадают с артефактами `.github/workflows/release.yml`
- [ ] Task: (Red) Тесты скачивания `updater/download_test.go` на `httptest`: файл сохраняется во временный путь, `progress` получает монотонный прогресс, SHA256 из `checksums.txt` сверяется, подменённое тело → ошибка и удаление temp, отсутствующий `checksums.txt` → фоллбек на сверку размера, несовпадение размера → ошибка, отмена `ctx` посреди тела → ошибка и удаление temp
  - Acceptance: тесты компилируются и падают
- [ ] Task: (Green) `updater/download.go` — потоковое скачивание с `io.MultiWriter`+`sha256`, парсер `checksums.txt`, фоллбек по размеру, `defer`-очистка temp, таймаут 5 минут
  - Acceptance: тесты фазы зелёные; временных файлов после прогона не остаётся
- [ ] Task: (Red→Green) `updater/apply.go` — `SelfUpdate` (реальный путь через `os.Executable`+`EvalSymlinks`, проба записи → `ErrNotWritable` с командой ручного обновления, `chmod 0755` на Unix) + `replaceExecutable` (`exe→exe.old`, `tmp→exe`, откат при сбое) + `CleanupStaleBackup`; тесты на временном каталоге с бутафорским «бинарником»: успешная подмена, откат при неудаче второго rename, отказ по правам, чистка `.old`
  - Acceptance: тесты зелёные на текущей ОС; при любой ошибке исходный файл остаётся байт-в-байт прежним

## Phase 4: Перезапуск и подключение к main.go
- [ ] Task: (Red→Green) `updater/restart_unix.go` / `restart_windows.go` (build-теги в стиле `platform/console_*.go`): `Restart(exePath)` через подменяемую переменную-хук; тест проверяет переданные путь/аргументы/окружение без реального exec
  - Acceptance: `go build ./...` проходит для обеих веток (`GOOS=windows` и `GOOS=linux` кросс-сборка)
- [ ] Task: `ui/update_flow.go` — `RunUpdateFlow(rel)`: консольный прогресс-бар в стиле `RunStartupSteps`, печать итога, понятный текст ошибок (включая команду ручной установки); unit-тест рендера прогресса на подменённом writer
  - Acceptance: вывод не ломает консоль при ошибке и при 100%
- [ ] Task: Интеграция в `main.go`: `CleanupStaleBackup()` в начале; после сплэша — `LoadState` + диалог при `IsNewer` + `go updater.Run(...)`; после `app.Run()` — обработка `app.UpdateRequested()`; ошибки обновления не прерывают запуск (сообщение + пауза + обычный старт)
  - Acceptance: `go run main.go` на dev-сборке ведёт себя ровно как раньше — ни файла, ни сетевых запросов, ни задержек

## Phase 5: UI — диалог, индикатор, горячая клавиша
- [ ] Task: (Red→Green) `headerLabel(current, latest)` в `ui/components.go` + `SetDynamicColors(true)` для шапки; тесты: без обновления — прежняя строка (включая правило префикса `v`), с обновлением — `⚡` и номер новой версии
  - Acceptance: существующие тесты `ui/header_test.go` зелёные без правок ожиданий для случая «обновления нет»
- [ ] Task: `ui/update_prompt.go` — `NewUpdatePromptApp(current, latest)` (tview-модалка по образцу `NewSetupApp`, `Run() bool`, дефолт и `Esc` → «Продолжить») + тест выбора кнопок без реального терминала
  - Acceptance: обе версии видны в тексте; выбор корректно возвращается вызывающему
- [ ] Task: `App.SetUpdateAvailable(latest)` + модалка по клавише `u/U/г/Г` в главном экране (`ui/input.go`), флаг `updateRequested` + `UpdateRequested()`, сообщение в статус-баре при отсутствии обновления; unit-тесты: сеттер перерисовывает шапку, подтверждение ставит флаг и останавливает приложение, клавиша без обновления не открывает модалку
  - Acceptance: тесты зелёные под `-race`; клавиша не конфликтует с существующими биндингами

## Phase 6: CI — контрольные суммы релиза
- [ ] Task: `.github/workflows/release.yml` — шаг генерации `checksums.txt` (`sha256sum finam-terminal-* > checksums.txt` в `dist/`) перед `softprops/action-gh-release`; проверка синтаксисом (`actionlint`/`yq`) и вручную по логике job
  - Acceptance: `files: dist/*` включает `checksums.txt`; формат строк совпадает с парсером из фазы 3

## Phase 7: Документация и верификация
- [ ] Task: Полная авто-проверка — `go build ./...`, `go vet ./...`, `go test ./...`, `go test -tags=integration ./api/...`, `CGO_ENABLED=1 go test -race` (обе сюиты), `make lint`, покрытие пакета `updater` > 80% (`make coverage`)
  - Acceptance: всё зелёное, покрытие подтверждено выводом `go tool cover -func`
- [ ] Task: Руководство пользователя — новый `docs/user_manual/updates.md` (автопроверка раз в сутки, значок `⚡`, диалог при запуске, клавиша `U`, что происходит при обновлении, файл `~/.finam-cli/update.json`, ручное обновление install-скриптом, почему в dev/Docker проверки нет) + ссылки в `index.md`, упоминание индикатора в `interface-overview.md` и клавиши `U` в списках горячих клавиш
  - Acceptance: раздел в стиле существующего мануала, содержание обновлено
- [ ] Task: Проектная документация — `CLAUDE.md` (пакет `updater/`, файл состояния, поток обновления), `CHANGELOG.md` (новая запись), `README.md` (раздел «Обновление»)
  - Acceptance: описания соответствуют реализации
- [ ] Task: (Ручной смоук) Собрать `-ldflags` со заниженной версией (например, `v0.10.0`): при запуске появляется диалог с обеими версиями; «Продолжить» → TUI с `⚡`; клавиша `U` → модалка; «Обновить и перезапустить» → скачивание, замена, перезапуск, в шапке актуальная версия; отдельно проверить запуск без сети (запуск не тормозит, `[WARN]` в логе) и dev-сборку (тишина)
  - Acceptance: пользователь подтверждает поведение на своей платформе
