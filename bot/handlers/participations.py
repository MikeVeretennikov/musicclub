from aiogram import Router
from aiogram.types import User
from aiogram_dialog import Dialog, DialogManager, Window
from aiogram_dialog.widgets.kbd import Button, Cancel, Column, Row, Select
from aiogram_dialog.widgets.text import Const, Format
from sqlalchemy import select
from sqlalchemy.orm import selectinload

from bot.models import SongParticipation
from bot.services.database import get_db_session
from bot.services.settings import settings
from bot.services.songparticipation import song_participation_list_out
from bot.services.songs import next_page, prev_page
from bot.states.editrole import EditRole
from bot.states.participations import MyParticipations

router = Router()


async def participations_getter(dialog_manager: DialogManager, event_from_user: User, **kwargs):
    """Fetch paginated songs for current page."""
    page = dialog_manager.dialog_data.get("page", 0)

    async with get_db_session() as session:
        result = await session.execute(
            select(SongParticipation)
            .where(SongParticipation.person_id == event_from_user.id)
            .options(
                selectinload(SongParticipation.song),
            )
            .order_by(SongParticipation.id)
        )
        participations = result.scalars().all()

    total_pages = max((len(participations) - 1) // settings.PAGE_SIZE + 1, 1)
    page %= total_pages
    start = page * settings.PAGE_SIZE
    end = start + settings.PAGE_SIZE
    dialog_manager.dialog_data["total_pages"] = total_pages

    return {
        "participations": await song_participation_list_out(participations[start:end]),
        "page": page + 1,
        "total_pages": total_pages,
    }


router.include_router(
    Dialog(
        Window(
            Const("Вот все места, в которых вы участвуете"),
            Column(
                Select(
                    Format("{item.where} - {item.role}"),
                    id="participation_select",
                    item_id_getter=lambda participation: f"{participation.participation_id}",
                    items="participations",
                    on_click=lambda c, b, m, i: m.start(
                        EditRole.menu,
                        data={"participation_id": i, "notify": False},
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
            Cancel(Const("Назад")),
            getter=participations_getter,
            state=MyParticipations.menu,
        )
    )
)
