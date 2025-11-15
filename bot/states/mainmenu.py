from aiogram.fsm.state import StatesGroup, State


class MainMenu(StatesGroup):
    menu = State()
    songs = State()
    events = State()
    vacant_positions = State()
