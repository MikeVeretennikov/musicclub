from aiogram.fsm.state import State, StatesGroup


class EditRole(StatesGroup):
    menu = State()
    remove_confirm = State()
    input_role = State()
