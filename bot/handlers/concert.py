import logging

from aiogram import Router
from aiogram.enums import ContentType
from aiogram.types import User, CallbackQuery, Message
from aiogram_dialog import Dialog, Window, DialogManager
from aiogram_dialog.widgets.input import MessageInput
from aiogram_dialog.widgets.text import Const, Format
from aiogram_dialog.widgets.kbd import Button, Row, Column, Cancel, Url
from aiogram_dialog.widgets.kbd import ScrollingGroup, Select
from sqlalchemy import select, delete
from sqlalchemy.orm.sync import update
from sqlalchemy.orm import selectinload


from bot.models import Song, SongParticipation, Person, Concert, TracklistEntry
from bot.services.database import get_db_session
from bot.services.settings import settings
from bot.services.songparticipation import song_participation_list_out
from bot.services.songs import prev_page, next_page, get_verbose_tracklist
from bot.services.url import parse_url
from bot.states.concert import ConcertInfo

router = Router()


async def concert_getter(dialog_manager: DialogManager, event_from_user: User, **kwargs) -> dict:
    concert_id = int(dialog_manager.start_data["concert_id"])
    async with get_db_session() as session:
        concert: Concert = (
            await session.execute(
                select(Concert)
                .where(Concert.id == concert_id)
                .options(selectinload(Concert.tracklist))
            )
        ).scalar_one_or_none()
        dialog_manager.dialog_data["tracklist"] = [
            track.id for track in concert.tracklist
        ]
    return {
        "is_admin": event_from_user.id in settings.ADMIN_IDS,
        **await get_verbose_tracklist(dialog_manager),
        "name": concert.name,
        "date": concert.date.isoformat(),
    }

async def delete_concert(callback: CallbackQuery, button: Button, manager: DialogManager):
    concert_id = int(manager.start_data["concert_id"])
    async with get_db_session() as session:
        await session.execute(
            delete(Concert)
            .where(Concert.id == concert_id)
        )
        await session.commit()
    await callback.answer("Успешно удалил концерт")
    await manager.done()


router.include_router(
    Dialog(
        Window(
            Format("Название: <b>{name}</b>"),
            Format("Дата: <b>{date}</b>"),
            Format("\n{verbose_tracklist}"),
            Button(Const("Удалить концерт"), on_click=delete_concert, when="is_admin", id="delete_concert"),
            Cancel(Const("Назад")),
            getter=concert_getter,
            state=ConcertInfo.menu,
        )
    )
)
