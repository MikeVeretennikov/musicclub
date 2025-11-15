FROM python:3.12-slim

WORKDIR /app

RUN apt-get update && apt-get install -y python3-venv gcc build-essential && rm -rf /var/lib/apt/lists/*

RUN pip install poetry

COPY pyproject.toml poetry.lock* ./

RUN poetry install --no-interaction --no-ansi --no-root

COPY . .

CMD ["poetry", "run", "python", "-m", "bot.main"]
