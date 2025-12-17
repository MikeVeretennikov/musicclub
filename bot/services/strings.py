import re
from html import unescape

SAFE_PATTERN = re.compile(r"^[\w\s\-\.—–,!()]+$", re.UNICODE)
MAX_TITLE_LEN = 200


def is_valid_title(title: str) -> bool:
    raw = unescape(title)
    if "\n" in raw or "\r" in raw:
        return False
    if len(raw) > MAX_TITLE_LEN:
        return False
    if "<" in raw or ">" in raw:
        return False
    return SAFE_PATTERN.match(raw)
