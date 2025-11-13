from aiogram import Router
from aiogram.filters import CommandStart, CommandObject
from aiogram.types import Message
from aiogram_dialog import DialogManager
from sqlalchemy import select

from bot.models import Person
from bot.services.database import get_db_session
from bot.states.editsong import EditSong
from bot.states.mainmenu import MainMenu

router = Router()


@router.message(CommandStart())
async def start_command(
    message: Message, dialog_manager: DialogManager, command: CommandObject
) -> None:
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
        await dialog_manager.start(EditSong.menu, data={"song_id": int(command.args)})
        return
    await dialog_manager.start(MainMenu.menu)
