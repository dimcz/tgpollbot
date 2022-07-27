# TGPollBot


---
## Шлюз для Telegram API

### Запуск

---
Для запуска необходимы следующие переменные окружения:
- X_API_KEY - ключ авторизации HTTP запросов
- USERS - список ID пользователей Telegram, разделенных запятой
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
      - REDIS_DB=redisdb://redis:6379/1
      - X_API_KEY=1234567890
      - USERS=123456789,225544522
      - TOKEN=XXXXXXXXXX:YYYYYYYYYYYYYYYYYYYYY
 ```

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

Если поле message превышает 300 символов, то задание разбивается на два этапа:
- посылка сообщения, с остатками текста, что превышает 300 символов
- посылка опроса, с длиной заголовка не более 300 символов

Максимальная длина полей buttons - 100 символов

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
  "status": "processing"
}
```
Выдается, в случае если опрос создан, но еще не получен ответ на этот опрос пользователем Telegram
```json
{
  "status": "done",
  "option": 0, // int значение - индекс выбранного ответа из массива buttons (счет начинается с нуля)
  "text": "Option 1" // string значение - текст выбранного ответа
}
```

В случае ошибок, сервер вернет ответ вида
```json
{
  "error": "краткое описание ошибки",
  "details": "подробное описание ошибки"
}
```
с соответствующим кодом ответа
### Telegram

---

Для работы с ботом, необходимо первоначально создать чат с ним:

Если пользователь не имеет доступа к сервису, то в ответ будет послано сообщение с текстом, где указан ID пользователя

![UserID](https://github.com/dimcz/tgpollbot/blob/main/docs/userid.jpg)

Иначе, пользователю будет сообщено, что он имеет доступ к сервису

![NotAllowed](https://github.com/dimcz/tgpollbot/blob/main/docs/start.jpg)

После регистрации нового опроса в системе пользователь телеграма получит 

![NewPoll](https://github.com/dimcz/tgpollbot/blob/main/docs/poll.jpg)

В случае, если поле message превышает 300 символов, задача разбивается на два запроса:
- сообщение с текстом
- сообщение с опросом

![BigPoll](https://github.com/dimcz/tgpollbot/blob/main/docs/bigpoll.jpg)