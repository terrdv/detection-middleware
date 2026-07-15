"""Shared traffic-simulation machinery.

A Profile describes how one *kind* of client behaves — its User-Agent, whether
it sends browser Accept headers, which path it hits, how many requests it makes,
and the think time between them. We drive many clients concurrently against the
running server and score the detector's verdict (the HTTP status) against each
request's known ground-truth label to build a confusion matrix.

Verdict mapping: 200 = Allow (treated as "human"); 429 = Challenge and
403 = Reject (both treated as "bot").
"""
from __future__ import annotations

import time
import urllib.error
import urllib.request
from collections import Counter, defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed
from dataclasses import dataclass
from typing import Callable

# A realistic desktop Chrome UA, shared by the human profile and the sneaky
# "looks human" bot.
CHROME_UA = (
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) "
    "AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)


@dataclass
class Profile:
    name: str
    label: str                     # ground truth: "human" or "bot"
    user_agent: str
    accept_headers: bool           # send Accept/Accept-Language/Accept-Encoding?
    path: str                      # "/" or the honeypot path
    n_requests: int
    interval: Callable[[], float]  # seconds to wait before each request


def _headers(profile: Profile, client_ip: str) -> dict:
    h = {"User-Agent": profile.user_agent, "X-Forwarded-For": client_ip}
    if profile.accept_headers:
        h["Accept"] = "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
        h["Accept-Language"] = "en-US,en;q=0.9"
        h["Accept-Encoding"] = "gzip, deflate, br"
    return h


def _send(url: str, headers: dict) -> int:
    req = urllib.request.Request(url, headers=headers, method="GET")
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            return resp.status
    except urllib.error.HTTPError as e:
        return e.code              # 403 / 429 arrive here
    except urllib.error.URLError:
        return 0                   # connection refused / server down


def _run_client(profile: Profile, client_ip: str, host: str) -> list:
    url = host + profile.path
    headers = _headers(profile, client_ip)
    out = []
    for _ in range(profile.n_requests):
        time.sleep(profile.interval())
        out.append((profile.label, profile.name, _send(url, headers)))
    return out


def run_population(population: list, host: str, max_workers: int | None = None) -> list:
    """population is a list of (Profile, client_ip). Each client runs in its own
    thread (they mostly sleep, so threads are cheap here)."""
    results: list = []
    workers = max_workers or max(1, len(population))
    with ThreadPoolExecutor(max_workers=workers) as ex:
        futures = [ex.submit(_run_client, p, ip, host) for p, ip in population]
        for f in as_completed(futures):
            results.extend(f.result())
    return results


def _predicted(status: int) -> str:
    return "human" if status == 200 else "bot"


def confusion_matrix(results: list) -> dict:
    tp = fp = fn = tn = errors = 0
    for label, _name, status in results:
        if status == 0:
            errors += 1
            continue
        pred = _predicted(status)
        if label == "bot" and pred == "bot":
            tp += 1
        elif label == "bot":
            fn += 1
        elif label == "human" and pred == "bot":
            fp += 1
        else:
            tn += 1
    return {"tp": tp, "fp": fp, "fn": fn, "tn": tn, "errors": errors}


def print_report(results: list) -> None:
    m = confusion_matrix(results)
    tp, fp, fn, tn = m["tp"], m["fp"], m["fn"], m["tn"]
    total = tp + fp + fn + tn
    precision = tp / (tp + fp) if (tp + fp) else 0.0
    recall = tp / (tp + fn) if (tp + fn) else 0.0
    accuracy = (tp + tn) / total if total else 0.0

    print("\n=== Confusion matrix (positive = bot) ===")
    print(f"{'':14}{'pred bot':>10}{'pred human':>12}")
    print(f"{'actual bot':14}{tp:>10}{fn:>12}")
    print(f"{'actual human':14}{fp:>10}{tn:>12}")
    if m["errors"]:
        print(f"(+{m['errors']} request errors excluded — is the server up?)")

    print("\n=== Metrics ===")
    print(f"precision (flagged that were truly bots): {precision:.3f}")
    print(f"recall    (bots that were caught):        {recall:.3f}")
    print(f"accuracy:                                 {accuracy:.3f}")

    print("\n=== Per-profile status codes ===")
    by_profile: dict = defaultdict(Counter)
    for label, name, status in results:
        by_profile[(label, name)][status] += 1
    for (label, name), counter in sorted(by_profile.items()):
        codes = "  ".join(f"{code}:{n}" for code, n in sorted(counter.items()))
        print(f"[{label:5}] {name:16} {codes}")
