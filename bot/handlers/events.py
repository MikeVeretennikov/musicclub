import logging
from datetime import date, datetime

from aiogram import Router
from aiogram.enums import ContentType
from aiogram.types import User, CallbackQuery, Message
from aiogram_dialog import Dialog, Window, DialogManager, ShowMode
from aiogram_dialog.widgets.input import MessageInput
from aiogram_dialog.widgets.text import Const, Format
from aiogram_dialog.widgets.kbd import (
    Button,
    Row,
    Column,
    Cancel,
    Url,
    Calendar,
)
from aiogram_dialog.widgets.kbd import ScrollingGroup, Select
from sqlalchemy import select, delete
from sqlalchemy.orm.sync import update
from sqlalchemy.orm import selectinload


from bot.models import Song, SongParticipation, Person, Concert, TracklistEntry
from bot.services.database import get_db_session
from bot.services.songs import (
    get_paginated_songs,
    prev_page,
    next_page,
    get_verbose_tracklist,
)
from bot.services.strings import is_valid_title
from bot.services.settings import settings
from bot.services.songparticipation import song_participation_list_out
from bot.services.url import parse_url
from bot.states.createevent import CreateEvent
from bot.states.participations import MyParticipations

router = Router()


async def tracklist_getter(dialog_manager: DialogManager, **kwargs):
    if "tracklist" not in dialog_manager.dialog_data:
        dialog_manager.dialog_data["tracklist"] = []

    return {
        "current_index": len(dialog_manager.dialog_data["tracklist"]) + 1,
        **await get_paginated_songs(dialog_manager),
        **await get_verbose_tracklist(dialog_manager),
    }


async def confirm_getter(dialog_manager: DialogManager, **kwargs):
    return {
        **await get_verbose_tracklist(dialog_manager),
        "date": dialog_manager.dialog_data["date"],
    }


async def on_title_input(
    message: Message, message_input: MessageInput, manager: DialogManager
):
    if message.from_user.id != manager.start_data["started_id"]:
        return
    if not is_valid_title(message.text):
        return
    manager.dialog_data["title"] = message.text
    await manager.switch_to(CreateEvent.tracklist_enable)


async def on_tracklist_enable(
    callback: CallbackQuery, button: Button, manager: DialogManager
):
    manager.dialog_data["tracklist_enabled"] = True
    await manager.switch_to(CreateEvent.add_song_to_tracklist)


async def on_tracklist_disable(
    callback: CallbackQuery, button: Button, manager: DialogManager
):
    manager.dialog_data["tracklist_enabled"] = False
    await manager.switch_to(CreateEvent.date)


async def on_date_input(
    callback: CallbackQuery,
    widget,
    manager: DialogManager,
    selected_date: date,
):
    manager.dialog_data["date"] = selected_date.isoformat()
    await manager.switch_to(CreateEvent.confirm)


async def on_song_picked(
    callback: CallbackQuery,
    button: Button,
    manager: DialogManager,
    song_id: str,
):
    if "tracklist" not in manager.dialog_data:
        manager.dialog_data["tracklist"] = []
    manager.dialog_data["tracklist"].append(int(song_id))
    await callback.answer("Окей, добавил в список")
    await manager.show()


async def on_confirm_tracklist(
    callback: CallbackQuery, button: Button, manager: DialogManager
):
    await callback.answer("Окей, финализировал список")
    await manager.switch_to(CreateEvent.date)


async def on_confirm_event(
    callback: CallbackQuery, button: Button, manager: DialogManager
):
    async with get_db_session() as session:
        concert = Concert(
            name=manager.dialog_data["title"],
            date=date.fromisoformat(manager.dialog_data["date"]),
        )
        session.add(concert)
        await session.commit()
        entries = []
        for n, track_id in enumerate(manager.dialog_data["tracklist"]):
            entries.append(
                TracklistEntry(
                    concert_id=concert.id,
                    position=n,
                    song_id=track_id,
                )
            )
        session.add_all(entries)
        await session.commit()
    await callback.answer("Событие создано успешно")
    await manager.done()


router.include_router(
    Dialog(
        Window(
            Const("Какое название у события?"),
            MessageInput(func=on_title_input, content_types=ContentType.TEXT),
            Cancel(Const("Отмена")),
            state=CreateEvent.title,
        ),
        Window(
            Const("Будет ли у события треклист?"),
            Button(
                Const("Будет"), id="tracklist_on", on_click=on_tracklist_enable
            ),
            Button(
                Const("Не будет"),
                id="tracklist_off",
                on_click=on_tracklist_disable,
            ),
            Cancel(Const("Отмена")),
            state=CreateEvent.tracklist_enable,
        ),
        Window(
            Format("Выбери песню которую поставить на {current_index} место"),
            Format("{verbose_tracklist}"),
            Column(
                Select(
                    Format("{item.title}"),
                    id="song_select",
                    item_id_getter=lambda song: song.id,
                    items="songs",
                    on_click=on_song_picked,
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
                Const("Подтвердить"),
                id="confirm_tracklist",
                on_click=on_confirm_tracklist,
            ),
            Cancel(Const("Отмена")),
            getter=tracklist_getter,
            state=CreateEvent.add_song_to_tracklist,
        ),
        Window(
            Const("Когда будет событие происходить?"),
            Calendar(id="event_date", on_click=on_date_input),
            Cancel(Const("Отмена")),
            state=CreateEvent.date,
        ),
        Window(
            Const("Давай еще раз пройдемся по введенным данным"),
            Format("\n{verbose_tracklist}"),
            Format("\nДата: {date}\n"),
            Button(
                Const("Подтвердить"),
                id="confirm_event",
                on_click=on_confirm_event,
            ),
            Cancel(Const("Отмена")),
            getter=confirm_getter,
            state=CreateEvent.confirm,
        ),
    )
)
