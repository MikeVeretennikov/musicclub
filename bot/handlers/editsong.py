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
from bot.services.role import is_valid_role
from bot.services.settings import settings
from bot.services.songparticipation import song_participation_list_out
from bot.services.url import parse_url
from bot.states.editrole import EditRole
from bot.states.editsong import EditSong

router = Router()
logger = logging.getLogger(__name__)


async def song_info_getter(dialog_manager: DialogManager, **kwargs) -> dict:
    async with get_db_session() as session:
        result = await session.execute(
            select(Song)
            .where(Song.id == int(dialog_manager.start_data["song_id"]))
            .limit(1)
        )
        song = result.scalar_one_or_none()
    return {
        "song_id": dialog_manager.start_data["song_id"],
        "song_title": song.title,
        "song_link": song.link,
    }


async def roles_info_getter(dialog_manager: DialogManager, **kwargs) -> dict:
    page = dialog_manager.dialog_data.get("page", 0)

    async with get_db_session() as session:
        result = await session.execute(
            select(SongParticipation).where(
                SongParticipation.song_id
                == int(dialog_manager.start_data["song_id"])
            )
        )
        participations: list[SongParticipation] = result.scalars().all()

    total_pages = max((len(participations) - 1) // settings.PAGE_SIZE + 1, 1)
    page %= total_pages
    start = page * settings.PAGE_SIZE
    end = start + settings.PAGE_SIZE
    dialog_manager.dialog_data["total_pages"] = total_pages

    return {
        **await song_info_getter(dialog_manager),
        "participations": await song_participation_list_out(
            participations[start:end]
        ),
        "page": page + 1,
        "total_pages": total_pages,
    }


async def join_as_getter(dialog_manager: DialogManager, **kwargs) -> dict:
    return {
        "role": dialog_manager.dialog_data["role"],
    }


async def next_page(c: CallbackQuery, b: Button, m: DialogManager):
    total_pages = m.dialog_data.get("total_pages", 1)
    page = m.dialog_data.get("page", 0)
    m.dialog_data["page"] = (page + 1) % total_pages
    await m.show()


async def prev_page(c: CallbackQuery, b: Button, m: DialogManager):
    total_pages = m.dialog_data.get("total_pages", 1)
    page = m.dialog_data.get("page", 0)
    m.dialog_data["page"] = (page - 1) % total_pages
    await m.show()


async def on_role_input(
    message: Message, msg_input: MessageInput, manager: DialogManager
):
    if not is_valid_role(message.text.strip()):
        await message.answer("Некорректная роль, попробуй еще раз")
        return
    manager.dialog_data["role"] = message.text.strip()
    await manager.switch_to(EditSong.confirm_join)


async def on_join(
    callback: CallbackQuery, button: Button, manager: DialogManager
):
    async with get_db_session() as session:
        result = (
            await session.execute(
                select(SongParticipation)
                .where(
                    SongParticipation.role == manager.dialog_data["role"],
                    SongParticipation.song_id
                    == int(manager.start_data["song_id"]),
                    SongParticipation.person_id == callback.from_user.id,
                )
                .limit(1)
            )
        ).scalar_one_or_none()
        if result:
            await callback.answer(
                "Вы уже участвуете в этой песне под этой ролью"
            )
            await manager.switch_to(EditSong.roles)
            return

        session.add(
            SongParticipation(
                person_id=callback.from_user.id,
                song_id=int(manager.start_data["song_id"]),
                role=manager.dialog_data["role"],
            )
        )
        await session.commit()
        await callback.answer("Ваше участие было записано")
        await manager.switch_to(EditSong.roles)


router.include_router(
    Dialog(
        Window(
            Format("ID: {song_id}\nНазвание: {song_title}"),
            Url(Const("Ссылка"), url=Format("{song_link}"), id="song_link"),
            Button(
                Const("Роли и присоединиться"),
                id="roles",
                on_click=lambda c, b, m: m.switch_to(EditSong.roles),
            ),
            Cancel(Const("Назад")),
            getter=song_info_getter,
            state=EditSong.menu,
        ),
        Window(
            Format("ID: {song_id}\nНазвание: {song_title}"),
            Column(
                Select(
                    Format("{item.who} - {item.role}"),
                    id="participation_select",
                    item_id_getter=lambda participation: f"{participation.participation_id}",
                    items="participations",
                    on_click=lambda c, b, m, i: m.start(
                        EditRole.menu, data={"participation_id": i}
                    ),
                ),
            ),
            Row(
                Button(Const("<"), id="prev", on_click=prev_page),
                Button(
                    Format("{page}/{total_pages}"),
                    id="pagecounter",
                    on_click=lambda c, b, m: c.answer("Мисклик"),
                ),
                Button(Const(">"), id="next", on_click=next_page),
            ),
            Button(
                Const("Присоединиться"),
                id="join",
                on_click=lambda c, b, m: m.switch_to(EditSong.join_as),
            ),
            Button(
                Const("Назад"),
                id="back",
                on_click=lambda c, b, m: m.switch_to(EditSong.menu),
            ),
            getter=roles_info_getter,
            state=EditSong.roles,
        ),
        Window(
            Const("В качестве какой роли хочешь присоединиться?"),
            MessageInput(
                content_types=ContentType.TEXT,
                id="join_as_role",
                func=on_role_input,
            ),
            Button(
                Const("Назад"),
                id="back",
                on_click=lambda c, b, m: m.switch_to(EditSong.roles),
            ),
            state=EditSong.join_as,
        ),
        Window(
            Format("Вы уверены что хотите присоединиться как <b>{role}</b>?"),
            Row(
                Button(Const("Да"), id="confirm", on_click=on_join),
                Button(
                    Const("Нет"),
                    id="deny",
                    on_click=lambda c, b, m: m.switch_to(EditSong.roles),
                ),
            ),
            getter=join_as_getter,
            state=EditSong.confirm_join,
        ),
    )
)
