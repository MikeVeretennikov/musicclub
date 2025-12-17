from aiogram.filters import BaseFilter
from aiogram.types import ChatMember, Message


class ChatMemberFilter(BaseFilter):
    def __init__(self, chat_id: int) -> None:
        self.chat_id = chat_id

    async def __call__(self, message: Message, **kwargs):  # noqa: ANN204
        member: ChatMember = await message.bot.get_chat_member(chat_id=self.chat_id, user_id=message.from_user.id)
        return member.status != "left"
