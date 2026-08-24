# Plan: Finam API 2.18.1/2.19.0 — корректный лот заявок (trade_lot_size) и realtime-котировки (SubscribeQuote)

## Overview
Обновляем SDK до версии 13.08.2026 (аддитивно, уже в кэше модулей) и задействуем два нововведения: `trade_lot_size` как приоритетный источник лота для торговых ордеров (двухуровневый кэш, единая точка чтения `GetLotSize` — модалка гарантированно показывает то, чем умножит `PlaceOrder`) и `SubscribeQuote` с `is_data_snapshot` вместо 5-секундного N+1 поллинга котировок (стрим-менеджер по образцу JWT-renewal, коалесирующая доставка в UI, мгновенный фоллбек на поллинг). Разработка по TDD; фазы 1–3 (лоты) поставляемы отдельно от фаз 4–6 (стриминг).

## Phase 1: Обновление SDK [checkpoint: 58b2c48]
- [x] Task: Бамп `go.mod` до `v0.0.0-20260813094515-ac0abddcd07d` + `go mod tidy`; прогнать обе сюиты БЕЗ правок кода (cb28fe4)
  - Acceptance: `go build ./...`, `go test ./...`, `go test -tags=integration ./api/...` зелёные; в `go.sum` новая версия; сеть не потребовалась (кэш модулей)

## Phase 2: trade_lot_size — модель, маппинг, кэш, заявки (api) [checkpoint: 2761900]
- [x] Task: (Red) Фикстура `DefaultAssetParams()` +`TradeLotSize: 5` (≠ лоту 10 из `DefaultAssetInfo`) + падающие тесты: маппинг в `GetAssetParams`, приоритет в `GetLotSize` (5 vs 10 → 5), фоллбек при trade=0 → 10 (e57f5fa)
  - Acceptance: тесты компилируются и падают, фиксируя маппинг и приоритет
- [x] Task: (Green) `models.AssetParams.TradeLotSize int64`; маппинг в `GetAssetParams` (api/client.go:1503-1567) + `storeTradeLotSize`; `tradeLotCache` в `Client`/`newClientFromConn`; `lotSizeLocked` + рефакторинг `GetLotSize` (df6b846)
  - Acceptance: тесты предыдущей задачи зелёные; существующие тесты лотов не сломаны
- [x] Task: (Red) Падающие тесты холодного пути и заявок: `fetchLotSize` наполняет оба уровня; повторный вызов — 0 RPC (счётчики мока, негативный кэш); ошибка `GetAssetParams` не мешает asset-лоту и повторяется при следующем промахе; `PlaceOrder`/`PlaceSLTPOrder` умножают на trade-лот (2 лота → Quantity "10") (9cff832)
  - Acceptance: тесты компилируются и падают
- [x] Task: (Green) Вторая ветка в `fetchLotSize` (вызов `GetAssetParams`, запись включая 0); расширенное miss-условие в `getFullSymbol` (:337, :346-349); `GetAccountDetails` → `lotSizeLocked` внутри удерживаемого RLock (47684ee)
  - Acceptance: тесты зелёные; race-детектор чист (нет вложенного RLock)
- [x] Task: `EnsureLotSize(accountID, symbol) float64` + интеграционные тесты (client_cache_integration_test.go): холодный символ прогревается через поток позиций, записанное `MockOrdersServer` количество = лоты × 5, профильный `GetAssetParams` греет кэш (f35a939)
  - Acceptance: интеграционные тесты зелёные end-to-end

## Phase 3: trade_lot_size — UI (профиль + модалки) [checkpoint: 8a8f714]
- [x] Task: (Red→Green) Строка `Trade Lot` в секции Trading профиля (ui/profile.go:193-212) при `TradeLotSize > 0` + unit-тест рендера (есть при 5, нет при 0) (cde2f07)
  - Acceptance: тест рендера зелёный; формат согласован с существующими строками секции
- [x] Task: `warmLotSizeAsync` в ui/app.go + вызовы в `OpenOrderModalWithTicker` (SetLotSize ПОСЛЕ синхронного GetSnapshots), `OpenOrderModal`, `ShowModifyOrderModal` (пересчёт предзаполненных лотов только при нетронутом поле); `APIClient.EnsureLotSize` + мок; unit-тесты (метка `Lots (size - 5)`, счётчик перечитывания) (e9517ac)
  - Acceptance: модалка со всех трёх путей открытия показывает trade-лот; `apply` не трогает модалку, открытую уже на другом инструменте

## Phase 4: Стриминг — testserver + чистое ядро
- [x] Task: `MockMarketDataServer.SubscribeQuote` (QuoteStreamItem{Quotes/StreamErr/Err}, QuoteStreamQueue cap 100, QuoteStreamCalled, QuoteStreamCallCount, LastQuoteStreamSymbols, SubscribeQuoteOverride; select на ctx.Done/очередь) + `DefaultStreamQuote(symbol, snapshot)` в testdata.go (53c5d3d)
  - Acceptance: мок компилируется, сервер стартует; очередь управляет стримом; символы запроса фиксируются
