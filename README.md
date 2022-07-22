# TGPollBot


---
## Шлюз для Telegram API

### Запуск

---
Для запуска необходимы следующие переменные окружения:
- X_API_KEY - ключ авторизации HTTP запросов
- USERS - список пользователй Telegram, разделенных запятой
- TOKEN - ключ авторизации бота в системе Telegram
- REDIS_DB - подключение к Redis серверу вида redisdb://username:password@localhost:port/db

 Примерный вид файла docker-compose.yml может быть таким:
 ```
 version: "3.9"

services:
  tgpollbot:
    image: skydim/tgpollbot
    container_name: tgpollbot
    restart: always
    ports:
      - 8080:8080
    environment:
      - X_API_KEY=1234567890
      - USERS=Username
      - TOKEN=XXXXXXXXXX:YYYYYYYYYYYYYYYYYYYYY
 ```
_Примечание: в случае, если не указана переменная REDIS_DB, в качестве DB очереди используется собственная
локальная база_

### HTTP API

---
_Для доступа используется ключ доступа X-API-Key, передаваемый в header_

#### Создать Опрос
_Запрос:_ POST /v1/

```json
{
  "message": "Poll title", // string значение
  "buttons": [
    "option 1", // string значение
    "option 2", // string значение
    "option 3" // string значение
  ]
}
```
_Ответ:_
```json
{
  "request_id": "4c037184-280b-41d9-831c-4d739c4b780c" // string значение, идентификатор опроса
}
```

#### Получить статус опроса

_Запрос:_ GET /v1/:request_id

_Пример:_ GET /v1/4c037184-280b-41d9-831c-4d739c4b780c

_Варианты ответов:_
```json
{
  "status": "process"
}
```
Выдается, в случае если опрос создан, но еще не получен ответ на этот опрос пользователем Telegram
```json
{
  "status": "done",
  "option": 1, // int значение - индекс выбранного ответа из массива buttons (счет начинается с нуля)
  "text": "Option 1" // string значение - текст выбранного ответа
}
```

### Telegram

---

Для работы с ботом, необходимо первоначально создать чат с ним:

![начало работы с ботом](https://github.com/dimcz/tgpollbot/blob/main/docs/start.jpg)

После регистрации нового опроса в системе пользователь телеграма получит 

![новый опрос](https://github.com/dimcz/tgpollbot/blob/main/docs/poll.jpg)