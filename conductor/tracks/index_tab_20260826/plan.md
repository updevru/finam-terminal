# Plan: Вкладка «Индекс» — витрина бумаг Индекса МосБиржи

## Overview
Добавляем четвёртую вкладку Index: состав Индекса МосБиржи из `AssetsService.GetConstituents("IMOEX@RTSX")` (метод уже в SDK из go.mod; символ индекса подтверждён разведкой на реальном API, кеш на сессию — 1–2 запроса в сутки), котировки и изменение за сессию — через существующий стрим `SubscribeQuote` (46 символов компонентов в подписке только при активной вкладке; `Quote.change` приходит в каждой котировке). Fallback при упавшем стриме — строго ограниченный батч `LastQuote` (лимит 200/мин на метод подтверждён документацией). `Enter` → профиль, `A` → стандартная модалка заявки. Разработка по TDD.

## Phase 1: Разведка GetConstituents на реальном API
- [x] Task: Одноразовый скрипт с реальным токеном (авторизация → bulk `Assets` → пробы `GetConstituents` для кандидатов и примеров из документации); результаты зафиксированы в spec.md, временный код удалён. Итог: `IMOEX@RTSX` — 46 компонентов (символы `SBER@MISX`, русские имена, сектора, веса); `IMOEX@MISX`/`MOEXBC@MISX` → NotFound; NDX/SPX работают, но вне скоупа; сам индекс в bulk-активах отсутствует → список индексов зашивается константой. (выполнено при создании трека 2026-08-26, без коммита)
  - Acceptance: spec.md обновлён фактами с реального API; выбран путь — состав из API, символ индекса константой

## Phase 2: API-слой — состав индекса и Quote.Change [checkpoint: cc78628]
- [x] Task: (Red→Green) `models.Quote.Change` + маппинг в `quoteToModel` (api/quote_stream.go:20-37); фикстуры `DefaultQuote()`/`DefaultStreamQuote()` в api/testserver/testdata.go дополнить `Change`; юнит-тесты: change маппится, nil → "N/A", инкремент стрима сохраняет Change (регресс `mergeQuote`) (e9db6b7)
  - Acceptance: тесты зелёные; существующие тесты quoteToModel/mergeQuote не сломаны
- [x] Task: (Red) `MockAssetsServer.GetConstituents` (фикстура `DefaultConstituents()` — 2 страницы с `next_cursor`, поля symbol/name/sector/weight; `GetConstituentsError` для инъекции ошибок; счётчик вызовов) + падающие интеграционные тесты `client_index_integration_test.go`: полная выборка через пагинацию с сохранением порядка, маппинг weight/name/ticker (символ → тикер до `@`), повторный вызов в TTL → 0 RPC (счётчик), ошибка рефетча → прежний состав (stale-on-error), ошибка первой загрузки → error, пустой ответ → error и кеш не отравлен, защитный предел страниц (6153c60)
  - Acceptance: тесты компилируются и падают
- [ ] Task: (Green) `models.IndexConstituent{Symbol, Ticker, Name, Sector, Weight}`; `Client.GetIndexConstituents(indexSymbol string) ([]models.IndexConstituent, error)` — цикл пагинации (предел 10 страниц + [WARN]), кеш в памяти по символу индекса (TTL 24ч, stale-on-error), `logGRPCError`; метод в `ui.APIClient` + мок ui-тестов (40cbdd1)
  - Acceptance: тесты Phase 2 зелёные end-to-end (bufconn)

## Phase 3: Четвёртая вкладка UI
- [x] Task: (Red) Тесты вкладки: цикл `nextTab`/`prevTab` проходит 4 вкладки (Positions→History→Orders→Index→Positions), `SetTab(TabIndex)` переключает страницу "index", заголовок содержит " Index ", фокус уходит в IndexTable (по образцу ui/portfolio_tabs_test.go, ui/input_handler_test.go) (5eb80b3)
  - Acceptance: тесты компилируются и падают
- [x] Task: (Green) components.go: константа `TabIndex`, `IndexTable` в `TabbedView` + страница "index", `UpdateHeader` с 4 табами; input.go: заменить `% 3` на `tabCount` (len-based), ветки `TabIndex` в `switchToTab`/`refresh`, `setupTableNavigation(IndexTable)` (e2fe015)
  - Acceptance: тесты предыдущей задачи зелёные; существующие тесты табов не сломаны
