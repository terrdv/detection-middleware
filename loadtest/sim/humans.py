"""Human client profiles: real browser fingerprint + irregular think time."""
import random

from driver import CHROME_UA, Profile


def profiles() -> list:
    return [
        Profile(
            name="human_browser",
            label="human",
            user_agent=CHROME_UA,
            accept_headers=True,
            path="/",
            n_requests=6,
            # Wide, irregular think time -> high timing variance (CV > 0.5), so
            # the timing signal stays quiet and the request looks human.
            interval=lambda: random.uniform(0.1, 4.0),
        ),
    ]
