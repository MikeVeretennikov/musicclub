from dataclasses import dataclass


@dataclass
class SongParticipationOut:
    id: int
    who: str
    role: str
