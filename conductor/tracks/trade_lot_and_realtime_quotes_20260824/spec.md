# Spec: Finam API 2.18.1/2.19.0 — корректный лот заявок (trade_lot_size) и realtime-котировки (SubscribeQuote)

## Problem
SDK запинен на псевдо-версии `v0.0.0-20260707135128-ee013ef14834` (07.07.2026, эпоха 2.17.0). После этого Finam выпустил 2.18.0 (20.07), 2.18.1 (04.08) и 2.19.0 (13.08) с двумя важными для трейдера нововведениями, которые терминал не использует:

- **2.18.1 — `GetAssetParamsResponse.trade_lot_size`**: «Размер лота инструмента для торговых операций. Если поле равно 0 — значение отсутствует». API прямо предписывает использовать это поле при формировании ордера. Терминал же умножает лоты на `lot_size` из `GetAsset` (`PlaceOrder`, `api/client.go:552-563`); при расхождении значений трейдер отправляет заявку не на тот объём.
- **2.19.0 — `Quote.is_data_snapshot` + потоковые данные**: терминал опрашивает котировки раз в 5 секунд по одному unary `LastQuote` на каждую позицию (N+1 RPC на счёт каждый тик, `api/client.go:848-880`, `ui/data.go:349-388`). Цены запаздывают до 5 с, лимиты API расходуются впустую. SDK давно содержит `SubscribeQuote`, а новый флаг снепшота позволяет корректно инициализировать состояние при (пере)подключении.

Серверные улучшения релизов (GetAsset для архивных бумаг, понятные ошибки авторизации, фоллбек `symbol` в Trades) достаются без кода. Состав индекса (`GetConstituents.weight`, 2.18.0) — вне скоупа по решению пользователя.

## Solution
Два независимых пласта поверх обновлённого SDK:

1. **Корректный лот заявок**: двухуровневый кэш лотов в `api.Client` — приоритетный `tradeLotCache` (из `GetAssetParams.trade_lot_size`) поверх существующего `assetLotCache` (из `GetAsset.lot_size`). `GetLotSize` остаётся единственной точкой чтения для модалки заявки, close-модалки, обогащения позиций, `PlaceOrder` и `PlaceSLTPOrder` — согласованность «модалка показывает ровно то, чем умножит PlaceOrder» структурная. Плюс строка `Trade Lot` в секции Trading профиля.
2. **Realtime-котировки**: стрим-менеджер в `api.Client` (callback-форма, по образцу JWT-renewal стрима: reconnect с backoff 1s→30s, декларативный набор символов, переподписка при его смене), обработка `is_data_snapshot` (снепшот = полная замена состояния, инкремент = мерж non-nil полей — все ценовые поля SDK `Quote` суть `*decimal.Decimal`), коалесирующая доставка в UI через один `QueueUpdateDraw`, мгновенный фоллбек на существующий поллинг при падении стрима. Стрим покрывает позиции активного счёта + символ открытого профиля; неактивные счета и бары остаются на поллинге.

Обновление SDK низкорисковое и не требует сети: целевая версия `v0.0.0-20260813094515-ac0abddcd07d` уже в кэше модулей (D:\go\pkg\mod), набор из 20 .go-файлов идентичен, все `_FullMethodName` совпадают — только аддитивные поля сообщений.

## Requirements

### 1. Обновление SDK (2.18.1/2.19.0)
- `go.mod`: `github.com/FinamWeb/finam-trade-api/go v0.0.0-20260813094515-ac0abddcd07d`; `go mod tidy`.
- Гейт: обе тестовые сюиты зелёные ДО любых правок кода (апгрейд аддитивен).

### 2. trade_lot_size — модель и маппинг
- `models.AssetParams` + `TradeLotSize int64` (0 = отсутствует; намеренное отступление от строковой конвенции — семантика «0 = нет» нужна ниже по потоку).
- `api/client.go` `GetAssetParams` (:1503-1567): маппинг `resp.TradeLotSize` + запись в trade-кэш (загрузка профиля бесплатно греет кэш).
- `ui/profile.go` секция Trading (:193-212): строка `Trade Lot` при `TradeLotSize > 0`.
- `api/testserver/testdata.go` `DefaultAssetParams()`: `TradeLotSize: 5` — намеренно ≠ `LotSize: "10"` из `DefaultAssetInfo`, чтобы приоритет проверялся end-to-end.

