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

```
{
  "message": "Poll title", // string значение, максимальная длина 4096 символов
  "buttons": [
    "option 1", // string значение, максимальная длина 100 символов
    "option 2", // string значение
    "option 3" // string значение
  ]
}
```

Если поле message превышает 300 символов, то задание разбивается на два этапа:
- *посылка сообщения, с содержимым поля message* 
- *посылка опроса, с заголовком "choose an option"*

_Ответ:_
```
{
  "request_id": "cbh3tjmg26ucsirdm4rg" // string значение, идентификатор опроса
}
```

#### Получить статус опроса

_Запрос:_ GET /v1/:request_id

_Пример:_ GET /v1/cbh3tjmg26ucsirdm4rg

_Варианты ответов:_
```json
{
  "status": "processing"
}
```
Выдается, в случае если опрос создан, но еще не получен ответ на этот опрос пользователем Telegram
```
{
  "status": "done",
  "option": 0, // int значение - индекс выбранного ответа из массива buttons (счет начинается с нуля)
  "text": "Option 1" // string значение - текст выбранного ответа
}
```

В случае ошибок, сервер вернет ответ вида
```
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

В случае, если поле message превышает 300 символов, задача разбивается на две части:
- сообщение с текстом
- сообщение с опросом

![BigPoll](https://github.com/dimcz/tgpollbot/blob/main/docs/bigpoll.jpg)