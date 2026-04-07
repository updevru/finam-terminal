# Plan: Отображение реальной версии программы в UI

## Overview
Создание выделенного пакета `version` с пакетными переменными, инжектируемыми через ldflags, замена хардкода `appVersion = "1.0.0"` в UI на динамическое значение, обновление CI/CD release workflow и Makefile для передачи git-тега в бинарь, добавление fallback-логики через `runtime/debug.ReadBuildInfo()` для dev-сборок без явной инжекции.

## Phase 1: Пакет `version` (TDD)
- [x] Task: Написать failing-тесты для пакета `version` (`version/version_test.go`) (e3c79fc)
  - Acceptance: Тесты покрывают:
    - Дефолтные значения переменных (`Version == "dev"`, `Commit == "unknown"`, `BuildDate == "unknown"`)
    - `String()` для случая инжектированной версии (например, после `Version = "v1.2.3"`)
    - `String()` для dev-сборки — формат `dev` или `dev (sha)` / `dev (sha, dirty)`
    - `Info()` возвращает корректные три значения
    - Таблично-управляемые тесты (table-driven) для разных комбинаций входов
  - Запустить `go test ./version/...` — должны падать (red phase)
- [x] Task: Создать `version/version.go` — package-level vars, `String()`, `Info()`, fallback через `runtime/debug.ReadBuildInfo` (b00918f)
  - Acceptance:
    - Все тесты из предыдущей задачи зелёные
    - `go vet ./version/...` без предупреждений
    - Покрытие ≥ 80% (`go test -cover ./version/...`)
    - Документация (godoc) на экспортированных идентификаторах

## Phase 2: Интеграция в UI
- [x] Task: Удалить `appVersion` из `ui/app.go`, импортировать `finam-terminal/version` в `ui/components.go`, заменить рендер заголовка (ac11d1c)
  - Acceptance:
    - `ui/app.go` не содержит `appVersion`
    - `ui/components.go:createHeader()` использует `version.String()` (формат: `" Finam Terminal " + prefix + version.String() + " "`, где `prefix == "v"` если `Version` не начинается с `v` и не равен `dev`, иначе пусто — избегаем двойного `v`)
    - `go build ./...` компилируется
- [x] Task: Запустить полный набор тестов и убедиться, что UI-тесты не сломались (e36e4c5)
  - Acceptance:
    - `go test ./...` зелёный
    - `go test -tags=integration ./api/...` зелёный

## Phase 3: Build tooling
- [x] Task: Обновить `Makefile` — добавить цель `build` с инжекцией версии из git (a0ed128)
  - Acceptance:
    - Новая цель `build` использует `git describe --tags --always --dirty` для `Version`, `git rev-parse HEAD` для `Commit`, `date -u +%Y-%m-%dT%H:%M:%SZ` для `BuildDate`
    - Команда: `go build -ldflags "-X finam-terminal/version.Version=$(VERSION) -X finam-terminal/version.Commit=$(COMMIT) -X finam-terminal/version.BuildDate=$(BUILD_DATE)" -o finam-terminal main.go`
    - `make build` локально успешно собирает бинарь
- [x] Task: Обновить `.github/workflows/release.yml` — добавить ldflags инжекцию в шаг `Build binary` (6cdc15c)
  - Acceptance:
    - Команда сборки передаёт `-ldflags "-X finam-terminal/version.Version=${{ github.ref_name }} -X finam-terminal/version.Commit=${{ github.sha }} -X finam-terminal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"`
    - YAML валиден (актуально проверить через `yamllint` или `gh workflow view`)
- [x] Task: Manual verification — собрать бинарь локально через `make build` и через `go build`, запустить, проверить заголовок (bd54d3d)
  - Acceptance:
    - `make build` → бинарь показывает версию из `git describe` (например `v1.2.3-5-gabc1234`)
    - `go build main.go` → бинарь показывает `dev (sha)` или `dev`
    - Оба бинаря не падают и не показывают `1.0.0`

## Phase 4: Документация
- [x] Task: Обновить `CLAUDE.md` — секция про пакет `version` и процесс выпуска (3e2d905)
  - Acceptance:
    - Добавлено описание пакета `version/` в Architecture
    - Добавлена инструкция: "Чтобы выпустить новую версию — создайте git-тег `vX.Y.Z` и push, CI соберёт релиз с этой версией"
    - Указано, как локально собрать с версией (`make build`)