### 3. trade_lot_size — двухуровневый кэш лотов
- `Client.tradeLotCache map[string]float64`. Семантика: ключ присутствует = «проверено»; значение 0 = «проверено, у API нет» (негативный кэш — без него символ без trade-лота дёргал бы `GetAssetParams` каждый 5-секундный тик).
- `storeTradeLotSize(fullSymbol, v)`: под `assetMutex.Lock`, пишет по fullSymbol и тикеру (часть до `@`), включая 0.
- `fetchLotSize` (:406-443): вторая независимая ветка — при отсутствии записи в trade-кэше вызвать `GetAssetParams`; ошибка → только `logGRPCError`, без негативной записи (повтор при следующем промахе, симметрично ветке `GetAsset`).
- `getFullSymbol` (:333-403): условие «промаха» в обеих ветках (:337, :346-349) расширить требованием записи в trade-кэше.
- `GetLotSize` (:446-461): рефакторинг в `RLock` + неэкспортируемый `lotSizeLocked(ticker)` с приоритетом: trade[ticker]>0 → trade[mic]>0 → asset[ticker] → asset[mic].
- `GetAccountDetails` (:815-821): внутри уже удерживаемого RLock заменить прямое чтение `assetLotCache` на `lotSizeLocked` (RWMutex нереентерабелен — `GetLotSize` оттуда звать нельзя).
- `PlaceOrder`/`PlaceSLTPOrder` — без изменений: уже ходят через `GetLotSize` (:553-556, :635-638).

### 4. trade_lot_size — прогрев в UI
- Новый `Client.EnsureLotSize(accountID, symbol string) float64` (= `getFullSymbol` + `GetLotSize`) + добавить в интерфейс `ui/app.go APIClient` и `ui/mock_client_test.go`.
- Хелпер `App.warmLotSizeAsync(accountID, symbol, apply func(lot float64))`: горутина → `EnsureLotSize` → `QueueUpdateDraw(apply)`.
- `OpenOrderModalWithTicker` (ui/app.go:192-222): `SetLotSize(GetLotSize(...))` ПОСЛЕ синхронного `GetSnapshots` (он греет кэш побочно), затем `warmLotSizeAsync`; `apply` перепроверяет, что модалка открыта на том же инструменте.
- `OpenOrderModal` (:690-693): + `warmLotSizeAsync` (позиции обычно тёплые — no-op).
- `ShowModifyOrderModal` (:493-505): + `warmLotSizeAsync`; пересчёт предзаполненных лотов, только если пользователь не менял поле.
- `OpenCloseModal` — без изменений (`Position.LotSize` уже приоритетный через п.3).

### 5. Стриминг — расширение testserver
- `MockMarketDataServer.SubscribeQuote` по образцу `JwtRenewalQueue`: `QuoteStreamItem{Quotes []*marketdata.Quote; StreamErr *marketdata.StreamError; Err error}`, поля `QuoteStreamQueue chan` (cap 100), `QuoteStreamCalled chan struct{}`, `QuoteStreamCallCount atomic.Int64`, `LastQuoteStreamSymbols atomic.Value`, `SubscribeQuoteOverride`. Цикл `select` на `stream.Context().Done()` / очередь.
- `testdata.go`: `DefaultStreamQuote(symbol string, snapshot bool)` — полные поля при снепшоте, только `Last`+`Timestamp` иначе.

### 6. Стриминг — чистое ядро (новый файл `api/quote_stream.go`)
- Выделить `quoteToModel(*marketdata.Quote) *models.Quote` из тела `GetQuotes` (:872-887) — чистый рефакторинг, существующие тесты — регрессионная сетка.
- `mergeQuote(prev, next)`: `next.IsDataSnapshot || prev == nil` → вернуть `next` целиком; иначе shallow-копия `prev` с перезаписью только non-nil полей — явный список: `Ask, AskSize, Bid, BidSize, Last, LastSize, Volume, Turnover, Open, High, Low, Close, Change, OpenInterest` (все `*decimal.Decimal`) + `Timestamp`. Без рефлексии.

