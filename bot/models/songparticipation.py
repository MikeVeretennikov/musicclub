from sqlalchemy import (
    BigInteger,
    Column,
    ForeignKey,
    Integer,
    String,
    UniqueConstraint,
)
from sqlalchemy.orm import relationship

from bot.models import Base


class SongParticipation(Base):
    __tablename__ = "song_participations"

    id = Column(Integer, primary_key=True)
    song_id = Column(Integer, ForeignKey("songs.id"), nullable=False)
    person_id = Column(BigInteger, ForeignKey("people.id"), nullable=False)
    role = Column(String(200), nullable=False)

    song = relationship("Song", back_populates="participations")
    person = relationship("Person", back_populates="participations")

    __table_args__ = (
        UniqueConstraint(
            "song_id",
            "person_id",
            "role",
            name="unique_song_role_per_person",
        ),
    )

    def __repr__(self) -> str:
        return f"<SongParticipation(song={self.song.title}, person={self.person.name}, role={self.role})>"
