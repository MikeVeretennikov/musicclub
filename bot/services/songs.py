from aiogram.types import CallbackQuery
from aiogram_dialog.widgets.kbd import Button
from sqlalchemy import select


from bot.models import Song
from bot.services.database import get_db_session
from bot.services.settings import settings
from aiogram_dialog import DialogManager


async def get_paginated_songs(dialog_manager: DialogManager) -> dict:
    page = dialog_manager.dialog_data.get("page", 0)

    async with get_db_session() as session:
        result = await session.execute(select(Song).order_by(Song.id))
        songs = result.scalars().all()

    total_pages = max((len(songs) - 1) // settings.PAGE_SIZE + 1, 1)
    page %= total_pages
    start = page * settings.PAGE_SIZE
    end = start + settings.PAGE_SIZE
    dialog_manager.dialog_data["total_pages"] = total_pages

    return {
        "songs": songs[start:end],
        "page": page + 1,
        "total_pages": total_pages,
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