### 7. Стриминг — менеджер в api.Client
- Публично (+ в `APIClient`): `StartQuoteStream(onQuote func(models.Quote), onState func(up bool))` (идемпотентно), `SetQuoteSymbols(symbols []string)` (декларативный желаемый набор; безопасно с UI-потока, не блокирует).
- Внутренности: `quoteMu`, `quoteSymbols` (нормализованный: фильтр по `@`, дедуп, сортировка — чистая `normalizeSymbols`), `quoteSubCancel`/`quoteCancel`, `quoteWake chan struct{}` (cap 1), `lastStreamQuotes map[string]*marketdata.Quote`, `runQuoteStream(ctx)`, `getStreamContext(parent)` — БЕЗ таймаута (`getContext()` несёт 30 с — для стрима непригоден), с текущим Authorization-токеном на каждую (пере)подписку.
- Цикл: зеркалит `subscribeJwtRenewal` (:149-207), переиспользует `sleepOrDone`/`nextBackoff` (1s→30s). Пустой набор → ждать `quoteWake`/ctx. Up-переход только на первом успешном `Recv` (gRPC-стримы открываются лениво — живость доказывают данные). `resp.Error != nil` → `[WARN]`, стрим живёт. Отмена `subCtx` при живом ctx жизни = добровольная переподписка (смена символов): без down-события и backoff, `lastStreamQuotes` подрезать до нового набора. `onState` дедуплицирован (только на смене состояния).
- `SetQuoteSymbols`: `slices.Equal` со старым → no-op; иначе сохранить, `quoteSubCancel()`, неблокирующий send в `quoteWake`.
- `Close()` (:128-136): + `quoteCancel`.

### 8. Стриминг — потребление в UI (новый файл `ui/stream.go`)
- Поля `App`: `quoteInboxMu`, `quoteInbox map[string]*models.Quote`, `quoteFlushQueued bool`, `streamLive atomic.Bool`.
- `onStreamQuote` (горутина api): при закрытом `stopChan` — дроп (защита от блокировки `QueueUpdateDraw` после `Stop()`); upsert в inbox; один запланированный `QueueUpdateDraw(flushQuoteInbox)` за раз (коалесинг: 50 тиков/с → максимум один draw в очереди).
- `flushQuoteInbox` (event-loop): слить inbox; под `dataMutex` upsert в `a.quotes[activeAccountID]`; перерисовать таблицу позиций (если видима) и Quote-секцию профиля (если открыт на слитом символе).
- `computeStreamSymbols(positions, profileOpen, profileSymbol)` — чистая: символы позиций активного счёта ∪ символ профиля, фильтр `@`, дедуп, сортировка.
- `recomputeStreamSymbols()` → `SetQuoteSymbols`; вызовы: `loadDataAsync` (в существующем QueueUpdateDraw, только активный счёт), `switchAccount` (ui/input.go:31-52), `OpenProfileForSymbol`/`CloseProfile` (ui/app.go:801-827). Старт: `Run()` после `go a.backgroundRefresh()`.
- `APIClient` + мок: `StartQuoteStream`, `SetQuoteSymbols` (no-op дефолты + захват символов для ассертов).

### 9. Стриминг — фоллбек (мгновенное переключение владения, без dead-man-порога)
- `loadDataAsync`: `skipQuotes := streamLive && accountID == activeAccountID`; при пропуске НЕ звать `GetQuotes` и **сохранять** существующую `a.quotes[accountID]` — текущая замена пустой картой (`ui/data.go:68-72`) затирала бы стрим-данные каждые 5 с. Это самая тонкая обязательная правка.
- Неактивные счета — поллинг как сегодня. `refreshProfileQuoteAndBars` (ui/data.go:297-346): при `streamLive` пропустить котировку, бары продолжают поллиться (SubscribeBars вне скоупа).
- Down: следующий 5-секундный тик возобновляет поллинг активного счёта. Up: серверный снепшот переписывает все символы. Флаппинг безвреден (up только после реальных данных).

### Не включаем (non-goals)
- `GetConstituents`/`weight` (состав индекса) — отклонено пользователем.
- `SubscribeAccount`, `SubscribeOrderBook`, `SubscribeBars`, `SubscribeOrders`/`SubscribeOrderTrade` — позиции/балансы/P&L и бары остаются на поллинге; стрим-менеджер строится расширяемым, но подключение остальных стримов — будущие треки.
- `ReportsService`, `OptionsChain`, `AllAssets` — вне скоупа.
- Изменение 5-секундного `refreshPeriod` и логики History/Orders.

