from aiogram.fsm.state import StatesGroup, State


class CreateEvent(StatesGroup):
    title = State()
    tracklist_enable = State()
    add_song_to_tracklist = State()
    date = State()
    confirm = State()
