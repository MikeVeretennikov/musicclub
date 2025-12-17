from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    PROJECT_NAME: str = "musicclubbot"
    LOG_LEVEL: str = "INFO"

    BOT_TOKEN: str
    ADMIN_IDS: set[int]
    CHAT_ID: int
    PAGE_SIZE: int = 4

    REDIS_HOST: str
    REDIS_PORT: int
    REDIS_PASSWORD: str
    REDIS_DB: int

    POSTGRES_URL: str


settings = Settings()
