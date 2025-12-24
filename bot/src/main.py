import os
import asyncio
from uuid import UUID
import logging

from aiogram import Bot, Dispatcher, Router
from aiogram.types import Message, InlineKeyboardMarkup, InlineKeyboardButton, WebAppInfo
from aiogram.filters import CommandStart, Command, CommandObject
from aiogram.utils.i18n import I18n, gettext as _
from aiogram.utils.i18n.middleware import I18nMiddleware
from db import create_connection, execute

logger = logging.getLogger(__name__)


def getenv(name: str):
    v = os.getenv(name)
    if not v:
        logger.error("%s environment variable is not set.", name)
        os.exit(1)
    return v


DB_URL = getenv("POSTGRES_URL")
BOT_TOKEN = getenv("BOT_TOKEN")
WEBAPP_URL = os.getenv("WEBAPP_URL", "http://localhost:5173")
DB_CONN = create_connection(DB_URL)


# ---------------- i18n ----------------
i18n = I18n(
    path="locales",
    default_locale="en",
    domain="bot",
)


class MyI18nMiddleware(I18nMiddleware):
    async def get_locale(self, event, data) -> str:
        user = data.get("event_from_user")
        return user.language_code if user and user.language_code else "en"


# ---------------- router ----------------
router = Router()


async def auth_confirm(token: UUID, telegram_user_id: int) -> bool:
    if DB_CONN is None:
        logger.error("Database connection is not available.")
        return False

    try:
        rows = execute(
            DB_CONN,
            "SELECT user_id, success FROM tg_auth_user WHERE id = %s",
            (str(token),),
            fetch=True,
        )
    except Exception as exc:
        logger.error("Failed to fetch auth request: %s", exc)
        return False

    if not rows:
        logger.info("No auth request found for token %s", token)
        return False

    user_id, success = rows[0]
    if success:
        logger.info("Auth token %s already used", token)
        return False

    try:
        execute(
            DB_CONN,
            "UPDATE tg_auth_user SET tg_user_id = %s, success = TRUE WHERE id = %s",
            (telegram_user_id, str(token)),
        )
        execute(
            DB_CONN,
            "UPDATE app_user SET tg_user_id = %s WHERE id = %s",
            (telegram_user_id, str(user_id)),
        )
        execute(
            DB_CONN,
            "UPDATE user_permissions SET edit_own_participation = TRUE, edit_own_songs = TRUE WHERE user_id = %s",
            (str(user_id),),
        )
    except Exception as exc:
        logger.error("Failed to update auth linking for token %s: %s", token, exc)
        return False

    logger.info("Auth confirmed for token %s and telegram user %s", token, telegram_user_id)
    return True


# ---------------- handlers ----------------
@router.message(CommandStart(deep_link=True))
async def cmd_start_with_args(message: Message, command: CommandObject):
    """
    Handles:
      /start auth_<uuid>
    """
    args = command.args
    logger.info("Received command start with %s", args)

    if not args or not args.startswith("auth_"):
        await message.answer(_("Invalid start parameter."))
        return

    raw_uuid = args.removeprefix("auth_")

    try:
        token = UUID(raw_uuid)
    except ValueError:
        await message.answer(_("Invalid authentication token."))
        return

    ok = await auth_confirm(token, message.from_user.id)

    if ok:
        await message.answer(_("✅ Authentication successful! You may return to the web app."))
    else:
        await message.answer(_("❌ Authentication failed or expired."))


@router.message(CommandStart())
async def cmd_start(message: Message):
    logger.info("Received command /start without args")

    keyboard = InlineKeyboardMarkup(
        inline_keyboard=[
            [
                InlineKeyboardButton(
                    text="🎸 Открыть Music Club",
                    web_app=WebAppInfo(url=WEBAPP_URL)
                )
            ]
        ]
    )

    await message.answer(
        _("Добро пожаловать в Music Club! 🎸\n\nНажмите кнопку ниже, чтобы открыть приложение:"),
        reply_markup=keyboard
    )


@router.message(Command("help"))
async def cmd_help(message: Message):
    await message.answer(_("Send /start to get the webapp link."))


# ---------------- entrypoint ----------------
async def main():
    logging.basicConfig(
        level=os.environ.get("LOGLEVEL", "INFO").upper(),
        format="%(levelname)s:\t[%(asctime)s] - %(message)s",
    )
    bot = Bot(BOT_TOKEN)
    dp = Dispatcher()

    dp.message.middleware(MyI18nMiddleware(i18n))
    dp.include_router(router)

    logger.info("Starting polling for bot")
    logger.info("WebApp URL: %s", WEBAPP_URL)
    await dp.start_polling(bot)


if __name__ == "__main__":
    asyncio.run(main())
