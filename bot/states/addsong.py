from aiogram.fsm.state import StatesGroup, State


class AddSong(StatesGroup):
    title = State()
    description = State()
    link = State()
    verify = State()
    add_role = State()
