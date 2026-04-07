# Spec: Отображение реальной версии программы в UI

## Problem
В заголовке TUI всегда отображается `Finam Terminal v1.0.0`, потому что `appVersion` захардкожен константой в `ui/app.go:16`. Это вводит пользователя в заблуждение — отображаемая версия не соответствует ни реальной версии бинаря, ни git-тегу, по которому он был собран. CI/CD workflow `.github/workflows/release.yml` собирает релизы по push-у тегов `v*`, но текущая команда `go build -v -o ... main.go` не передаёт значение тега в бинарь — никакой инжекции не происходит.

## Solution
Заменить хардкод на динамическую версию, инжектируемую в бинарь во время сборки через `-ldflags "-X ..."`. Версия берётся из git-тега в CI/CD pipeline (`${{ github.ref_name }}`). Для локальных и dev-сборок без явной инжекции использовать fallback через `runtime/debug.ReadBuildInfo()` — оттуда читается `vcs.revision` (короткий SHA) и `vcs.modified` (dirty flag). Создать выделенный пакет `version` с пакетными переменными `Version`, `Commit`, `BuildDate` и функцией `String()`, чтобы:
- инжекция через ldflags была чистой и переиспользуемой;
- UI-код не зависел от деталей source-of-truth версии;
- появилась естественная точка расширения для будущих метаданных билда.

## Requirements

### Пакет `version`
- Создать `version/version.go` с пакетными переменными (тип `string`, не `const` — иначе ldflags `-X` не сработает):
  - `Version = "dev"` — версия (например, `v1.2.3` или `dev`)
  - `Commit = "unknown"` — git commit SHA
  - `BuildDate = "unknown"` — дата сборки
- Функция `String() string` — форматирует строку для отображения:
  - Если `Version != "dev"`: возвращает `Version` (например, `v1.2.3`)
  - Если `Version == "dev"` и есть VCS info: возвращает `dev (a1b2c3d)` или `dev (a1b2c3d, dirty)`
  - Если ничего нет: возвращает `dev`
- Функция `Info() (version, commit, date string)` — возврат всех полей (полезно для CLI/диагностики)
- При инициализации (`init()` или ленивой логике в `String()`): если `Version == "dev"` и `Commit == "unknown"`, попытаться прочитать `runtime/debug.ReadBuildInfo()` и заполнить `Commit` из `vcs.revision`, `BuildDate` из `vcs.time`, учесть `vcs.modified` для пометки `dirty`

### UI integration
- Удалить константу `appVersion` из `ui/app.go`
- В `ui/components.go:199` заменить `fmt.Sprintf(" Finam Terminal v%s ", appVersion)` на использование `version.String()` (формат, например, `" Finam Terminal " + version.String() + " "` или аналогично)
- Импортировать `finam-terminal/version` в `ui/components.go`
- При `Version == "dev"` префикс `v` не добавлять — `dev (sha)` уже понятно
- При `Version == "v1.2.3"` показать как есть

### Build tooling
- Обновить `.github/workflows/release.yml` job `build`:
  - Собирать с `-ldflags "-X finam-terminal/version.Version=${{ github.ref_name }} -X finam-terminal/version.Commit=${{ github.sha }} -X finam-terminal/version.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"`
  - Также передать `-trimpath` для воспроизводимости (опционально)
- Обновить `Makefile`:
  - Добавить цель `build`, которая собирает с инжекцией версии из `git describe --tags --always --dirty` и `git rev-parse HEAD`
  - Локальная сборка `go build` без `make` должна оставаться функциональной (fallback на `dev` через `debug.ReadBuildInfo`)

### Тесты
- Unit-тесты для пакета `version`:
  - Дефолтные значения (`Version = "dev"`, `Commit = "unknown"`)
  - `String()` для случая когда `Version` установлен (имитация ldflags-инжекции)
  - `String()` для dev-сборки с моком VCS info (если возможно через `debug.ReadBuildInfo`)
  - `Info()` возвращает корректный кортеж
- Существующие UI тесты должны продолжать проходить после рефакторинга

## Acceptance Criteria
- [ ] Создан пакет `version` с переменными `Version`, `Commit`, `BuildDate` и функциями `String()`, `Info()`
- [ ] Хардкод `appVersion = "1.0.0"` удалён из `ui/app.go`
- [ ] `ui/components.go` использует `version.String()` для рендера заголовка
- [ ] Локальный `go build main.go` собирает бинарь, который показывает `dev (commit-sha)` или просто `dev` в заголовке
- [ ] `make build` собирает бинарь с версией из `git describe`
- [ ] CI release workflow собирает релиз с версией, равной git-тегу (например `v1.2.3`)
- [ ] Запуск релизного бинаря показывает `Finam Terminal v1.2.3` в заголовке (или эквивалентный формат без двойного `v`)
- [ ] Покрытие unit-тестами пакета `version` ≥ 80%
- [ ] `go test ./...` и `go test -tags=integration ./api/...` зелёные
- [ ] `go vet ./...` и `golangci-lint run ./...` без предупреждений
- [ ] CLAUDE.md обновлён: описан пакет `version`, способ инжекции и порядок выпуска новой версии через git-теги

## Edge Cases
- **Сборка из tarball без .git**: `debug.ReadBuildInfo` не вернёт `vcs.*` settings → fallback показывает `dev`
- **Сборка с грязным working tree** (`make build` локально): `git describe --dirty` добавит суффикс `-dirty`, а `vcs.modified=true` в `debug.ReadBuildInfo` — нужно правильно сшивать оба источника
- **Сборка в CI на не-tag коммите** (например, push в main через release.yml — сейчас не происходит, но возможно в будущем): должна корректно деградировать до commit SHA
- **Длинные tag имена** (`v1.2.3-rc1`, `v2.0.0-beta.1+build.42`): должны корректно отображаться в заголовке без обрезки или поломки tview-рендера
- **Двойной `v`**: tag уже содержит `v` (`v1.2.3`), не нужно добавлять ещё один префикс — формат должен быть `Finam Terminal v1.2.3`, а не `Finam Terminal vv1.2.3`
- **Конфликт `const` vs `var`**: `appVersion` сейчас `const` — ldflags не может перезаписать константу, поэтому переменная в `version` пакете обязательно должна быть `var`

## Out of Scope
- CLI флаг `--version` для `main.go` (можно добавить отдельным треком, если будет нужно)
- Embedding версии в Docker image labels (release.yml уже использует `docker/metadata-action`, который покрывает теги образа)
- Автоматическое создание git-тегов / релизных нот

## Dependencies
- `runtime/debug` (стандартная библиотека Go)
- Никаких новых внешних зависимостей
