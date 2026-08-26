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

## Phase 3: Четвёртая вкладка UI [checkpoint: cdbd65d]
- [x] Task: (Red) Тесты вкладки: цикл `nextTab`/`prevTab` проходит 4 вкладки (Positions→History→Orders→Index→Positions), `SetTab(TabIndex)` переключает страницу "index", заголовок содержит " Index ", фокус уходит в IndexTable (по образцу ui/portfolio_tabs_test.go, ui/input_handler_test.go) (5eb80b3)
  - Acceptance: тесты компилируются и падают
- [x] Task: (Green) components.go: константа `TabIndex`, `IndexTable` в `TabbedView` + страница "index", `UpdateHeader` с 4 табами; input.go: заменить `% 3` на `tabCount` (len-based), ветки `TabIndex` в `switchToTab`/`refresh`, `setupTableNavigation(IndexTable)` (e2fe015)
  - Acceptance: тесты предыдущей задачи зелёные; существующие тесты табов не сломаны
- [x] Task: (Red→Green) Константа `indexList = [{Symbol: "IMOEX@RTSX", Name: "Индекс МосБиржи"}]` + состояние вкладки в `App` (состав, карта `indexQuotes`, флаги загрузки/ошибки) + `loadIndexAsync` (goroutine → `GetIndexConstituents` → QueueUpdateDraw; вызывается при первом входе на вкладку и при `R` после ошибки) + рендер `updateIndexTable` (ui/render.go): заголовок с именем индекса; колонки Ticker | Name | Price | Chg | Chg% | Weight | Volume; сортировка по весу убыв. (без весов — по алфавиту); Chg%/Chg от `Quote.Change` и `close = last − change` с защитой от деления на ноль; формат Weight выбрать по факту данных (нормировка неочевидна — см. spec); зелёный/красный/нейтральный; «—» при отсутствии данных; состояния «Loading…» и «ошибка + повтор по R»; тесты рендера и загрузки (027ac80)
  - Acceptance: тесты зелёные; рендер использует только состояние в памяти, ни одного вызова клиента из рендера; повторный вход на вкладку не дёргает загрузку при живом кеше

## Phase 4: Котировки — стрим + ограниченный фоллбек [checkpoint: 164f7bd]
- [x] Task: (Red) Тесты `computeStreamSymbols` с индексом: вкладка Index активна → позиции ∪ символы состава (∪ профиль, если открыт); неактивна → как раньше; тест `flushQuoteInbox`: котировки индексных символов пишутся в `a.indexQuotes` независимо от выбранного счёта и видны рендеру (a1ffe91)
  - Acceptance: тесты компилируются и падают
- [ ] Task: (Green) Расширить `computeStreamSymbols(positions, profileOpen, profileSymbol, indexActive bool, indexSymbols []string)` (ui/stream.go:131-155) + `recomputeStreamSymbols` при входе/выходе с вкладки (`switchToTab`) и после успешной загрузки состава; `flushQuoteInbox` дополнительно раскладывает inbox в `a.indexQuotes` и перерисовывает вкладку, когда она активна (ee0776f)
  - Acceptance: тесты зелёные; уход с вкладки возвращает подписку к набору позиций (захват `SetQuoteSymbols` в моке)
- [x] Task: (Red→Green) Фоллбек: вход на вкладку при `!streamLive` и загруженном составе → разовый батч `GetQuotes("", indexSymbols)` в goroutine; `R` на вкладке — ручной батч; автообновление при down-стриме — не чаще 1/60с и только при активной вкладке (отметка `lastIndexPoll`, проверка в тике `backgroundRefresh`); `ResourceExhausted` → [WARN], флаг отключения автополлинга до конца сессии, статус-бар сообщает, ручной `R` остаётся. Тесты предиката (матрица: стрим up/down × вкладка активна/нет × кулдаун × флаг отключения) (b06ffe8)
  - Acceptance: тесты зелёные; в живом-стрим сценарии батч не вызывается вовсе (счётчик мока)
- [x] Task: (Red→Green) Защита стрима позиций: если после включения индексных символов стрим не перешёл в up за 60с (или N=3 подряд неудачных подписок), индексные символы исключаются из подписки до конца сессии (`indexStreamDisabled`), [WARN]-лог, вкладка живёт на фоллбеке; детектор — чистая функция + интеграционный сценарий с `SubscribeQuoteOverride`, отклоняющим подписки, где символов больше K (0663c0b)
  - Acceptance: сценарий показывает восстановление стрима позиций после исключения индексных символов
- [x] Task: Ревизия гонок: мутации `indexQuotes`/состава/флагов вкладки только на event loop под `dataMutex`; `CGO_ENABLED=1 go test -race ./...` и `-tags=integration ./api/...` (6f7aa9a — локально `-race` невозможен: на машине нет C-компилятора, гейт остаётся за CI; вместо него добавлен стресс-тест конкурентного доступа)
  - Acceptance: race-детектор чист на обеих сюитах

