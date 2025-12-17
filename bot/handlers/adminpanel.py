from aiogram import Router
from aiogram.types import Message, User
from aiogram_dialog import Dialog, DialogManager, Window
from aiogram_dialog.widgets.input import MessageInput
from aiogram_dialog.widgets.kbd import Button, Cancel
from aiogram_dialog.widgets.text import Const
from sqlalchemy import select

from bot.models import Person
from bot.services.database import get_db_session
from bot.services.settings import settings
from bot.states.adminpanel import AdminPanel
from bot.states.createevent import CreateEvent

router = Router()


async def admin_panel_getter(dialog_manager: DialogManager, event_from_user: User, **kwargs):
    if event_from_user.id not in settings.ADMIN_IDS:
        await dialog_manager.done()
    return {}


async def on_announcement(message: Message, msg_input: MessageInput, manager: DialogManager):
    async with get_db_session() as session:
        users: list[Person] = (await session.execute(select(Person))).scalars().all()

    counter = 0
    for user in users:
        try:
            await message.copy_to(user.id)
            counter += 1
        except:  # noqa: E722, S112
            continue

    await message.reply(
        text=(
            f"Отправил сообщение {counter} пользователям. "
            f"Это {(counter / len(users)) * 100}% базы данных, в ней всего {len(users)}"
        )
    )
    await manager.switch_to(AdminPanel.menu)


router.include_router(
    Dialog(
        Window(
            Const("Админ панель"),
            Button(
                Const("Создать мероприятие"),
                id="create",
                on_click=lambda c, b, m: m.start(CreateEvent.title, data={"started_id": c.from_user.id}),
            ),
            Button(
                Const("Сделать объявление"),
                id="announcement",
                on_click=lambda c, b, m: m.switch_to(AdminPanel.announcement),
            ),
            Cancel(Const("Назад")),
            getter=admin_panel_getter,
            state=AdminPanel.menu,
        ),
        Window(
            Const("Сделать объявление: отправь мне сообщение и я разошлю его всем кого знаю"),
            MessageInput(func=on_announcement),
            Cancel(Const("Назад")),
            getter=admin_panel_getter,
            state=AdminPanel.announcement,
        ),
    )
)
