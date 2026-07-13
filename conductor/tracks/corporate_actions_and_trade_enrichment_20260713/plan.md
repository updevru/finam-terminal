# Plan: Интеграция данных Finam API 2.16/2.17 — НКД сделок, связь заявок и календари корпоративных действий

## Overview
Задействуем данные, добавленные в Finam API 2.16.0/2.17.0 и уже доступные в запиненном SDK. Две небольшие правки существующих вызовов (НКД+валюта в History, связь заявок в Orders) и новый сервис `CorporateActionsService`, поверхность которого выводится только в профиле инструмента (дивиденды/сплиты для акций, купоны/амортизация/оферты для облигаций). Без обновления SDK/auth/config. Разработка по TDD: тесты API/мока пишутся до реализации; unit-тесты рендера — вместе с рендером. Фазы независимы и тестируются по отдельности.

## Phase 1: Быстрые табличные обогащения (НКД + валюта в History, связь заявок в Orders) [checkpoint: 3d15eef]
- [x] Task: Расширить модели — `models.Trade` (+`AccruedInterest string`, `Currency string`) и `models.Order` (+`TriggeredOrderID string`) (30bb6fe)
  - Acceptance: поля добавлены; `go build ./...` зелёный; существующие тесты не сломаны
- [x] Task: (Red) Расширить фикстуры `DefaultTrades` (НКД+валюта у облигационной сделки) и `DefaultOrders` (`TriggeredOrderId`) + интеграционные ассерты в history/orders тестах — падают (506ef16)
  - Acceptance: тесты компилируются и падают, фиксируя ожидаемый маппинг (НКД/валюта на `Trade`, `TriggeredOrderID` на `Order`)
- [x] Task: (Green) Заполнить `AccruedInterest`/`Currency` в `GetTradeHistory` (nil-safe) и `TriggeredOrderID` в `GetActiveOrders` (13d2dae)
  - Acceptance: интеграционные тесты зелёные; для не-облигаций НКД пустой; nil-обёртки не роняют маппинг
- [x] Task: Колонка `НКД` в `updateHistoryTable` (формат `"12.34 RUB"`, выравнивание вправо, пусто для не-облигаций) + unit-тест рендера (52eea1a)
  - Acceptance: 7 колонок; облигационная строка показывает НКД, акция — пусто; Price/Total не изменены
- [x] Task: Маркер `↳` в `updateOrdersTable` через кросс-референс `TriggeredOrderID` по набору заявок + unit-тест рендера (bf09fbe)
  - Acceptance: связанная строка помечена `↳` с id связанной заявки; при наличии только одной стороны связи рендер не падает

## Phase 2: Фундамент CorporateActionsService (клиент, модели, мок, методы API) [checkpoint: 82c0431]
- [x] Task: Проводка `corporateActionsClient` в `Client`/`newClientFromConn` (поле, `NewCorporateActionsServiceClient(conn)`, импорт `.../v1/corporateactions`) (2c99970)
  - Acceptance: компилируется; клиент инициализируется, в т.ч. в интеграционном пути через bufconn
- [x] Task: Новые модели `Dividend`, `Split`, `BondEvent` (+плоские детали купона/амортизации/оферты) и поля `Dividends`/`Splits`/`BondEvents` в `InstrumentProfile` (009d4b1)
  - Acceptance: типы добавлены, `go build ./...` зелёный; поля-строки готовы к прямому рендеру
- [x] Task: `MockCorporateActionsServer` (`api/testserver/corporateactions_server.go`) + регистрация в `server.go` + фикстуры `DefaultDividends`/`DefaultSplits`/`DefaultBondEvents` (покрыть все ветки oneof и обёртки-указатели) (240f3fd)
  - Acceptance: мок реализует 6 методов и отдаёт фикстуры; зарегистрирован в `NewTestServer`; сервер стартует
- [x] Task: (Red) Интеграционные тесты `client_corporate_actions_integration_test.go` для `GetDividends`/`GetSplits`/`GetBondEvents` — падают (69a70fb)
  - Acceptance: тесты компилируются и падают; проверяют объединение past+future, сортировку по дате, флаги `IsFuture`, маппинг oneof (Coupon/Amortization/Offer) и nil-safe обёрток
