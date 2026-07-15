"""Combine human + bot populations, drive them against the running server, and
print a confusion matrix of the detector's verdicts.

The server must run with X-Forwarded-For trusted so each simulated client gets
its own identity:

    TRUST_XFF=1 go run ./cmd/server

Then:

    python loadtest/sim/run.py                       # 25 clients per profile
    python loadtest/sim/run.py --clients-per-profile 50
"""
import argparse

import bots
import humans
from driver import print_report, run_population


def build_population(clients_per_profile: int) -> list:
    population = []
    counter = {"n": 0}

    def alloc(prefix: str) -> str:
        counter["n"] += 1
        n = counter["n"]
        return f"{prefix}.0.{(n // 256) % 256}.{n % 256}"

    for prof in humans.profiles():
        for _ in range(clients_per_profile):
            population.append((prof, alloc("10")))  # humans -> 10.x
    for prof in bots.profiles():
        for _ in range(clients_per_profile):
            population.append((prof, alloc("11")))  # bots -> 11.x
    return population


def main() -> None:
    ap = argparse.ArgumentParser()
    ap.add_argument("--host", default="http://localhost:8080")
    ap.add_argument("--clients-per-profile", type=int, default=25)
    args = ap.parse_args()

    population = build_population(args.clients_per_profile)
    print(f"driving {len(population)} clients against {args.host} ...")
    results = run_population(population, args.host)
    print(f"collected {len(results)} requests")
    print_report(results)


if __name__ == "__main__":
    main()
