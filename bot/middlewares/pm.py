from aiogram import BaseMiddleware
from aiogram.types import Message, CallbackQuery
from typing import Callable, Dict, Any, Awaitable


class PrivateChatOnlyMiddleware(BaseMiddleware):
    async def __call__(
        self,
        handler: Callable[[Any, Dict[str, Any]], Awaitable[Any]],
        event: Message | CallbackQuery,
        data: Dict[str, Any],
    ) -> Any:
        chat = event.chat if isinstance(event, Message) else event.message.chat
        if chat.type != "private":
            me = await event.bot.get_me()
            if isinstance(event, Message) and me.username in event.text:
                try:
                    await event.reply(
                        f"Бот работает только в приватных сообщениях.\n\nНапиши мне: @{me.username}"
                    )
                except:
                    pass
            return None

        return await handler(event, data)
