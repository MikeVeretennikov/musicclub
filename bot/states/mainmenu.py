from aiogram.fsm.state import State, StatesGroup


class MainMenu(StatesGroup):
    menu = State()
    songs = State()
    events = State()
    vacant_positions = State()
