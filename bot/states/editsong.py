from aiogram.fsm.state import State, StatesGroup


class EditSong(StatesGroup):
    menu = State()
    roles = State()
    join_as = State()
    confirm_join = State()
