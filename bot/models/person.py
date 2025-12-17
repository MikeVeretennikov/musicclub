from sqlalchemy import BigInteger, Column, String
from sqlalchemy.orm import relationship

from bot.models import Base


class Person(Base):
    __tablename__ = "people"

    id = Column(BigInteger, primary_key=True)
    name = Column(String(100), nullable=False)

    participations = relationship("SongParticipation", back_populates="person")

    def __repr__(self) -> str:
        return f"<Person(name={self.name})>"
