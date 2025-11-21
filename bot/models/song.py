from sqlalchemy import Column, Integer, String
from sqlalchemy.orm import relationship

from bot.models import Base


class Song(Base):
    __tablename__ = "songs"

    id = Column(Integer, primary_key=True)
    title = Column(String(200), nullable=False)
    description = Column(String(500), nullable=True)
    link = Column(String(200), nullable=True)

    participations = relationship("SongParticipation", back_populates="song")
    tracklist_entries = relationship("TracklistEntry", back_populates="song")

    def __repr__(self):
        return f"<Song(title={self.title})>"
