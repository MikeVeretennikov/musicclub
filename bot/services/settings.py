from typing import Set

from pydantic import Field, computed_field
from pydantic_settings import BaseSettings


class Settings(BaseSettings):
    PROJECT_NAME: str = "musicclubbot"
    LOG_LEVEL: str = "INFO"

    BOT_TOKEN: str
    ADMIN_IDS: Set[int]
    CHAT_LINK: str
    PAGE_SIZE: int = 4

    REDIS_HOST: str
    REDIS_PORT: int
    REDIS_PASSWORD: str
    REDIS_DB: int

    POSTGRES_URL: str


settings = Settings()
