
---

````markdown
# 📸 PictBot — Telegram Web App с галереями

**PictBot** — это Telegram Web App, позволяющий пользователям создавать и просматривать фото-галереи, подписываться на других, добавлять галереи в избранное и искать по хештегам. Проект вдохновлён Pinterest, но работает исключительно внутри Telegram с авторизацией через `initData`.

---

Возможности

- ✅ Создание и удаление **собственных** галерей
- ⭐ Добавление галерей в **избранное**
- 🔍 Поиск галерей по **хештегам**
- 🧾 Подписка на других пользователей
- 📦 Docker-образ для развёртывания на сервере

---

Технологии

- **Go** (Golang)  
- **PostgreSQL**  
- **HTML/CSS/JS** (Telegram WebApp)
- **godotenv**, **telegram-bot-api/v5**

---

## ⚙️ Установка и запуск

### 1. Клонируйте репозиторий

```bash
git clone https://github.com/ayrinwave/pictbot.git
cd pictbot
````

### 2. Создайте `.env` на основе шаблона

```bash
cp .env.example .env
```

Заполните файл `.env` своими данными (токен Telegram, данные для подключения к БД и т.д.).

### 3. Создайте таблицы в БД

Перед запуском убедитесь, что у вас запущена PostgreSQL, затем выполните:

```bash
psql -U postgres -d telegram_test -f schema.sql
```
Файл schema.sql находится в подпапке /db.

### 4. Запуск напрямую (Go)

```bash
go run main.go
```




## 🧑‍💻 Автор

Разработчик: [ayrinwave](https://github.com/ayrinwave)