- [x] Task: (Green) Реализовать `GetDividends`/`GetSplits`/`GetBondEvents` в `api/client.go` (past 12 мес DESC + future ASC, limit 20; `formatDecimal`; `logGRPCError`; nil-safe) (235664c)
  - Acceptance: интеграционные тесты предыдущей задачи зелёные
- [x] Task: Добавить 3 метода в интерфейс `ui/app.go APIClient` и реализовать в `ui/mock_client_test.go` (9cff229)
  - Acceptance: интерфейс и тестовый двойник обновлены; `go test ./...` компилируется

## Phase 3: Календари по акциям в профиле (дивиденды + сплиты)
- [x] Task: (Red) Расширить `ui/profile_test.go` — для акции ожидаются секции «Dividends» и «Splits» (capped 3+3, подсказка `…`); тест падает (5473cb2)
  - Acceptance: тест фиксирует наличие секций, формат строк (дата, размер/коэффициент, валюта/лот) и ограничение
- [x] Task: Тип-гейтед догрузка дивидендов+сплитов в `loadProfileAsync` для акции (параллельно, не фатально) (2a1069e)
  - Acceptance: для акции профиль содержит `Dividends`/`Splits`; сбой запроса убирает секцию, профиль рендерится; для не-акции запросы не идут
- [ ] Task: (Green) Рендер секций «Dividends»/«Splits» в `renderInfoPanel` (компактно, 3 будущих + 3 прошлых, сортировка по дате, `…`)
  - Acceptance: тест профиля зелёный; для фьючерса/опциона/облигации этих секций нет

## Phase 4: Календари по облигациям в профиле (купоны/амортизация/оферты)
- [ ] Task: (Red) Расширить `ui/profile_test.go` — для облигации ожидаются секции «Coupons»/«Amortization»/«Offers» с деталями; тест падает
  - Acceptance: тест фиксирует купон (ставка `%`, дата фиксации), амортизацию (процент, новый номинал), оферту (PUT/CALL, цена, окно дат)
- [ ] Task: Догрузка событий по облигации через `GetBondEvents` в `loadProfileAsync` (ветка `BondFaceValue != ""`, не фатально)
  - Acceptance: для облигации профиль содержит `BondEvents`; сбой убирает секцию, не роняет профиль
- [ ] Task: (Green) Рендер секций облигационных событий в `renderInfoPanel` по `Kind` (Coupon/Amortization/Offer, компактно, 3+3, сортировка, `…`)
  - Acceptance: тест профиля зелёный; купон показывает ставку и дату фиксации, оферта — тип/цену/даты

## Phase 5: Верификация и финализация
- [ ] Task: Полная авто-проверка — `go build ./...`, `go vet ./...`, `go test ./...`, `go test -tags=integration ./api/...`, `make lint`
  - Acceptance: нет ошибок и предупреждений, все тесты зелёные, линтер чистый
- [ ] Task: Обновить документацию — `CLAUDE.md` (новые секции API: CorporateActions `GetDividends`/`GetSplits`/`GetBondEvents`; поля `AccruedInterest`/`Currency` на `Trade`; `TriggeredOrderID` на `Order`; новый мок-сервер), при необходимости `product.md`/`CHANGELOG.md`
  - Acceptance: документация соответствует реализации
- [ ] Task: Обновить руководство пользователя `docs/user_manual/` — `history.md` (колонка `НКД`), `orders.md` (маркер `↳` связи родитель→порождённая заявка), `profile.md` (секции календарей: дивиденды/сплиты для акций, купоны/амортизация/оферты для облигаций), при необходимости `index.md` (пункт в «Возможности»)
  - Acceptance: разделы описаны в стиле существующего мануала (таблицы колонок/полей, навигационные футеры); скриншоты в `media/` обновляются опционально при ручном смоуке
- [ ] Task: (Ручной смоук) На реальном ключе открыть профиль акции и облигации (увидеть календари), проверить НКД в History по облигационной сделке и `↳` в Orders
  - Acceptance: пользователь подтверждает отображение календарей, НКД и связи заявок
