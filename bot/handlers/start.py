from aiogram import Router
from aiogram.filters import CommandStart, CommandObject
from aiogram.fsm.state import StatesGroup, State
from aiogram.types import Message
from aiogram_dialog import DialogManager, Dialog, Window
from aiogram_dialog.widgets.kbd import Url
from aiogram_dialog.widgets.text import Const, Format
from sqlalchemy import select

from bot.filters.ChatMemberFilter import ChatMemberFilter
from bot.models import Person
from bot.services.database import get_db_session
from bot.services.invite_link import ChatInviteLinkManager
from bot.services.settings import settings
from bot.states.editsong import EditSong
from bot.states.mainmenu import MainMenu

router = Router()


@router.message(CommandStart(), ChatMemberFilter(chat_id=settings.CHAT_ID))
async def start_command(
    message: Message, dialog_manager: DialogManager, command: CommandObject
) -> None:
    await dialog_manager.reset_stack()
    async with get_db_session() as session:
        stmt = select(Person).where(Person.id == message.from_user.id)
        result = await session.execute(stmt)
        if not result.scalar_one_or_none():
            person = Person(
                id=message.from_user.id, name=message.from_user.full_name
            )
            session.add(person)
            await session.commit()
            await message.answer(f"Welcome, to the club, buddy!")

    if command.args:
        await dialog_manager.start(
            EditSong.menu, data={"song_id": int(command.args)}
        )
        return
    await dialog_manager.start(MainMenu.menu)


@router.message(CommandStart(), ~ChatMemberFilter(chat_id=settings.CHAT_ID))
async def start_command_not_member(
    message: Message, dialog_manager: DialogManager
) -> None:
    await dialog_manager.reset_stack()
    await dialog_manager.start(ChatInviteStates.invite)


async def not_member_dialog_getter(dialog_manager: DialogManager, **kwargs):
    return {"invite_link": ChatInviteLinkManager.link.invite_link}


class ChatInviteStates(StatesGroup):
    invite = State()


router.include_router(
    Dialog(
        Window(
            Const(
                "Вы должны быть участником приватного чата чтоб использовать этого бота"
            ),
            Url(url=Format("{invite_link}"), text=Const("Вступить в чат")),
            getter=not_member_dialog_getter,
            state=ChatInviteStates.invite,
        )
    )
)