- [~] Task: (Red→Green) Константа `indexList = [{Symbol: "IMOEX@RTSX", Name: "Индекс МосБиржи"}]` + состояние вкладки в `App` (состав, карта `indexQuotes`, флаги загрузки/ошибки) + `loadIndexAsync` (goroutine → `GetIndexConstituents` → QueueUpdateDraw; вызывается при первом входе на вкладку и при `R` после ошибки) + рендер `updateIndexTable` (ui/render.go): заголовок с именем индекса; колонки Ticker | Name | Price | Chg | Chg% | Weight | Volume; сортировка по весу убыв. (без весов — по алфавиту); Chg%/Chg от `Quote.Change` и `close = last − change` с защитой от деления на ноль; формат Weight выбрать по факту данных (нормировка неочевидна — см. spec); зелёный/красный/нейтральный; «—» при отсутствии данных; состояния «Loading…» и «ошибка + повтор по R»; тесты рендера и загрузки
  - Acceptance: тесты зелёные; рендер использует только состояние в памяти, ни одного вызова клиента из рендера; повторный вход на вкладку не дёргает загрузку при живом кеше

## Phase 4: Котировки — стрим + ограниченный фоллбек
- [ ] Task: (Red) Тесты `computeStreamSymbols` с индексом: вкладка Index активна → позиции ∪ символы состава (∪ профиль, если открыт); неактивна → как раньше; тест `flushQuoteInbox`: котировки индексных символов пишутся в `a.indexQuotes` независимо от выбранного счёта и видны рендеру
  - Acceptance: тесты компилируются и падают
- [ ] Task: (Green) Расширить `computeStreamSymbols(positions, profileOpen, profileSymbol, indexActive bool, indexSymbols []string)` (ui/stream.go:131-155) + `recomputeStreamSymbols` при входе/выходе с вкладки (`switchToTab`) и после успешной загрузки состава; `flushQuoteInbox` дополнительно раскладывает inbox в `a.indexQuotes` и перерисовывает вкладку, когда она активна
  - Acceptance: тесты зелёные; уход с вкладки возвращает подписку к набору позиций (захват `SetQuoteSymbols` в моке)
- [ ] Task: (Red→Green) Фоллбек: вход на вкладку при `!streamLive` и загруженном составе → разовый батч `GetQuotes("", indexSymbols)` в goroutine; `R` на вкладке — ручной батч; автообновление при down-стриме — не чаще 1/60с и только при активной вкладке (отметка `lastIndexPoll`, проверка в тике `backgroundRefresh`); `ResourceExhausted` → [WARN], флаг отключения автополлинга до конца сессии, статус-бар сообщает, ручной `R` остаётся. Тесты предиката (матрица: стрим up/down × вкладка активна/нет × кулдаун × флаг отключения)
  - Acceptance: тесты зелёные; в живом-стрим сценарии батч не вызывается вовсе (счётчик мока)
- [ ] Task: (Red→Green) Защита стрима позиций: если после включения индексных символов стрим не перешёл в up за 60с (или N=3 подряд неудачных подписок), индексные символы исключаются из подписки до конца сессии (`indexStreamDisabled`), [WARN]-лог, вкладка живёт на фоллбеке; детектор — чистая функция + интеграционный сценарий с `SubscribeQuoteOverride`, отклоняющим подписки, где символов больше K
  - Acceptance: сценарий показывает восстановление стрима позиций после исключения индексных символов
- [ ] Task: Ревизия гонок: мутации `indexQuotes`/состава/флагов вкладки только на event loop под `dataMutex`; `CGO_ENABLED=1 go test -race ./...` и `-tags=integration ./api/...`
  - Acceptance: race-детектор чист на обеих сюитах

## Phase 5: Действия с бумагой
- [ ] Task: (Red→Green) `Enter` на строке → `OpenProfileForSymbol(symbol)` (возврат из профиля — обратно на вкладку Index с сохранённым выделением); `A`/`Ф` → `OpenOrderModalWithTicker(symbol)` (полный символ проходит существующий путь без изменений, лот прогревается `warmLotSizeAsync`); подсказки статус-бара для вкладки Index («A Buy  R Refresh», ui/render.go:604-619); тесты input-обработчиков и статус-бара
  - Acceptance: тесты зелёные; модалка и профиль открываются с корректным инструментом с любой строки

## Phase 6: Документация и верификация
- [ ] Task: Полная авто-проверка — `go build ./...`, `go vet ./...`, `go test ./...`, `go test -tags=integration ./api/...`, `CGO_ENABLED=1 go test -race` (обе сюиты), `make lint`
  - Acceptance: нет ошибок и предупреждений, линтер чистый
- [ ] Task: Документация — CLAUDE.md (пункт «Index Constituents (GetConstituents)» в API Implementation Details, вкладка Index, расширение стрим-набора, мок в testserver), conductor/product.md (Key Features), CHANGELOG.md, README.md, docs/user_manual/ (страница вкладки Индекс: навигация, клавиши, поведение при недоступном стриме)
  - Acceptance: документация соответствует реализации
- [ ] Task: (Ручной смоук) Реальный ключ: вкладка показывает 46 бумаг IMOEX и наполняется ≤2с при живом стриме; в логе один `GetConstituents` за сессию и нет ResourceExhausted; Enter → профиль; A → модалка с корректным лотом; уход с вкладки → лог переподписки на набор позиций; обрыв сети → фоллбек и восстановление
  - Acceptance: пользователь подтверждает поведение
