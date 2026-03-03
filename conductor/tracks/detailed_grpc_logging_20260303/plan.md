# Plan: Detailed gRPC Error Logging for Broker Support

## Overview

Добавить подробное логирование ошибок для всех gRPC-вызовов в `api/client.go`. Единственный файл для изменения. Вводим хелпер-функцию для единообразного формата, затем применяем ко всем вызовам.

## Phase 1: Logging Helper

- [x] Task: Создать хелпер `logGRPCError` в `api/client.go` e9f6b1c
  - Сигнатура: `func (c *Client) logGRPCError(service, method string, err error, params ...string)`
  - Извлекает gRPC status code и message через `status.FromError(err)`
  - Формирует лог-строку: `[ERROR] <Service>.<Method> failed | <params...> | gRPC code: <code> | Message: <msg> | Endpoint: <endpoint>`
  - `params` — пары "Key: value" для параметров запроса
  - Acceptance: функция компилируется, формат лога соответствует спецификации

## Phase 2: Auth & Asset Cache (3 вызова)

- [ ] Task: `authenticate` — AuthService.Auth
  - Параметры для лога: нет (токен секретный)
  - Acceptance: лог содержит сервис, метод, gRPC code, endpoint
- [ ] Task: `loadAssetCache` — AssetsService.Assets
  - Параметры для лога: нет (пустой запрос)
  - Acceptance: лог содержит сервис, метод, gRPC code, endpoint
- [ ] Task: `GetAccounts` (TokenDetails) — AuthService.TokenDetails
  - Параметры для лога: нет (токен секретный)
  - Acceptance: лог содержит сервис, метод, gRPC code, endpoint

## Phase 3: Account Operations (4 вызова)

- [ ] Task: `GetAccounts` (GetAccount loop) — уже реализован, рефакторить на хелпер
  - Параметры: AccountId
  - Acceptance: формат сохранён, используется хелпер
- [ ] Task: `GetAccountDetails` — AccountsService.GetAccount
  - Параметры: AccountId
  - Acceptance: лог + wrapped error return сохранён
- [ ] Task: `GetTradeHistory` — AccountsService.Trades
  - Параметры: AccountId, Interval (start/end в RFC3339)
  - Acceptance: лог + wrapped error return сохранён
- [ ] Task: `GetActiveOrders` — OrdersService.GetOrders
  - Параметры: AccountId
  - Acceptance: лог + wrapped error return сохранён

## Phase 4: Market Data (4 вызова)

- [ ] Task: `GetQuotes` — MarketDataService.LastQuote
  - Параметры: Symbol
  - Acceptance: заменяет текущий `[WARN]`, лог + continue сохранён
- [ ] Task: `GetSnapshots` — MarketDataService.LastQuote
  - Параметры: Symbol
  - Acceptance: заменяет текущий `[WARN]`, лог + continue сохранён
- [ ] Task: `GetBars` — MarketDataService.Bars
  - Параметры: Symbol, Timeframe, Interval
  - Acceptance: лог + wrapped error return сохранён

## Phase 5: Asset Info (4 вызова)

- [ ] Task: `getFullSymbol` — AssetsService.GetAsset
  - Параметры: Symbol, AccountId
  - Acceptance: заменяет текущий `[WARN]`, fallback-логика сохранена
- [ ] Task: `fetchLotSize` — AssetsService.GetAsset
  - Параметры: Symbol, AccountId
  - Acceptance: заменяет текущий `[WARN]`
- [ ] Task: `GetAssetInfo` — AssetsService.GetAsset
  - Параметры: Symbol, AccountId
  - Acceptance: лог + wrapped error return сохранён
- [ ] Task: `GetAssetParams` — AssetsService.GetAssetParams
  - Параметры: Symbol, AccountId
  - Acceptance: лог + wrapped error return сохранён

## Phase 6: Orders & Schedule (2 вызова)

- [ ] Task: `GetSchedule` — AssetsService.Schedule
  - Параметры: Symbol
  - Acceptance: лог + wrapped error return сохранён
- [ ] Task: `PlaceOrder` — OrdersService.PlaceOrder
  - Параметры: AccountId, Symbol, Side, Quantity (в лотах)
  - Acceptance: лог + wrapped error return сохранён

## Phase 7: Verification

- [ ] Task: `go build ./...` — сборка без ошибок
- [ ] Task: `go test ./...` — тесты проходят
- [ ] Task: Ревью — все 16 вызовов покрыты, формат единообразный
