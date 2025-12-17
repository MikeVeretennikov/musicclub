from aiogram.fsm.state import State, StatesGroup


class CreateEvent(StatesGroup):
    title = State()
    tracklist_enable = State()
    add_song_to_tracklist = State()
    date = State()
    confirm = State()
