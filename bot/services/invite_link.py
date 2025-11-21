from aiogram import Bot
from aiogram.types import ChatInviteLink


class ChatInviteLinkManager:
    chat_id: int
    link: ChatInviteLink

    bot: Bot

    def __init__(self, bot: Bot, chat_id: int):
        ChatInviteLinkManager.bot = bot
        ChatInviteLinkManager.chat_id = chat_id

    async def startup(self):
        ChatInviteLinkManager.link = await self.bot.create_chat_invite_link(
            chat_id=self.chat_id, name="invite", creates_join_request=True
        )

    async def shutdown(self):
        await self.bot.revoke_chat_invite_link(
            chat_id=self.chat_id, invite_link=self.link.invite_link
        )