## Acceptance Criteria
- [ ] go.mod на `v0.0.0-20260813094515-ac0abddcd07d`; обе сюиты зелёные сразу после бампа.
- [ ] `models.AssetParams.TradeLotSize` маппится из ответа; профиль показывает `Trade Lot` при >0 и не показывает при 0.
- [ ] `GetLotSize` предпочитает trade-лот (фикстуры 5 vs 10 → 5), фоллбек на asset-лот при 0/отсутствии; негативный кэш: повторный промах без RPC (счётчики мока).
- [ ] `PlaceOrder`/`PlaceSLTPOrder`: 2 лота × trade-лот 5 → `Quantity "10"` в записанном `MockOrdersServer` запросе.
- [ ] Модалка заявки показывает `Lots (size - 5)` — то же значение, которым умножит `PlaceOrder`; все три пути открытия (тикер/позиции/изменение заявки) прогревают кэш.
- [ ] `GetAccountDetails` кладёт приоритетный лот в `Position.LotSize` без вложенного RLock (race-детектор чист).
- [ ] `MockMarketDataServer.SubscribeQuote` управляем очередью; `mergeQuote`: снепшот заменяет всё, инкремент сохраняет прежний Bid при обновлении Last, первое сообщение без флага не теряется.
- [ ] Интеграционно: доставка снепшота (+`onState(true)` только после первого Recv), мерж инкремента, переподписка при смене символов (CallCount=2, без down-события), reconnect после обрыва (down → повторная подписка ≈1 с → up), пустой набор не подписывается, `Close()` останавливает менеджер, in-band `StreamError` не рвёт стрим.
- [ ] При живом стриме поллинг НЕ затирает `a.quotes` активного счёта; при down следующий тик (≤5 с) возобновляет поллинг.
- [ ] `go build ./...`, `go vet ./...`, `go test ./...`, `go test -tags=integration ./api/...`, `CGO_ENABLED=1 go test -race` (обе сюиты), `make lint` — чисто.
- [ ] Документация: CLAUDE.md (Realtime Quotes, TradeLotSize, стрим-мок, список интеграционных файлов), CHANGELOG.md, README.md; руководство `docs/user_manual/`: profile.md (строка Trade Lot), trading.md (источник лота в модалке), positions.md (realtime-обновление котировок и фоллбек).
- [ ] (Ручной смоук) Реальный ключ: `Trade Lot` в профиле; лог `[INFO] Quote stream` up; колонка Value обновляется чаще 5 с; при обрыве сети — down-лог и продолжение обновлений поллингом, после восстановления стрим возвращается.

## Edge Cases
- `trade_lot_size == 0` = «значения нет» → фоллбек на asset-лот; негативный кэш обязателен (иначе шторм `GetAssetParams` каждый тик поллинга).
- Ошибка `GetAssetParams` в `fetchLotSize` → без негативной записи, asset-лот работает, повтор при следующем промахе.
- `GetAssetParams` account-scoped, кэш — по символу: trade-лот считаем свойством инструмента, кэшируем под первым коснувшимся счётом (задокументировать).
- Гонка модалки: сабмит в окне между записью кэша и перерисовкой метки — один хоп event-loop; оба пути читают один кэш (опциональное ужесточение — перечитать лот на сабмите — не в скоупе).
- Все ценовые поля `Quote` — `*decimal.Decimal`: «мерж непустых» = «мерж non-nil указателей»; nil-поля рендерятся `formatDecimal` → "N/A".
- `IsDataSnapshot` сидит на внутреннем `Quote` (тег 17), не на `SubscribeQuoteResponse`; ответ батчевый (`Quote []*Quote`) — обрабатывать срезом.
- Успешный `SubscribeQuote` без Recv ничего не доказывает (ленивое открытие) — up только после первого сообщения.
- `QueueUpdateDraw` после `Stop()` блокирует горутину навсегда → проверка `stopChan` перед постановкой + одиночный flush-флаг.
- bufconn: `TestServer.Stop()` уже неграциозный (из-за долгоживущих стримов) — обязателен и для квот-стрима; тесты синхронизируются каналами с таймаутами, без sleep; reconnect-тест терпит реальный 1-секундный backoff (прецедент — token-refresh тесты).
- Токен на долгоживущем стриме: свежий на каждую (пере)подписку через `getStreamContext`; открытый стрим полагается на то, что сервер не отзывает токен мид-стрим (то же допущение, что у JWT-renewal стрима).

## Dependencies
- SDK `github.com/FinamWeb/finam-trade-api/go@v0.0.0-20260813094515-ac0abddcd07d` — уже в кэше модулей (D:\go\pkg\mod), сеть не нужна.
- Реальный ключ `tapi_sk_...` — только для ручного смоука; авто-тесты через bufconn-мок.
- Релизы: [2.18.0](https://github.com/FinamWeb/finam-trade-api/releases/tag/2.18.0), [2.18.1](https://github.com/FinamWeb/finam-trade-api/releases/tag/2.18.1), [2.19.0](https://github.com/FinamWeb/finam-trade-api/releases/tag/2.19.0)
