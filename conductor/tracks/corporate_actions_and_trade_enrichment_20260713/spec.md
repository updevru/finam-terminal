# Spec: Интеграция данных Finam API 2.16/2.17 — НКД сделок, связь заявок и календари корпоративных действий

## Problem
SDK уже обновлён до псевдо-версии `v0.0.0-20260707135128-ee013ef14834` (после релизов 2.16.0 и 2.17.0), но новые данные, которые эти релизы добавили в API, в терминале не показываются:

- **2.16.0** — календари дивидендов и сплитов по акциям (новый сервис `CorporateActionsService`), а также новые поля в методе `Trades`: `accrued_interest` (НКД) и `currency` (валюта цены сделки).
- **2.17.0** — календарь событий по облигациям (купоны/амортизация/оферты) в том же сервисе, а также поле `triggered_order_id` в методах работы с заявками (`GetOrders` и др.).

Сейчас пользователь не видит: НКД по облигационным сделкам («грязную» цену), связь «стоп сработал → породил биржевую заявку», и полностью лишён календарей выплат/событий по бумагам — за этим приходится идти на сторонние сайты.

## Solution
Задействовать уже доступные в SDK данные. Работа делится на два независимых пласта:

1. **Обогащение существующих вызовов** (без нового сервиса): показать НКД и валюту сделки во вкладке History и пометить связь родитель→порождённая заявка во вкладке Orders.
2. **Новый сервис `CorporateActionsService`**, поверхность которого выводится только в профиле инструмента: календарь дивидендов и сплитов для акций, календарь купонов/амортизации/оферт для облигаций.

Обновление SDK, auth и конфигурации **не требуется** — всё уже в запиненной версии. Слой API продолжает конвенцию: SDK-типы (`*decimal.Decimal`, `*date.Date`, `*wrapperspb.*`) заранее форматируются в строки для отображения, UI никогда не касается proto-типов.

## Requirements

### 1. НКД и валюта во вкладке History (2.16.0)
- Добавить в `models.Trade` поля `AccruedInterest string` и `Currency string` (заранее отформатированы).
- В `api/client.go` `GetTradeHistory` заполнять их из `AccountTrade.AccruedInterest` (`*decimal.Decimal`, поле 10, nil-safe) и `AccountTrade.Currency` (`string`, поле 11).
- В `ui/render.go` `updateHistoryTable` добавить **одну** объединённую колонку `НКД` (формат `"12.34 RUB"`, выравнивание вправо), пустую для не-облигационных сделок. Колонки Price/Total не трогаем → всего 7 колонок.

### 2. Связь родитель→порождённая заявка во вкладке Orders (2.17.0)
- Добавить в `models.Order` поле `TriggeredOrderID string`.
- В `api/client.go` `GetActiveOrders` заполнять его из `OrderState.TriggeredOrderId` (`string`, поле 12).
- В `ui/render.go` `updateOrdersTable` кросс-референсом по текущему набору заявок помечать связанную строку(и) значком `↳` и показывать id связанной заявки. Корректная деградация, когда в активном наборе присутствует только одна сторона связи (родитель без дочерней или наоборот).

### 3. Новый сервис CorporateActionsService — проводка и модели (2.16.0/2.17.0)
- Подключить `corporateActionsClient corporateactions.CorporateActionsServiceClient` в `Client`/`newClientFromConn` (поле + `corporateactions.NewCorporateActionsServiceClient(conn)` + импорт).
- Новые модели:
  - `Dividend{ Date, Amount, Currency string; IsFuture bool }`
  - `Split{ Date, OldRatio, NewRatio, NewLot, ConvType string; IsFuture bool }`
  - `BondEvent{ Date, Kind, Value, Currency string; IsFuture bool }` + плоские опциональные строки деталей (купон: `RecordDate`/`StartDate`/`FaceValue`/`Percent`; амортизация: `NewFaceValue`/`InitialFaceValue`/`Percent`; оферта: `Type`/`Price`/`Start`/`End`/`Agent`). `Kind` ∈ {Coupon, Amortization, Offer}.
  - Расширить `InstrumentProfile`: `Dividends []Dividend`, `Splits []Split`, `BondEvents []BondEvent`.
- Три метода API, каждый объединяет past+future в один отсортированный по дате срез с флагами `IsFuture`, переиспользуя `formatDecimal` и `logGRPCError`:
  - `GetDividends(symbol)` — `GetPastDividends` (последние 12 мес, DESC, limit 20) + `GetFutureDividends` (ASC, limit 20)
  - `GetSplits(symbol)` — `GetPastSplits` + `GetFutureSplits`, те же окна
  - `GetBondEvents(symbol)` — `GetPastBondsEvents` + `GetFutureBondsEvents`, те же окна; маппит `oneof` (`CouponDetails`/`AmortizationDetails`/`OfferDetails`) в `Kind` + плоские детали
- Добавить эти 3 метода в интерфейс `ui/app.go APIClient` и в `ui/mock_client_test.go`.

### 4. Календари по акциям в профиле (2.16.0)
- В `ui/data.go` `loadProfileAsync` добавить тип-гейтед догрузку: для акции (не фьючерс/опцион/облигация) параллельно тянуть дивиденды и сплиты; загрузка **не фатальна** (сбой убирает секцию, не роняет профиль — как остальные части профиля).
- В `ui/profile.go` `renderInfoPanel` в блоке, специфичном для типа инструмента, рендерить компактные секции «Dividends» (дата закрытия реестра, размер на акцию, валюта) и «Splits» (дата, коэффициент old→new, новый лот). Ограничение ~3 будущих + 3 прошлых на секцию, сортировка по дате, подсказка `…` при наличии большего числа.

