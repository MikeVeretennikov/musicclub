import asyncio
import importlib
import logging
from pathlib import Path
from types import ModuleType
from typing import Iterable

import redis.asyncio as redis
from aiogram import Bot, Dispatcher
from aiogram.client.default import DefaultBotProperties
from aiogram.enums import ParseMode
from aiogram.fsm.storage.base import DefaultKeyBuilder
from aiogram.fsm.storage.redis import RedisStorage
from aiogram_dialog import setup_dialogs

from bot.middlewares.pm import PrivateChatOnlyMiddleware
from bot.models import Base
from bot.services.database import init_db, close_db, get_db_session, engine
from bot.services.invite_link import ChatInviteLinkManager
from .services.settings import settings

logger = logging.getLogger(__name__)

HANDLERS_PACKAGE = "bot.handlers"
HANDLERS_PATH = Path(__file__).parent / "handlers"


def register_all_middlewares(dp: Dispatcher) -> None:
    dp.message.middleware(PrivateChatOnlyMiddleware())
    dp.callback_query.middleware(PrivateChatOnlyMiddleware())


def _iter_handler_modules() -> Iterable[ModuleType]:
    logger.info("Handerls_path: %s", HANDLERS_PATH)
    for module in HANDLERS_PATH.rglob("*.py"):
        if module.name == "__init__.py":
            continue
        relative = module.relative_to(HANDLERS_PATH).with_suffix("")
        dotted = ".".join((HANDLERS_PACKAGE, *relative.parts))
        yield importlib.import_module(dotted)


def register_all_handlers(dp: Dispatcher) -> None:
    routers = []
    for module in _iter_handler_modules():
        router = getattr(module, "router", None)
        if router is None:
            continue
        routers.append(router)

    if routers:
        dp.include_routers(*routers)
    else:
        logger.warning("Не найдено ни одного роутера для регистрации")


async def on_startup(bot: Bot) -> None:
    await ChatInviteLinkManager(bot, settings.CHAT_ID).startup()


async def on_shutdown(bot: Bot) -> None:
    await ChatInviteLinkManager(bot, settings.CHAT_ID).shutdown()


async def run_bot() -> None:
    storage = RedisStorage(
        redis=redis.Redis(
            host=settings.REDIS_HOST,
            port=settings.REDIS_PORT,
            password=settings.REDIS_PASSWORD,
            db=settings.REDIS_DB,
        ),
        key_builder=DefaultKeyBuilder(with_destiny=True),
    )

    bot = Bot(
        token=settings.BOT_TOKEN,
        default=DefaultBotProperties(parse_mode=ParseMode.HTML),
    )
    dp = Dispatcher(storage=storage)

    register_all_middlewares(dp)
    register_all_handlers(dp)
    setup_dialogs(dp)

    dp.startup.register(on_startup)
    dp.shutdown.register(on_shutdown)

    try:
        await dp.start_polling(bot)
    finally:
        await dp.storage.close()
        await bot.session.close()


async def main() -> None:
    logger.info("Запущен бот в проекте: %s", settings.PROJECT_NAME)

    await init_db()
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    logger.info("Database initialized successfully")
    await run_bot()
    await close_db()


if __name__ == "__main__":
    logging.basicConfig(level=settings.LOG_LEVEL)
    asyncio.run(main())
