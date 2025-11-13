from sqlalchemy import select

from bot.models import SongParticipation, Person, Song
from bot.schemas import SongParticipationOut
from bot.services.database import get_db_session


async def song_participation_list_out(
    arr: list[SongParticipation],
) -> list[SongParticipationOut] | None:
    person_ids = {p.person_id for p in arr}
    async with get_db_session() as session:
        result = await session.execute(
            select(Person).where(Person.id.in_(person_ids))
        )
        persons = {p.id: p for p in result.scalars().all()}

    if len(persons) != len(person_ids):
        return None

    return [
        SongParticipationOut(
            participation_id=part.id,
            person_id=part.person_id,
            song_id=part.song_id,
            who=persons[part.person_id].name,
            role=part.role,
        )
        for part in arr
    ]