### 5. Календари по облигациям в профиле (2.17.0)
- В `loadProfileAsync` для облигации (`BondFaceValue != ""`) догружать события через `GetBondEvents`; не фатально.
- В `renderInfoPanel` рендерить компактные секции облигационных событий, различая `Kind`:
  - **Купоны** — дата выплаты, ставка `%` (`ValuePercent`), дата фиксации (`RecordDate`).
  - **Амортизация** — дата, процент, новый номинал.
  - **Оферты** — дата (окно `Start`/`End`), тип (PUT/CALL), цена.
  - Те же ограничения 3+3, сортировка, подсказка `…`.

### Определение типа инструмента
- Используем существующий механизм профиля: **облигация** ⇔ `AssetDetails.BondFaceValue != ""`; **опцион** ⇔ `ContractSize != "" && Strike != ""`; **фьючерс** ⇔ `ContractSize != ""`; **акция** ⇔ ни одно из перечисленных. Дивиденды/сплиты — ветка «акция», события по облигации — ветка «облигация». Для фьючерсов/опционов календари не запрашиваются.

### Не включаем (non-goals)
- Не обновляем SDK/auth/config (уже на нужной версии).
- Не трогаем `GetConstituents`/`index_inclusion_date` (в терминале нет представления состава индексов).
- Не добавляем прокрутку/полноэкранный оверлей календаря — только компактные inline-секции в существующей info-панели профиля (~42 колонки).
- НКД показываем только в History; отдельную колонку валюты в History не добавляем (валюта входит в объединённую колонку `НКД`).
- Не показываем календари в списке позиций/поиске — только в профиле.
- `SubscribeOrderTrade`/`SubscribeOrders` (стриминговые) не задействуем — только используемый `GetOrders`.

## Acceptance Criteria
- [ ] `models.Trade` содержит `AccruedInterest`/`Currency`; History показывает колонку `НКД` («12.34 RUB») для облигационной сделки и пусто для акции.
- [ ] `models.Order` содержит `TriggeredOrderID`; Orders помечает связанную заявку значком `↳` с id связанной заявки; при одной стороне связи не падает.
- [ ] `corporateActionsClient` подключён в `newClientFromConn` (в т.ч. через bufconn в тестах).
- [ ] `GetDividends`/`GetSplits`/`GetBondEvents` возвращают объединённый past+future срез, отсортированный по дате, с корректными `IsFuture`; nil-обёртки не роняют маппинг.
- [ ] Профиль **акции** показывает секции Dividends и Splits (capped 3+3, `…` при переполнении); профиль **облигации** — Coupons/Amortization/Offers; для фьючерса/опциона календарей нет.
- [ ] Сбой любого запроса календаря убирает только свою секцию, профиль рендерится.
- [ ] `go build ./...`, `go test ./...`, `go test -tags=integration ./api/...` зелёные; `make lint` без новых замечаний.
- [ ] `MockCorporateActionsServer` реализует все 6 методов; фикстуры `DefaultDividends`/`DefaultSplits`/`DefaultBondEvents` покрывают ветки oneof.
- [ ] Руководство пользователя `docs/user_manual/` обновлено: `history.md` (колонка `НКД`), `orders.md` (маркер `↳`), `profile.md` (секции календарей); при необходимости `index.md`.
- [ ] (Ручной смоук) На реальном ключе: профиль акции показывает дивиденды/сплиты, профиль облигации — купоны/оферты; НКД виден в History по облигационной сделке; `↳` виден в Orders.

## Edge Cases
- Все новые SDK-поля — указатели/обёртки: `*decimal.Decimal`, `*date.Date`, `*wrapperspb.StringValue`, `*wrapperspb.Int32Value`. Перед форматированием обязательна nil-проверка (иначе паника). `AccountTrade.AccruedInterest` для не-облигаций nil → колонка `НКД` пустая.
- `Dividend.Currency` — **plain string**, а `BondEvent.Currency` — `*wrapperspb.StringValue` (разный тип, разный доступ).
- `Future*`-запросы **не принимают** `DateFrom`/`DateTo` (только Symbol/SortDirection/Limit/Offset); `Past*`-запросы принимают. Не передавать даты в future-запросы.
- Вызов календаря не по типу инструмента (например, дивиденды по облигации) или упавший вызов → секция просто отсутствует, без краша и без блокировки остального профиля.
- `↳`: дочерняя (порождённая) заявка может быть уже исполнена и отсутствовать в активном наборе, равно как и родительская. Маркер/референс показываем по той стороне, что есть; отсутствие второй стороны — не ошибка.
- `BondEvent.Type == UNSPECIFIED` (0) или неизвестный oneof — пропускаем/показываем нейтрально, без паники.
- Пустые ответы сервиса (нет дивидендов/купонов) → секция не рендерится (не показываем пустой заголовок).

## Dependencies
- SDK `github.com/FinamWeb/finam-trade-api/go@v0.0.0-20260707135128-ee013ef14834` (уже запинен; содержит пакет `corporateactions` и новые поля на `AccountTrade`/`OrderState`).
- Реальный ключ `tapi_sk_...` — только для ручного смоука; авто-тесты идут через мок `bufconn`.
- Релизы: [2.16.0](https://github.com/FinamWeb/finam-trade-api/releases/tag/Release-2.16.0), [2.17.0](https://github.com/FinamWeb/finam-trade-api/releases/tag/2.17.0)
