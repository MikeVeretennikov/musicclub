# Инструкция по деплою

## Подготовка VDS сервера

### 1. Подключение к серверу
```bash
ssh -i ~/.ssh/selectel_musicclub root@82.148.28.47
```

### 2. Создание директории проекта
```bash
mkdir -p /opt/musicclub
cd /opt/musicclub
```

### 3. Клонирование репозитория
```bash
git clone https://github.com/MikeVeretennikov/musicclub.git .
```

### 4. Создание .env файла
```bash
nano .env
```

Заполните следующие переменные:
```env
LOG_LEVEL=INFO

# Telegram Bot
BOT_TOKEN=ваш_токен_от_BotFather
BOT_USERNAME=@ваш_бот
CHAT_ID=-100ваш_chat_id
ADMIN_IDS="[ваш_telegram_id]"

# Backend
GRPC_PORT=6969
JWT_SECRET=сгенерируйте_случайную_строку
JWT_TTL_SECONDS=7200
SKIP_CHAT_MEMBERSHIP_CHECK=false

# PostgreSQL
POSTGRES_USER=postgres
POSTGRES_PASSWORD=сильный_пароль_для_production
POSTGRES_DB=musicclub
POSTGRES_HOST=db
POSTGRES_PORT=5432
POSTGRES_URL=postgresql://postgres:ваш_пароль@db:5432/musicclub?sslmode=disable

# Frontend
VITE_GRPC_HOST=https://musicalclub.duckdns.org
WEBAPP_URL=https://musicalclub.duckdns.org
```

### 5. Получение SSL сертификата

Сначала запустите только nginx без SSL (закомментируйте SSL строки в nginx/conf.d/default.conf):
```bash
docker compose -f docker-compose.prod.yml up -d nginx
```

Получите сертификат:
```bash
docker compose -f docker-compose.prod.yml run --rm certbot certonly \
  --webroot \
  --webroot-path=/var/www/certbot \
  --email ваш-email@example.com \
  --agree-tos \
  --no-eff-email \
  -d musicalclub.duckdns.org
```

Раскомментируйте SSL строки в nginx/conf.d/default.conf и перезапустите nginx:
```bash
docker compose -f docker-compose.prod.yml restart nginx
```

### 6. Запуск всех сервисов
```bash
docker compose -f docker-compose.prod.yml up -d
```

### 7. Проверка статуса
```bash
docker compose -f docker-compose.prod.yml ps
docker compose -f docker-compose.prod.yml logs -f
```

## Настройка GitHub Secrets

Перейдите в Settings → Secrets and variables → Actions вашего репозитория и добавьте:

### Required secrets:
- **DOCKERHUB_USERNAME**: ваш Docker Hub username
- **DOCKERHUB_TOKEN**: ваш Docker Hub токен (создайте на hub.docker.com)
- **VDS_HOST**: IP адрес вашего VDS сервера
- **VDS_USERNAME**: `root` или другой пользователь
- **VDS_SSH_KEY**: содержимое файла с приватным SSH ключом
- **VITE_GRPC_HOST**: `https://ваш-домен.duckdns.org`

## Как работает автоматический деплой

1. Вы делаете `git push` в ветку `master`
2. GitHub Actions:
   - Собирает Docker образы (backend, frontend, bot)
   - Пушит их в Docker Hub
   - Подключается к VDS по SSH
   - Выполняет `docker compose pull && docker compose up -d`
3. Сервер автоматически обновляется новыми версиями

## Полезные команды на сервере

### Просмотр логов
```bash
cd /opt/musicclub
docker compose -f docker-compose.prod.yml logs -f
docker compose -f docker-compose.prod.yml logs -f backend
docker compose -f docker-compose.prod.yml logs -f frontend
```

### Перезапуск сервисов
```bash
docker compose -f docker-compose.prod.yml restart
docker compose -f docker-compose.prod.yml restart backend
```

### Обновление кода вручную
```bash
cd /opt/musicclub
git pull origin master
docker compose -f docker-compose.prod.yml pull
docker compose -f docker-compose.prod.yml up -d
```

### Очистка Docker
```bash
docker system prune -af
```

### Проверка SSL сертификата
```bash
docker compose -f docker-compose.prod.yml logs certbot
```

## Troubleshooting

### Сертификат не получается
- Убедитесь что порт 80 открыт: `ufw allow 80`
- Проверьте что домен указывает на сервер: `dig musicalclub.duckdns.org`
- Проверьте логи certbot

### Приложение не запускается
- Проверьте логи: `docker compose -f docker-compose.prod.yml logs`
- Убедитесь что .env файл создан и заполнен
- Проверьте что база данных запущена: `docker compose -f docker-compose.prod.yml ps`

### GitHub Actions не может подключиться к серверу
- Проверьте что SSH ключ добавлен в GitHub Secrets
- Убедитесь что ключ имеет правильные права: `chmod 600 ~/.ssh/selectel_musicclub`
