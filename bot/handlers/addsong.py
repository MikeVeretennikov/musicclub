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

from bot.models import Song, PendingRole
from bot.services.database import get_db_session
from bot.services.settings import settings
from bot.services.strings import is_valid_title
from bot.services.url import parse_url
from bot.states.addsong import AddSong

router = Router()


async def on_title_input(
    message: Message,
    message_input: MessageInput,
    dialog_manager: DialogManager,
):
    if not is_valid_title(message.text):
        return
    dialog_manager.dialog_data["title"] = message.text
    await dialog_manager.next()


async def on_description_input(
    message: Message,
    message_input: MessageInput,
    dialog_manager: DialogManager,
):
    if not is_valid_title(message.text):
        return
    dialog_manager.dialog_data["description"] = message.text
    await dialog_manager.next()


async def on_description_skip(
    callback: CallbackQuery,
    button: Button,
    dialog_manager: DialogManager,
):
    dialog_manager.dialog_data["description"] = None
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


async def on_role_input(
    message: Message,
    message_input: MessageInput,
    dialog_manager: DialogManager,
):
    if not is_valid_title(message.text):
        return

    dialog_manager.dialog_data["roles"].append(message.text)


async def input_roles_getter(dialog_manager: DialogManager, **kwargs):
    if "roles" not in dialog_manager.dialog_data:
        dialog_manager.dialog_data["roles"] = []
    return {
        "can_remove_last": len(dialog_manager.dialog_data["roles"]) > 0,
        "roles": "\n".join(dialog_manager.dialog_data.get("roles", [])),
    }


async def remove_last_added_role(
    callback: CallbackQuery, button: Button, manager: DialogManager
):
    manager.dialog_data["roles"] = manager.dialog_data["roles"][:-1]


async def verify_info_getter(dialog_manager: DialogManager, **kwargs):
    return {
        "title": dialog_manager.dialog_data["title"],
        "link": dialog_manager.dialog_data["link"],
        "roles": "\n".join(dialog_manager.dialog_data.get("roles", [])),
    }


def generate_roles_text(roles: list[str]) -> str:
    return "\n".join(f"    - {role}" for role in roles)


async def add_song(
    callback: CallbackQuery, button: Button, dialog_manager: DialogManager
):
    async with get_db_session() as session:
        song = Song(
            title=dialog_manager.dialog_data["title"],
            link=dialog_manager.dialog_data["link"],
        )
        session.add(song)
        await session.commit()

        # Create PendingRole entries for each role
        for role in dialog_manager.dialog_data.get("roles", []):
            pending_role = PendingRole(song_id=song.id, role=role)
            session.add(pending_role)
        await session.commit()

    await callback.answer("Песня успешно создана")

    await callback.bot.send_message(
        chat_id=settings.CHAT_ID,
        text=(
            "Добавлена новая песня!\n\n"
            f"Название: <b>{dialog_manager.dialog_data['title']}\n</b>"
            f'<a href="{dialog_manager.dialog_data["link"]}">Ссылка на прослушивание</a>\n'
            f"\n{generate_roles_text(dialog_manager.dialog_data['roles'])}\n\n"
            f'<b><a href="https://t.me/{(await callback.bot.get_me()).username}?start={song.id}">Присоединиться</a></b>'
        ),
    )

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
            Const("Есть ли какие-то комментарии к песне?"),
            Button(
                Const("Пропустить"),
                id="skip_description",
                on_click=on_description_skip,
            ),
            Cancel(Const("Отмена")),
            MessageInput(
                content_types=ContentType.TEXT, func=on_description_input
            ),
            state=AddSong.description,
        ),
        Window(
            Const("Дай ссылку на песню"),
            Cancel(Const("Отмена")),
            MessageInput(content_types=ContentType.TEXT, func=on_link_input),
            state=AddSong.link,
        ),
        Window(
            Const(
                "А теперь давай пройдемся по тому, каких людей тебе нужно набрать на роли. Отправь название роли"
            ),
            Format("{roles}", when="roles"),
            MessageInput(
                content_types=ContentType.TEXT,
                func=on_role_input,
                id="input_seeking_role",
            ),
            Button(
                Const("Удалить последнюю роль"),
                id="pop_last",
                on_click=remove_last_added_role,
                when="can_remove_last",
            ),
            Button(
                Const("Дальше"),
                id="to_verify",
                on_click=lambda c, b, m: m.switch_to(AddSong.verify),
            ),
            Cancel(Const("Отмена")),
            getter=input_roles_getter,
            state=AddSong.add_role,
        ),
        Window(
            Const(
                "Уверен что хочешь добавить эту песню? Проверь информацию еще раз:"
            ),
            Format("Название: {title}\nСсылка: {link}\n"),
            Format("Роли, которые ищем:\n{roles}", when="roles"),
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
