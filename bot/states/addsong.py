from aiogram.fsm.state import State, StatesGroup


class AddSong(StatesGroup):
    title = State()
    description = State()
    link = State()
    verify = State()
    add_role = State()
