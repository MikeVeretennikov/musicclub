from sqlalchemy import Column, Date, Integer, String, func
from sqlalchemy.orm import relationship

from bot.models import Base


class Concert(Base):
    __tablename__ = "concerts"

    id = Column(Integer, primary_key=True)
    name = Column(String(150), nullable=False)
    date = Column(Date(), server_default=func.now())

    tracklist = relationship(
        "TracklistEntry",
        back_populates="concert",
        order_by="TracklistEntry.position",
        cascade="all, delete-orphan",
    )

    def __repr__(self) -> str:
        return f"<Concert(name={self.name}, date={self.date})>"
