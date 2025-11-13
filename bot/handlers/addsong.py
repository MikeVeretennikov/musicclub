import logging

from aiogram import Router
from aiogram.enums import ContentType
from aiogram.types import User, CallbackQuery, Message
from aiogram_dialog import Dialog, Window, DialogManager
from aiogram_dialog.widgets.input import MessageInput
from aiogram_dialog.widgets.text import Const, Format
from aiogram_dialog.widgets.kbd import Button, Row, Column, Cancel
from aiogram_dialog.widgets.kbd import ScrollingGroup, Select
from sqlalchemy import select

from bot.models import Song
from bot.services.database import get_db_session
from bot.services.settings import settings
from bot.services.url import parse_url
from bot.states.addsong import AddSong

router = Router()


async def on_title_input(
    message: Message,
    message_input: MessageInput,
    dialog_manager: DialogManager,
):
    dialog_manager.dialog_data["title"] = message.text
    await dialog_manager.next()


async def on_link_input(
    message: Message,
    message_input: MessageInput,
    dialog_manager: DialogManager,
):
    url = parse_url(message.text)
    if not url:
        return
    dialog_manager.dialog_data["link"] = url
    await dialog_manager.next()


async def verify_info_getter(dialog_manager: DialogManager, **kwargs):
    return {
        "title": dialog_manager.dialog_data["title"],
        "link": dialog_manager.dialog_data["link"],
    }


async def add_song(
    callback: CallbackQuery, button: Button, dialog_manager: DialogManager
):
    async with get_db_session() as session:
        session.add(
            Song(
                title=dialog_manager.dialog_data["title"],
                link=dialog_manager.dialog_data["link"],
            )
        )
        await session.commit()
    await callback.answer("Песня успешно создана")
    await dialog_manager.done()


router.include_router(
    Dialog(
        Window(
            Const("Как называется твоя песня?"),
            Cancel(Const("Отмена")),
            MessageInput(content_types=ContentType.TEXT, func=on_title_input),
            state=AddSong.title,
        ),
        Window(
            Const("Дай ссылку на песню"),
            Cancel(Const("Отмена")),
            MessageInput(content_types=ContentType.TEXT, func=on_link_input),
            state=AddSong.link,
        ),
        Window(
            Const(
                "Уверен что хочешь добавить эту песню? Проверь информацию еще раз:"
            ),
            Format("Название: {title}\nСсылка: {link}\n"),
            Row(
                Button(Const("Уверен"), id="confirm", on_click=add_song),
                Button(
                    Const("Не уверен"),
                    id="deny",
                    on_click=lambda c, b, m: m.switch_to(AddSong.title),
                ),
            ),
            MessageInput(content_types=ContentType.TEXT, func=on_title_input),
            getter=verify_info_getter,
            state=AddSong.verify,
        ),
    )
)
