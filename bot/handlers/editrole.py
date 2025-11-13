import logging

from aiogram import Router
from aiogram.enums import ContentType
from aiogram.types import User, CallbackQuery, Message
from aiogram_dialog import Dialog, Window, DialogManager
from aiogram_dialog.widgets.input import MessageInput
from aiogram_dialog.widgets.text import Const, Format
from aiogram_dialog.widgets.kbd import Button, Row, Column, Cancel, Url
from aiogram_dialog.widgets.kbd import ScrollingGroup, Select
from sqlalchemy import select

from bot.models import Song, SongParticipation, Person
from bot.services.database import get_db_session
from bot.services.settings import settings
from bot.services.url import parse_url
from bot.states.editrole import EditRole

router = Router()


async def main_getter(dialog_manager: DialogManager, **kwargs) -> dict:
    async with get_db_session() as session:
        participation: SongParticipation = (
            await session.execute(
                select(SongParticipation).where(
                    SongParticipation.id
                    == int(dialog_manager.start_data["participation_id"])
                )
            )
        ).scalar_one_or_none()
        person: Person = (
            await session.execute(
                select(Person).where(Person.id == participation.person_id)
            )
        ).scalar_one_or_none()
        song: Song = (
            await session.execute(
                select(Song).where(Song.id == participation.song_id)
            )
        ).scalar_one_or_none()
    return {
        "participation_id": dialog_manager.start_data["participation_id"],
        "person_id": person.id,
        "person_name": person.name,
        "song_title": song.title,
        "role": participation.role,
    }


router.include_router(
    Dialog(
        Window(
            Format(
                "<b>{person_name}</b>\nв <b>{song_title}</b>\nкак <b>{role}</b>"
            ),
            Url(
                Const("Перейти в профиль"), Format("tg://user?id={person_id}")
            ),
            Cancel(Const("Назад")),
            getter=main_getter,
            state=EditRole.menu,
        )
    )
)
