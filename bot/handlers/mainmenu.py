import logging

from aiogram import Router
from aiogram.types import User, CallbackQuery
from aiogram_dialog import Dialog, Window, DialogManager
from aiogram_dialog.widgets.text import Const, Format
from aiogram_dialog.widgets.kbd import Button, Row, Column
from aiogram_dialog.widgets.kbd import ScrollingGroup, Select
from sqlalchemy import select

from bot.models import Song
from bot.services.database import get_db_session
from bot.services.settings import settings
from bot.services.songs import get_paginated_songs, prev_page, next_page
from bot.states.addsong import AddSong
from bot.states.adminpanel import AdminPanel
from bot.states.editsong import EditSong
from bot.states.mainmenu import MainMenu
from bot.states.participations import MyParticipations

router = Router()


# ----- Getters -----
async def main_menu_getter(event_from_user: User, **kwargs):
    return {
        "is_admin": event_from_user.id in settings.ADMIN_IDS,
        "chat_link": settings.CHAT_LINK,
    }


async def songs_getter(dialog_manager: DialogManager, **kwargs):
    """Fetch paginated songs for current page."""
    return {
        **await get_paginated_songs(dialog_manager),
    }


async def events_getter(dialog_manager: DialogManager, event_from_user: User, **kwargs):
    ...
    return {
        "is_admin": event_from_user.id in settings.ADMIN_IDS,
    }

# ----- Button Handlers -----
async def show_song(
    c: CallbackQuery, w: Button, m: DialogManager, item_id: str
):
    await m.start(EditSong.menu, data={"song_id": item_id})


# ----- Dialog Definition -----
router.include_router(
    Dialog(
        # --- Main menu ---
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
                Const("Ближайшие мероприятия"),
                id="events",
                on_click=lambda c, b, m: m.switch_to(MainMenu.events),
            ),
            getter=main_menu_getter,
            state=MainMenu.menu,
        ),
        # --- Songs list with pagination ---
        Window(
            Const("<b>Вот список песен</b>\n"),
            Column(
                Select(
                    Format("{item.title}"),
                    id="song_select",
                    item_id_getter=lambda song: song.id,
                    items="songs",
                    on_click=show_song,
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
        # --- Events placeholder ---
        Window(
            Const("Вот ближайшие мероприятия"),
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