- [x] Task: (Red) Тесты `mergeQuote` (снепшот затирает неприсланное инкрементом поле; инкремент сохраняет Bid при обновлении Last; первое сообщение без флага = полное состояние) и `quoteToModel` (nil-поля → "N/A") (da2ea60)
  - Acceptance: тесты компилируются и падают
- [ ] Task: (Green) Новый `api/quote_stream.go`: выделить `quoteToModel` из `GetQuotes` (:872-887, чистый рефакторинг) + `mergeQuote` (явный список 14 decimal-полей + Timestamp, non-nil перезапись)
  - Acceptance: новые тесты зелёные; существующие тесты `GetQuotes` зелёные (регрессионная сетка рефакторинга)

## Phase 5: Стриминг — менеджер в api.Client
- [ ] Task: (Red) Интеграционные тесты `client_quote_stream_integration_test.go` (7 сценариев: доставка снепшота + up только после первого Recv + символы запроса; мерж инкремента; переподписка при смене символов без down-события, CallCount=2; reconnect после обрыва через ≈1 с backoff; пустой набор не подписывается; Close останавливает; in-band StreamError не рвёт стрим). Синхронизация каналами + таймауты, без sleep
  - Acceptance: тесты компилируются и падают
- [ ] Task: (Green) `StartQuoteStream`/`SetQuoteSymbols`/`runQuoteStream`/`getStreamContext` (без таймаута, свежий токен на (пере)подписку)/`normalizeSymbols`; переиспользовать `sleepOrDone`/`nextBackoff`; `Close()` + `quoteCancel`
  - Acceptance: все 7 интеграционных сценариев зелёные; race-детектор чист
- [ ] Task: Unit-тесты: `normalizeSymbols` (фильтр `@`, дедуп, сортировка) + переподписка через `fakeQuoteStream` (клон `fakeJwtRenewalStream`)
  - Acceptance: unit-тесты зелёные без bufconn

## Phase 6: Стриминг — потребление в UI и фоллбек
- [ ] Task: (Red) Тесты `computeStreamSymbols` (позиции ∪ профиль, фильтр, дедуп), `flushQuoteInbox` (прямой вызов, upsert в quotes), предиката поллинга (матрица стрим up/down × активный/другой счёт)
  - Acceptance: тесты компилируются и падают
- [ ] Task: (Green) `ui/stream.go`: inbox + коалесинг (один QueueUpdateDraw за раз, дроп при закрытом stopChan), `onStreamState` (`streamLive`, `[INFO]`-лог), `computeStreamSymbols`/`recomputeStreamSymbols`; вызовы из `loadDataAsync`/`switchAccount`/`OpenProfileForSymbol`/`CloseProfile`; старт в `Run()`; `APIClient` + мок (`StartQuoteStream`, `SetQuoteSymbols` с захватом)
  - Acceptance: тесты зелёные; захваченные `SetQuoteSymbols` при переключении счёта содержат символы нового счёта
- [ ] Task: Фоллбек: `skipQuotes` в `loadDataAsync` + сохранение `a.quotes[accountID]` вместо замены пустой картой (ui/data.go:68-72); skip котировки в `refreshProfileQuoteAndBars` при живом стриме (бары остаются)
  - Acceptance: при `streamLive` поллинг не затирает котировки активного счёта; при down следующий тик возобновляет поллинг; тест предиката зелёный
- [ ] Task: Ревизия гонок: все мутации `a.quotes` только в QueueUpdateDraw-замыканиях под `dataMutex`; `CGO_ENABLED=1 go test -race` обеих сюит
  - Acceptance: race-детектор чист на unit + integration

## Phase 7: Верификация и финализация
- [ ] Task: Полная авто-проверка — `go build ./...`, `go vet ./...`, `go test ./...`, `go test -tags=integration ./api/...`, `CGO_ENABLED=1 go test -race` (обе сюиты), `make lint`
  - Acceptance: нет ошибок и предупреждений, линтер чистый
- [ ] Task: Документация — CLAUDE.md (новый пункт «Realtime Quotes (SubscribeQuote)», TradeLotSize у GetAssetParams, стрим-мок в testserver, обновлённый список интеграционных файлов), CHANGELOG.md (2.18.1/2.19.0), README.md
  - Acceptance: документация соответствует реализации
- [ ] Task: Руководство пользователя `docs/user_manual/` — profile.md (строка Trade Lot), trading.md (источник лота в модалке заявки), positions.md (realtime-обновление котировок и фоллбек на поллинг), при необходимости index.md
  - Acceptance: разделы в стиле существующего мануала
- [ ] Task: (Ручной смоук) Реальный ключ: `Trade Lot` в профиле; `[INFO] Quote stream` up в логе; колонка Value обновляется чаще 5 с; обрыв сети → down-лог + продолжение поллингом → восстановление стрима; тестовая заявка: объём у брокера = «штукам» модалки
  - Acceptance: пользователь подтверждает поведение
