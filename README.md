# Многопоточная очередь сообщений

Работает по принципу FIFO.
Первым аргументом принимает порт на котором будет слушать.

## Положить сообщение в очередь
`PUT /queue?v=value`

### Параметры:
- `queue` - имя очереди (может быть любым)
- `value` - сообщение очереди

### Ответы:
- Успех - пустое тело, статус 200
- Отсутствие параметра 'v' - пустое тело, статус 400

### Примеры:
```
curl -XPUT http://127.0.0.1/pet?v=cat
curl -XPUT http://127.0.0.1/pet?v=dog
curl -XPUT http://127.0.0.1/role?v=manager
curl -XPUT http://127.0.0.1/role?v=executive
```

## Забрать сообщение из очереди
`GET /queue?timeout=time`

### Параметры:
- `queue` - имя очереди
- `time` - время (в секундах) в течении которого будет ожидаться новое сообщение (необязательный параметр), сообщения выдаются в порядке FIFO (первый запросил - первый получил)

### Ответы:
- Успех - сообщение из очереди, статус 200
- Отсутсвие сообщения, либо истечение таймаута - пустое тело, статус 404

### Примеры:
```
curl http://127.0.0.1/pet
curl http://127.0.0.1/role?timeout=10
```