## Phase 5: Действия с бумагой [checkpoint: 7df52e7]
- [x] Task: (Red→Green) `Enter` на строке → `OpenProfileForSymbol(symbol)` (возврат из профиля — обратно на вкладку Index с сохранённым выделением); `A`/`Ф` → `OpenOrderModalWithTicker(symbol)` (полный символ проходит существующий путь без изменений, лот прогревается `warmLotSizeAsync`); подсказки статус-бара для вкладки Index («A Buy  R Refresh», ui/render.go:604-619); тесты input-обработчиков и статус-бара (7a10aaa)
  - Acceptance: тесты зелёные; модалка и профиль открываются с корректным инструментом с любой строки

## Phase 6: Документация и верификация
- [x] Task: Полная авто-проверка — `go build ./...`, `go vet ./...`, `go test ./...`, `go test -tags=integration ./api/...`, `CGO_ENABLED=1 go test -race` (обе сюиты), `make lint` — всё зелёное, кроме `-race`: на машине нет C-компилятора (gcc/clang/zig отсутствуют), гейт остаётся за CI; вместо него локально прогнан стресс-тест конкурентного доступа
  - Acceptance: нет ошибок и предупреждений, линтер чистый
- [x] Task: Документация — CLAUDE.md (пункт «Index Constituents (GetConstituents)» в API Implementation Details, вкладка Index, расширение стрим-набора, мок в testserver), conductor/product.md (Key Features), CHANGELOG.md, README.md, docs/user_manual/ (страница вкладки Индекс: навигация, клавиши, поведение при недоступном стриме) (0420c03)
  - Acceptance: документация соответствует реализации
- [ ] Task: (Ручной смоук) Реальный ключ: вкладка показывает 46 бумаг IMOEX и наполняется ≤2с при живом стриме; в логе один `GetConstituents` за сессию и нет ResourceExhausted; Enter → профиль; A → модалка с корректным лотом; уход с вкладки → лог переподписки на набор позиций; обрыв сети → фоллбек и восстановление
  - Acceptance: пользователь подтверждает поведение

## Phase 7: Правки по итогам ручной проверки (2026-08-26)
Ручной смоук на реальном ключе опроверг два проектных допущения и вскрыл два дефекта рендера. Факты из `finam-terminal.log`:
- `SubscribeQuote` **имеет** лимит на число символов: `InvalidArgument: Maximum number of symbols exceeded` при 46 символах (работали подписки на 1/2/4/5; точное значение лимита неизвестно и не документировано).
- Батч `LastQuote` по 46 символам мгновенно ловит `ResourceExhausted: Too Many Requests`, то есть бюджет 200/мин не применим к всплеску; плюс `GetQuotes` тратит один общий 30s-контекст на весь цикл, из-за чего хвост символов падает с `DeadlineExceeded`.
- Заголовок таблицы не закреплён: при прокрутке 46 строк он уходит за экран, унося подписи колонок и единственный источник `Expansion` (tview считает ширину только по видимым строкам), из-за чего таблица схлопывается по ширине контента.

- [x] Task: (Red→Green) Рендер таблицы: `SetFixed(1, 0)` для `IndexTable` (заголовок закреплён) + `Expansion` на всех ячейках, а не только на заголовке (Name забирает свободную ширину); тесты на закрепление заголовка и на ненулевой expansion в строках данных
  - Acceptance: подписи колонок видны при любой прокрутке, таблица занимает всю доступную ширину (4f1b06e)
- [x] Task: (Red→Green) Адаптивный лимит символов подписки в `api`: `SetQuoteSymbols` сохраняет приоритетный порядок вызывающего (без сортировки), клиент режет список до `quoteSymbolCap`; при ошибке «Maximum number of symbols exceeded» кап уменьшается вдвое и подписка повторяется без сообщения об обрыве. Позиции идут первыми в приоритете и никогда не вытесняются
  - Acceptance: интеграционный сценарий с мок-сервером, отклоняющим подписки больше K символов, сходится к рабочему размеру сам; позиции всегда в подписке (6b32fb5)
- [x] Task: (Red→Green) Индексные символы в подписке — окно вокруг видимых строк таблицы, пересчёт при прокрутке; фоллбек-батч ограничен тем же окном, у каждого `LastQuote` собственный дедлайн (исправление общего контекста в `GetQuotes`)
  - Acceptance: всплеск запросов не превышает размера окна; при прокрутке котировки следуют за курсором (6b32fb5, b913669)
- [ ] Task: Документация и повторный ручной смоук
  - Acceptance: пользователь подтверждает поведение
