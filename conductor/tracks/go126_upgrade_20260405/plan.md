# Plan: Обновление Go до 1.26 и внедрение нововведений

## Overview
Обновление проекта finam-terminal с Go 1.24 до Go 1.26. Включает обновление go.mod и зависимостей, автоматическую модернизацию кода через `go fix`, ручное внедрение новых API стандартной библиотеки и языковых конструкций, верификацию через тесты и статический анализ.

## Phase 1: Обновление версии Go и зависимостей
- [x] Task: Обновить `go.mod` — установить `go 1.26.0`, удалить директиву `toolchain` *(b1e7999)*
  - Acceptance: `go.mod` содержит `go 1.26.0`, нет строки `toolchain`
- [x] Task: Обновить все зависимости (`go get -u ./...`, `go mod tidy`) *(219d440)*
  - Acceptance: `go.sum` обновлен, `go build ./...` компилируется без ошибок
- [x] Task: Запустить `go vet ./...` и `go test ./...` — убедиться, что всё работает на Go 1.26 до начала модернизации *(verified)*
  - Acceptance: Нет ошибок компиляции, vet и тесты проходят

## Phase 2: Автоматическая модернизация через `go fix`
- [x] Task: Запустить `go fix -diff ./...` для предварительного просмотра изменений *(reviewed)*
  - Acceptance: Получен список предлагаемых изменений, изучен вручную
  - Файлы: chart.go, components.go, data.go, positions_layout_test.go, utils.go
  - Изменения: rangeint, minmax, minor reformatting
- [x] Task: Применить `go fix ./...` — автоматическая модернизация кода *(94abfda)*
  - Acceptance: Все автоматические фиксы применены
  - Ожидаемые изменения:
    - `rangeint`: for-циклы в `ui/chart.go`, `ui/profile.go`, тестах → `range N`
    - `minmax`: if/else паттерны → `min()`/`max()` в `ui/chart.go` и др.
    - `stringscut`: `strings.Index` → `strings.Cut` в `ui/utils.go`
    - Другие модернизации по результатам анализа
- [x] Task: Запустить `go fix -diff ./...` повторно — убедиться, что все фиксы применены (синергетические фиксы) *(87d205d)*
  - Acceptance: `go fix -diff ./...` не показывает новых изменений
- [x] Task: `go build ./...` и `go test ./...` после автоматической модернизации *(verified)*
  - Acceptance: Всё компилируется и тесты проходят

## Phase 3: Ручное внедрение новых API и языковых конструкций

### 3.1 Языковые изменения
- [x] Task: Внедрить `new(expr)` — найти и заменить паттерны создания указателей на значения *(N/A — no pointer helpers or proto.T() calls found)*
  - Файлы: `api/client.go`, `ui/modal.go` и другие места с protobuf/gRPC кодом
  - Паттерн: `func newT(v T) *T { return &v }` или `v := T; &v` → `new(T(value))`
  - Acceptance: Все helper-функции для создания указателей заменены на `new(expr)`

### 3.2 Стандартная библиотека
- [x] Task: Внедрить `errors.AsType[T]()` вместо `errors.As` где применимо *(N/A — no errors.As calls found)*
  - Поиск: `errors.As(` в кодовой базе
  - Acceptance: Все вызовы `errors.As` заменены на типобезопасный `errors.AsType[T]()`
- [x] Task: Внедрить итераторы `reflect.Type.Fields()` в тестах *(0d14526)*
  - Файлы: `ui/positions_layout_test.go` (строки 40, 51 — индексные циклы по NumField)
  - Паттерн: `for i := 0; i < val.NumField(); i++` → `for field := range typeOfT.Fields()`
  - Acceptance: Индексные циклы по reflect заменены на итераторы
- [x] Task: Проверить и внедрить другие новые API где применимо *(N/A — no applicable patterns found)*
  - `bytes.Buffer.Peek()` — нет паттернов чтения без продвижения
  - `log/slog.NewMultiHandler()` — slog не используется
  - `net/netip.Prefix.Compare()` — netip не используется
  - Acceptance: Все применимые новые API внедрены

## Phase 4: Верификация и финализация
- [x] Task: Полная проверка — `go build ./...`, `go vet ./...`, `go test ./...` *(verified)*
  - Acceptance: Нет ошибок, нет предупреждений, все тесты зелёные
- [x] Task: Финальный `go fix -diff ./...` — подтвердить что модернизация полная *(verified)*
  - Acceptance: Нет предлагаемых изменений
- [x] Task: Обновить CLAUDE.md — отразить новую версию Go (1.26) *(1a64e29)*
  - Acceptance: Документация актуальна
