from aiogram import Router
from aiogram.types import CallbackQuery, User
from aiogram_dialog import Dialog, DialogManager, Window
from aiogram_dialog.widgets.kbd import Button, Cancel
from aiogram_dialog.widgets.text import Const, Format
from sqlalchemy import delete, select
from sqlalchemy.orm import selectinload

from bot.models import Concert
from bot.services.database import get_db_session
from bot.services.settings import settings
from bot.services.songs import get_verbose_tracklist
from bot.states.concert import ConcertInfo

router = Router()


async def concert_getter(dialog_manager: DialogManager, event_from_user: User, **kwargs) -> dict:
    concert_id = int(dialog_manager.start_data["concert_id"])
    async with get_db_session() as session:
        concert: Concert = (
            await session.execute(
                select(Concert).where(Concert.id == concert_id).options(selectinload(Concert.tracklist))
            )
        ).scalar_one_or_none()
        dialog_manager.dialog_data["tracklist"] = [track.id for track in concert.tracklist]
    return {
        "is_admin": event_from_user.id in settings.ADMIN_IDS,
        **await get_verbose_tracklist(dialog_manager),
        "name": concert.name,
        "date": concert.date.isoformat(),
    }


async def delete_concert(callback: CallbackQuery, button: Button, manager: DialogManager):
    concert_id = int(manager.start_data["concert_id"])
    async with get_db_session() as session:
        await session.execute(delete(Concert).where(Concert.id == concert_id))
        await session.commit()
    await callback.answer("Успешно удалил концерт")
    await manager.done()


router.include_router(
    Dialog(
        Window(
            Format("Название: <b>{name}</b>"),
            Format("Дата: <b>{date}</b>"),
            Format("\n{verbose_tracklist}"),
            Button(
                Const("Удалить концерт"),
                on_click=delete_concert,
                when="is_admin",
                id="delete_concert",
            ),
            Cancel(Const("Назад")),
            getter=concert_getter,
            state=ConcertInfo.menu,
        )
    )
)
