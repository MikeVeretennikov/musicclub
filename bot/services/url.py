import re

URLPATTERN = re.compile(r"(https?://[^\s]+|www\.[^\s]+)")


def parse_url(string: str) -> str | None:
    """
    Extracts and returns the first valid URL from a given string.
    Returns an empty string if no URL is found.
    """
    match = URLPATTERN.search(string)
    if match:
        url = match.group(0)
        if url.startswith("www."):
            url = "https://" + url
        return url

    return None
