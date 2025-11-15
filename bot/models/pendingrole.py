from sqlalchemy import Column, Integer, String, ForeignKey, DateTime, func
from sqlalchemy.orm import relationship

from bot.models import Base


class PendingRole(Base):
    __tablename__ = "pending_roles"

    id = Column(Integer, primary_key=True)
    song_id = Column(Integer, ForeignKey("songs.id"), nullable=False)
    role = Column(String(200), nullable=False)
    created_at = Column(DateTime, server_default=func.now(), nullable=False)

    song = relationship("Song", backref="pending_roles")

    def __repr__(self):
        return f"<PendingRole(song_id={self.song_id}, role={self.role})>"
