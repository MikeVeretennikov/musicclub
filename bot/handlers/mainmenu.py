import logging
from datetime import date, datetime

from aiogram import Router
from aiogram.types import User, CallbackQuery
from aiogram_dialog import Dialog, Window, DialogManager
from aiogram_dialog.widgets.text import Const, Format
from aiogram_dialog.widgets.kbd import Button, Row, Column
from aiogram_dialog.widgets.kbd import ScrollingGroup, Select
from sqlalchemy import select
from sqlalchemy.orm import selectinload

from bot.models import Song, Concert, PendingRole, SongParticipation, Person
from bot.services.database import get_db_session
from bot.services.settings import settings
from bot.services.songs import get_paginated_songs, prev_page, next_page
from bot.states.addsong import AddSong
from bot.states.adminpanel import AdminPanel
from bot.states.concert import ConcertInfo
from bot.states.editsong import EditSong
from bot.states.mainmenu import MainMenu
from bot.states.participations import MyParticipations

router = Router()


async def main_menu_getter(event_from_user: User, **kwargs):
    return {
        "is_admin": event_from_user.id in settings.ADMIN_IDS,
    }


async def songs_getter(dialog_manager: DialogManager, **kwargs):
    """Fetch paginated songs for current page."""
    return {
        **await get_paginated_songs(dialog_manager),
    }


async def events_getter(
    dialog_manager: DialogManager, event_from_user: User, **kwargs
):
    page = dialog_manager.dialog_data.get("page", 0)

    async with get_db_session() as session:
        concerts: list[Concert] = (
            (
                await session.execute(
                    select(Concert)
                    .where(Concert.date >= datetime.now().date())
                    .order_by(Concert.id)
                )
            )
            .scalars()
            .all()
        )

    total_pages = max((len(concerts) - 1) // settings.PAGE_SIZE + 1, 1)
    page %= total_pages
    start = page * settings.PAGE_SIZE
    end = start + settings.PAGE_SIZE
    dialog_manager.dialog_data["total_pages"] = total_pages

    return {
        "events": concerts[start:end],
        "page": page + 1,
        "total_pages": total_pages,
        "is_admin": event_from_user.id in settings.ADMIN_IDS,
    }


async def vacant_positions_getter(
    dialog_manager: DialogManager, event_from_user: User, **kwargs
):
    """Fetch paginated vacant positions (pending roles)."""
    page = dialog_manager.dialog_data.get("page", 0)

    async with get_db_session() as session:
        pending_roles: list[PendingRole] = (
            (
                await session.execute(
                    select(PendingRole)
                    .options(selectinload(PendingRole.song))
                    .order_by(PendingRole.created_at.desc())
                )
            )
            .scalars()
            .all()
        )

    total_pages = max((len(pending_roles) - 1) // settings.PAGE_SIZE + 1, 1)
    page %= total_pages
    start = page * settings.PAGE_SIZE
    end = start + settings.PAGE_SIZE
    dialog_manager.dialog_data["total_pages"] = total_pages
    dialog_manager.dialog_data["page"] = page

    return {
        "pending_roles": pending_roles[start:end],
        "page": page + 1,
        "total_pages": total_pages,
    }


async def on_pending_role_selected(
    callback: CallbackQuery,
    button: Button,
    manager: DialogManager,
    item_id: str,
):
    async with get_db_session() as session:
        pending_role: PendingRole = (
            await session.execute(
                select(PendingRole).where(PendingRole.id == int(item_id))
            )
        ).scalar_one_or_none()
        await session.delete(pending_role)
        session.add(
            SongParticipation(
                person_id=callback.from_user.id,
                song_id=pending_role.song_id,
                role=pending_role.role,
            )
        )
        await session.commit()
        await callback.answer("Ваше участие было записано")
    await manager.switch_to(MainMenu.menu)
    await manager.start(MyParticipations.menu)


router.include_router(
    Dialog(
        Window(
            Const("<b>Главное меню</b>\n\nЧто желаешь поделать сегодня?\n"),
            Const("<b>Ты админ, кстати</b>\n", when="is_admin"),
            Button(
                Const("Админ-панель"),
                id="admin_panel",
                when="is_admin",
                on_click=lambda c, b, m: m.start(AdminPanel.menu),
            ),
            Button(
                Const("Песни"),
                id="songs",
                on_click=lambda c, b, m: m.switch_to(MainMenu.songs),
            ),
            Button(
                Const("Мои участия"),
                id="participations",
                on_click=lambda c, b, m: m.start(MyParticipations.menu),
            ),
            Button(
                Const("Открытые позиции"),
                id="positions",
                on_click=lambda c, b, m: m.switch_to(
                    MainMenu.vacant_positions
                ),
            ),
            Button(
                Const("Ближайшие мероприятия"),
                id="events",
                on_click=lambda c, b, m: m.switch_to(MainMenu.events),
            ),
            getter=main_menu_getter,
            state=MainMenu.menu,
        ),
        Window(
            Const("<b>Вот список песен</b>\n"),
            Column(
                Select(
                    Format("{item.title}"),
                    id="song_select",
                    item_id_getter=lambda song: song.id,
                    items="songs",
                    on_click=lambda c, b, m, item_id: m.start(
                        EditSong.menu, data={"song_id": item_id}
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
                Const("Добавить песню"),
                id="add_song",
                on_click=lambda c, b, m: m.start(AddSong.title),
            ),
            Button(
                Const("Назад"),
                id="Back",
                on_click=lambda c, b, m: m.switch_to(MainMenu.menu),
            ),
            getter=songs_getter,
            state=MainMenu.songs,
        ),
        Window(
            Const("Вот доступные позиции"),
            Column(
                Select(
                    Format("{item.song.title} - {item.role}"),
                    id="roles_select",
                    item_id_getter=lambda role: role.id,
                    items="pending_roles",
                    on_click=on_pending_role_selected,
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
                Const("Назад"),
                id="back",
                on_click=lambda c, b, m: m.switch_to(MainMenu.menu),
            ),
            getter=vacant_positions_getter,
            state=MainMenu.vacant_positions,
        ),
        Window(
            Const("Вот ближайшие мероприятия"),
            Column(
                Select(
                    Format("{item.name}"),
                    id="event_select",
                    item_id_getter=lambda event: event.id,
                    items="events",
                    on_click=lambda c, b, m, i: m.start(
                        ConcertInfo.menu, data={"concert_id": i}
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
                Const("Назад"),
                id="back",
                on_click=lambda c, b, m: m.switch_to(MainMenu.menu),
            ),
            getter=events_getter,
            state=MainMenu.events,
        ),
    )
)
