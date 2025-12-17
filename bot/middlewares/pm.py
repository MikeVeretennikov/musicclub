import builtins
import contextlib
from collections.abc import Awaitable, Callable
from typing import Any

from aiogram import BaseMiddleware
from aiogram.types import CallbackQuery, Message


class PrivateChatOnlyMiddleware(BaseMiddleware):
    async def __call__(
        self,
        handler: Callable[[Any, dict[str, Any]], Awaitable[Any]],
        event: Message | CallbackQuery,
        data: dict[str, Any],
    ) -> Any:
        chat = event.chat if isinstance(event, Message) else event.message.chat
        if chat.type != "private":
            me = await event.bot.get_me()
            if isinstance(event, Message) and me.username in event.text:
                with contextlib.suppress(builtins.BaseException):
                    await event.reply(f"Бот работает только в приватных сообщениях.\n\nНапиши мне: @{me.username}")
            return None

        return await handler(event, data)
