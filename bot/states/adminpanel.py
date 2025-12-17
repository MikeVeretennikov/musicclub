from aiogram.fsm.state import State, StatesGroup


class AdminPanel(StatesGroup):
    menu = State()
    announcement = State()
