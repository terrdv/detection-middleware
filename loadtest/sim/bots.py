"""Bot client profiles, including the interesting case: a request that looks
perfectly human on every stateless signal but betrays itself through timing."""
from driver import CHROME_UA, Profile


def profiles() -> list:
    return [
        # Obvious scraper: scripting-tool UA + no browser Accept headers.
        # Caught by the user_agent and headers signals.
        Profile(
            name="bot_scraper",
            label="bot",
            user_agent="python-requests/2.31.0",
            accept_headers=False,
            path="/",
            n_requests=6,
            interval=lambda: 0.2,
        ),
        # Scanner: pokes the honeypot path -> instant reject regardless of
        # anything else.
        Profile(
            name="bot_scanner",
            label="bot",
            user_agent="python-requests/2.31.0",
            accept_headers=False,
            path="/wp-admin",
            n_requests=3,
            interval=lambda: 0.5,
        ),
        # The sneaky one: real Chrome UA + full Accept headers + a normal path,
        # so every stateless signal says "human". But it fires on a metronome
        # (constant interval -> near-zero timing variance), which the timing
        # signal flags. This is the "human-like data, low variance -> bot" case.
        Profile(
            name="bot_metronome",
            label="bot",
            user_agent=CHROME_UA,
            accept_headers=True,
            path="/",
            n_requests=10,
            interval=lambda: 1.0,  # constant -> CV ~ 0
        ),
    ]
