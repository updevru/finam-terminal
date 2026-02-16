# Plan: Отображение дат в таймзоне пользователя

## Overview

Конвертация всех временных меток из UTC в локальную таймзону при получении данных из API. Изменения минимальны — добавление `.Local()` к вызовам `.AsTime()` в `api/client.go`.

## Phase 1: Конвертация временных меток в API клиенте

- [x] Task: Конвертировать timestamp сделок в локальное время `6fd774c`
  - File: `api/client.go`, строка ~733
  - Change: `t.Timestamp.AsTime()` → `t.Timestamp.AsTime().Local()`
  - Acceptance: Даты сделок в History отображаются в локальной таймзоне

- [x] Task: Конвертировать timestamp ордеров в локальное время `3b63d90`
  - File: `api/client.go`, строка ~810
  - Change: `o.TransactAt.AsTime()` → `o.TransactAt.AsTime().Local()`
  - Acceptance: Даты ордеров в Orders отображаются в локальной таймзоне

- [x] Task: Конвертировать timestamp котировок в локальное время `32351c4`
  - File: `api/client.go`, строки ~654, ~853
  - Change: `q.Timestamp.AsTime()` → `q.Timestamp.AsTime().Local()`
  - Acceptance: Временные метки котировок в локальной таймзоне

- [ ] Task: Конвертировать дату открытия счёта в локальное время
  - File: `api/client.go`, строки ~530, ~562
  - Change: `accountResp.OpenAccountDate.AsTime()` → `accountResp.OpenAccountDate.AsTime().Local()`
  - Acceptance: Дата открытия счёта в локальной таймзоне

## Phase 2: Верификация

- [ ] Task: Собрать проект и убедиться в отсутствии ошибок компиляции
  - Command: `go build ./...`
  - Acceptance: Проект собирается без ошибок

- [ ] Task: Ручная проверка — запустить приложение и убедиться, что время совпадает с системным
  - Acceptance: Даты в History и Orders совпадают с ожидаемыми в локальной таймзоне
