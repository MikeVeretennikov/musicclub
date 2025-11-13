import re
from html import unescape

SAFE_PATTERN = re.compile(r"^[\w\s\-\.,!()]+$", re.UNICODE)


def is_valid_title(title: str) -> bool:
    raw = unescape(title)
    if "\n" in raw or "\r" in raw:
        return False
    if len(raw) > 200:
        return False
    if "<" in raw or ">" in raw:
        return False
    if not SAFE_PATTERN.match(raw):
        return False

    return True
