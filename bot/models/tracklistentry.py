from sqlalchemy import Column, ForeignKey, Integer, UniqueConstraint
from sqlalchemy.orm import relationship

from bot.models import Base


class TracklistEntry(Base):
    __tablename__ = "tracklist_entries"

    id = Column(Integer, primary_key=True)
    concert_id = Column(
        Integer,
        ForeignKey("concerts.id", ondelete="CASCADE"),
        nullable=False,
    )
    song_id = Column(Integer, ForeignKey("songs.id"), nullable=False)
    position = Column(Integer, nullable=False)

    concert = relationship("Concert", back_populates="tracklist")
    song = relationship("Song", back_populates="tracklist_entries")

    __table_args__ = (UniqueConstraint("concert_id", "position", name="unique_song_position_per_concert"),)

    def __repr__(self) -> str:
        return f"<TracklistEntry(concert={self.concert.name}, position={self.position}, song={self.song.title})>"
