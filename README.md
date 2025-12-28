# Music Club Bot

## 🚀 Быстрый старт для локальной разработки

Эта инструкция покажет, как запустить весь проект локально с Telegram ботом.

### 1. Создание бота

1. Открой [@BotFather](https://t.me/BotFather) в Telegram
2. Отправь команду `/newbot`
3. Следуй инструкциям: введи имя и username для бота
4. Скопируй полученный **BOT_TOKEN** (например: `2201663460:AAFEvojHympVcu9IIYDEUJYeJhGCbtN4ffo`)
5. Отправь `/setdomain` и выбери своего бота, чтобы настроить домен для WebApp (пока можешь пропустить, настроишь после получения tunnel URL)

### 2. Настройка HTTPS туннелей

Telegram требует HTTPS для WebApp, поэтому используем Cloudflare Tunnel.

**Установка cloudflared** (если еще не установлен):
```bash
npm install -g cloudflared
```

**Запусти два туннеля в отдельных терминалах:**

Терминал 1 - для backend:
```bash
npx cloudflared tunnel --url http://localhost:6969
```
Скопируй полученный URL (например: `https://gain-murray-attach-pool.trycloudflare.com`)

Терминал 2 - для frontend:
```bash
npx cloudflared tunnel --url http://localhost:5173
```
Скопируй полученный URL (например: `https://improve-relatively-colorado-objects.trycloudflare.com`)

### 3. Настройка переменных окружения

1. Скопируй файл с примером:
   ```bash
   cp .env.example .env
   ```

2. Открой `.env` и заполни следующие переменные:

   ```bash
   # Telegram Bot
   BOT_TOKEN=ваш_токен_от_BotFather
   BOT_USERNAME=@ваш_бот_username

   # Backend URL for Frontend из cloudflared tunnel (терминал 1)
   VITE_GRPC_HOST=https://ваш-tunnel-url-для-backend.trycloudflare.com

   # Frontend URL for WebApp - URL из cloudflared tunnel (терминал 2)
   WEBAPP_URL=https://ваш-tunnel-url-для-frontend.trycloudflare.com

   # Остальные переменные можно оставить по умолчанию
   SKIP_CHAT_MEMBERSHIP_CHECK=true  # Отключить проверку членства в чате для разработки
   ```

3. Вернись в [@BotFather](https://t.me/BotFather) и настрой WebApp:
   - Отправь `/mybots`
   - Выбери своего бота
   - Выбери "Bot Settings" → "Menu Button"
   - Отправь URL из `WEBAPP_URL`

### 4. Запуск сервисов

Запусти все сервисы (PostgreSQL, Redis, Backend, Frontend, Bot) одной командой, обязательно на этапе сборки указав файл с переменными окружения для dev среды: 

```bash
docker compose -f docker-compose.yml --env-file .env up --build
```

Флаг `--build` пересоберет образы с новыми переменными окружения.

Для запуска в фоновом режиме добавь `-d`:
```bash
docker compose -f docker-compose.yml up --build -d
```

### 5. Проверка работы

1. **Backend**: Откройте в браузере ваш backend tunnel URL - должна открыться страница "404 page not found"
2. **Frontend**: Откройте в браузере ваш frontend tunnel URL - должно загрузиться приложение и сказать что его можно использовать только в телеграме 
3. **Bot**: Откройте своего бота в Telegram и нажмите кнопку Menu/WebApp - должно открыться приложение
пример успешных логов docker compose 
![alt text](docs/image.png)
### 6. Просмотр логов

Логи всех сервисов:
```bash
docker compose -f docker-compose.dev.yml logs -f
```

Логи конкретного сервиса:
```bash
docker compose -f docker-compose.dev.yml logs -f bot
docker compose -f docker-compose.dev.yml logs -f backend
docker compose -f docker-compose.dev.yml logs -f frontend
```

### 7. Остановка сервисов

```bash
docker compose -f docker-compose.dev.yml down
```

Удалить также и данные базы:
```bash
docker compose -f docker-compose.dev.yml down --volumes
```

### 🔧 Обновление tunnel URLs

Если cloudflare tunnel URL изменился (при перезапуске tunnel):

1. Обнови `VITE_GRPC_HOST` и `WEBAPP_URL` в `.env`
2. **Важно**: Пересобери frontend с флагом `--no-cache`, так как Vite встраивает переменные на этапе сборки:
   ```bash
   docker compose -f docker-compose.dev.yml build --no-cache frontend
   docker compose -f docker-compose.dev.yml up -d frontend
   ```
3. Сделай жесткий refresh в браузере: `Ctrl + Shift + R` (Windows/Linux) или `Cmd + Shift + R` (Mac)
4. Обнови Menu Button URL в [@BotFather](https://t.me/BotFather)

### 📦 Доступные сервисы

После запуска будут доступны:
- **Backend (gRPC)**: http://localhost:6969 (и через tunnel)
- **Frontend**: http://localhost:5173 (и через tunnel)
- **PostgreSQL**: localhost:5432
- **Redis**: localhost:6379
- **Adminer** (веб-интерфейс БД): http://localhost:8080
