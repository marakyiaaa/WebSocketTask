# WebSocket

Сервис реализует WebSocket-обработчик (`gotiber/websocket`) c авторизацией по JWT и 
интеграцией с Kafka через `sarama`.

## Архитектура

1. **JWT авторизация** — токен передаётся в заголовке `Authorization: Bearer <token>`. Проверка выполняется через `internal/auth`.
2. **WebSocket** — создаётся клиент, который отправляет все входящие
   сообщения в Kafka (`internal/ws/client.go`).
3. **Kafka Producer** — синхронный продюсер публикует сообщения в указанный топик в
   формате JSON (`internal/kafka/producer.go`).
4. **Kafka Consumer** — consumer group читает топик и передаёт сообщения активным
   соединениям через `Hub` (`internal/kafka/consumer.go`, `internal/ws/hub.go`).

Клиент отправляет на сервер JSON:

```json
{
  "recipients": ["user-1", "user-2"],
  "payload": {"text": "я семен лобанов"}
}
```

Сообщение, которое кладётся в Kafka, имеет вид:

```json
{
  "id": "0e99075e-d552-43c8-a2e0-a0b648f33221",
  "sender_id": "user-1",
  "recipients": [
    "user-1",
    "user-2"
  ],
  "payload": {
    "text": "я семен лобанов"
  },
  "sent_at": "2025-12-02T15:03:15.729641Z"
}
```

### Команды запуска

```
make kafka-start - поднять брокер
make start - запуск 
make create-tocken - генерация JWT
```

## Дополнение

- При отсутствии активных WebSocket соединений consumer возвращает
  ошибку `recipient offline`, оставляет сообщение в топике до появления
  нового соединения.
- Модели сообщений не выделены в отдельный слой бизнес логики.
