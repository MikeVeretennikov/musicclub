from pydantic import BaseModel


class SongParticipationOut(BaseModel):
    participation_id: int
    person_id: int
    song_id: int
    who: str
    role: str
