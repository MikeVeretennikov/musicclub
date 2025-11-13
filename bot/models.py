from sqlalchemy import (
    Column,
    Integer,
    String,
    ForeignKey,
    DateTime,
    Enum,
    Table,
    UniqueConstraint,
    BigInteger, Date,
)
from sqlalchemy.orm import relationship, declarative_base
from sqlalchemy.sql import func

Base = declarative_base()


class Person(Base):
    __tablename__ = "people"

    id = Column(BigInteger, primary_key=True)
    name = Column(String(100), nullable=False)

    participations = relationship("SongParticipation", back_populates="person")

    def __repr__(self):
        return f"<Person(name={self.name})>"


class Song(Base):
    __tablename__ = "songs"

    id = Column(Integer, primary_key=True)
    title = Column(String(200), nullable=False)
    link = Column(String(200), nullable=True)

    participations = relationship("SongParticipation", back_populates="song")
    tracklist_entries = relationship("TracklistEntry", back_populates="song")

    def __repr__(self):
        return f"<Song(title={self.title})>"


class SongParticipation(Base):
    __tablename__ = "song_participations"

    id = Column(Integer, primary_key=True)
    song_id = Column(Integer, ForeignKey("songs.id"), nullable=False)
    person_id = Column(Integer, ForeignKey("people.id"), nullable=False)
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

    def __repr__(self):
        return f"<SongParticipation(song={self.song.title}, person={self.person.name}, role={self.role})>"


class Concert(Base):
    __tablename__ = "concerts"

    id = Column(Integer, primary_key=True)
    name = Column(String(150), nullable=False)
    date = Column(Date(), server_default=func.now())

    tracklist = relationship(
        "TracklistEntry",
        back_populates="concert",
        order_by="TracklistEntry.position",
    )

    def __repr__(self):
        return f"<Concert(name={self.name}, date={self.date})>"


class TracklistEntry(Base):
    __tablename__ = "tracklist_entries"

    id = Column(Integer, primary_key=True)
    concert_id = Column(Integer, ForeignKey("concerts.id"), nullable=False)
    song_id = Column(Integer, ForeignKey("songs.id"), nullable=False)
    position = Column(Integer, nullable=False)

    concert = relationship("Concert", back_populates="tracklist")
    song = relationship("Song", back_populates="tracklist_entries")

    __table_args__ = (
        UniqueConstraint(
            "concert_id", "position", name="unique_song_position_per_concert"
        ),
    )

    def __repr__(self):
        return f"<TracklistEntry(concert={self.concert.name}, position={self.position}, song={self.song.title})>"
