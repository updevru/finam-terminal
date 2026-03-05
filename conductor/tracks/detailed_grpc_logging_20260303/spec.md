# Spec: Detailed gRPC Error Logging for Broker Support

## Problem

При ошибках вызовов gRPC API брокера в лог пишется минимум информации — обычно только `"failed to ...: <error>"`. Этого недостаточно для обращения в техподдержку брокера: непонятно какой сервис, метод, с какими параметрами вызывался и какой именно gRPC-код вернулся. Только один вызов (`GetAccounts → GetAccount`) уже имеет подробное логирование после недавнего фикса.

## Solution

Привести все gRPC-вызовы в `api/client.go` к единому формату подробного логирования ошибок. Каждый лог-entry при ошибке должен содержать достаточно информации, чтобы поддержка брокера могла понять что случилось.

## Формат логирования

```
[ERROR] <Service>.<Method> failed | <Param1>: <value1> | ... | gRPC code: <code> | Message: <msg> | Endpoint: <endpoint>
```

- **Service.Method** — какой gRPC сервис и метод вызывался
- **Параметры** — все несекретные параметры запроса (AccountId, Symbol, Interval и т.д.)
- **gRPC code** — код ошибки из `status.FromError(err)`
- **Message** — текст ошибки
- **Endpoint** — адрес сервера (`c.conn.Target()`)

## Requirements

- Все 16 gRPC-вызовов в `api/client.go` должны логировать ошибки в едином формате
- Секретные данные (токен, apiToken) НЕ логируются
- Параметры запроса логируются (AccountId, Symbol, Timeframe, Interval)
- gRPC status code извлекается через `google.golang.org/grpc/status`
- Существующее поведение (return error / continue) не меняется
- Вызовы, которые уже логируют `[WARN]`, заменяются на новый формат

## Scope — все gRPC-вызовы

| # | Function | Service.Method | Текущее логирование |
|---|----------|---------------|-------------------|
| 1 | authenticate | AuthService.Auth | Нет (wrapped error) |
| 2 | loadAssetCache | AssetsService.Assets | Нет (WARN via caller) |
| 3 | getFullSymbol | AssetsService.GetAsset | WARN, базовый |
| 4 | fetchLotSize | AssetsService.GetAsset | WARN, базовый |
| 5 | GetAccounts (TokenDetails) | AuthService.TokenDetails | Нет |
| 6 | GetAccounts (GetAccount) | AccountsService.GetAccount | **Уже подробный** |
| 7 | GetAccountDetails | AccountsService.GetAccount | Нет |
| 8 | GetQuotes | MarketDataService.LastQuote | WARN, базовый |
| 9 | GetTradeHistory | AccountsService.Trades | Нет |
| 10 | GetActiveOrders | OrdersService.GetOrders | Нет |
| 11 | GetSnapshots | MarketDataService.LastQuote | WARN, базовый |
| 12 | GetBars | MarketDataService.Bars | Нет |
| 13 | GetAssetInfo | AssetsService.GetAsset | Нет |
| 14 | GetAssetParams | AssetsService.GetAssetParams | Нет |
| 15 | GetSchedule | AssetsService.Schedule | Нет |
| 16 | PlaceOrder | OrdersService.PlaceOrder | Нет |

## Acceptance Criteria

- [ ] Все 16 gRPC-вызовов логируют ошибки в едином формате
- [ ] Формат включает: сервис, метод, параметры, gRPC code, message, endpoint
- [ ] Секретные данные не попадают в лог
- [ ] Существующее поведение (error return / continue) не изменено
- [ ] `go build ./...` без ошибок
- [ ] `go test ./...` проходят
