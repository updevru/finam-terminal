# Plan: Обновление finam-trade-api SDK и переход на новый формат API-ключей

## Overview
Обновление зависимости `github.com/FinamWeb/finam-trade-api/go` с уровня 2.14.0 до последней версии (коммит 07.07.2026, после 2.17.0), приведение онбординга и документации к новому короткому формату ключей `tapi_sk_...` (релиз 2.15.0), передача `source_app_id` при авторизации и переход с таймерного обновления токена на стриминг `SubscribeJwtRenewal`. Логика приёма ключа не меняется — обратная совместимость со старыми длинными токенами сохраняется. Разработка ведётся по TDD: тесты пишутся до реализации.

## Phase 1: Обновление SDK [checkpoint: 4e743aa]
- [x] Task: Обновить зависимость до `v0.0.0-20260707135128-ee013ef14834` (`go get github.com/FinamWeb/finam-trade-api/go@ee013ef148348c91b6b7d19d5f4008f6c1b2c65b`) и выполнить `go mod tidy` (74d6f7d)
  - Acceptance: `go.mod`/`go.sum` содержат целевую псевдо-версию, `go mod tidy` без изменений после повторного запуска
- [x] Task: Базовая верификация после бампа (до изменений auth) — `go build ./...`, `go test ./...`, `go test -tags=integration ./api/...`, `make lint` (verified, no code change)
  - Acceptance: сборка зелёная, все тесты проходят, линтер без новых замечаний; при поломке (маловероятно, изменения аддитивные) — точечно починить

## Phase 2: Новый формат ключей — онбординг и документация
- [x] Task: Обновить онбординг в `ui/setup.go` — ссылка на портал `https://api.finam.ru/tokens/`, упоминание короткого ключа `tapi_sk_...` и Legacy-статуса старых ключей (686f2ba)
  - Acceptance: тексты/ссылки актуальны; логика ввода и поведенческая валидация (`GetAccounts()`) не изменены
- [x] Task: Обновить `.env.example` и `README.md` под новый формат ключей (ссылка `api.finam.ru/tokens`, заметка о `tapi_sk_`/Legacy) (930c1fa)
  - Acceptance: оба файла отражают новый формат; переменная `FINAM_API_TOKEN` сохранена
- [x] Task: Добавить запись в `CHANGELOG.md` (апдейт SDK + поддержка нового формата ключей + переход на `SubscribeJwtRenewal`) (6d01382)
  - Acceptance: `CHANGELOG.md` содержит новую датированную запись

## Phase 3: source_app_id и переход на SubscribeJwtRenewal (TDD)
- [ ] Task: Реализовать стриминговый `SubscribeJwtRenewal` в `MockAuthServer` (`api/testserver/`) + фикстуры для управления потоком JWT
  - Acceptance: мок принимает `SubscribeJwtRenewalRequest`, отдаёт поток `SubscribeJwtRenewalResponse{Token}`, есть способ инжектировать очередные токены/ошибки и наблюдать `SourceAppId`
- [ ] Task: (Red) Переписать `client_token_refresh_integration_test.go` под стримовую модель — тесты определяют ожидаемое поведение и падают
  - Acceptance: тесты компилируются и падают (реализации ещё нет): проверяют обновление `c.token` из стрима, реконнект после обрыва, остановку по `Close()`
- [ ] Task: Ввести константу `source_app_id` (напр. `"finam-terminal"`) и передавать её в `auth.AuthRequest.SourceAppId` и `auth.SubscribeJwtRenewalRequest.SourceAppId`
  - Acceptance: оба запроса содержат непустой `SourceAppId`; unit-тест на `authenticate()` подтверждает передачу
- [ ] Task: (Green) Заменить таймерный `startTokenRefresh` на цикл `SubscribeJwtRenewal` с реконнектом/бэкоффом и остановкой по `ctx`/`Close()`
  - Acceptance: обновление токена идёт через стрим; таймерная логика удалена; интеграционные тесты из предыдущей задачи зелёные
- [ ] Task: Unit-тесты на устойчивость — реконнект с бэкоффом при обрыве стрима и graceful stop при `Close()` (без ошибок в логах)
  - Acceptance: ветви ошибок и остановки покрыты тестами; `go test ./...` зелёный

## Phase 4: Верификация и финализация
- [ ] Task: Полная авто-проверка — `go build ./...`, `go vet ./...`, `go test ./...`, `go test -tags=integration ./api/...`, `make lint`
  - Acceptance: нет ошибок и предупреждений, все тесты зелёные, линтер чистый
- [ ] Task: Обновить `CLAUDE.md` — описание auth-потока (стрим `SubscribeJwtRenewal` вместо таймера, `source_app_id`) и версию Trade API/SDK
  - Acceptance: документация соответствует реализации
- [ ] Task: (Ручной смоук, вариант B) Запуск приложения с реальным ключом `tapi_sk_...` — подтвердить вход, загрузку счетов и приход новых JWT по стриму; проверить, что старый длинный ключ тоже работает
  - Acceptance: пользователь подтверждает успешный вход новым ключом и работу обновления токена; обратная совместимость сохранена
