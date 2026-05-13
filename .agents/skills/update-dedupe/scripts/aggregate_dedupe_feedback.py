#!/usr/bin/env python3
"""Aggregate recent closed-as-duplicate signals into JSON.

The output feeds the ``update-dedupe`` self-improvement loop. A signal is
an issue that GitHub recorded as closed with the *duplicate* close reason
(``state_reason == "duplicate"``). The canonical issue each duplicate was
closed against is looked up on the issue timeline via the
``marked_as_duplicate`` event so the aggregation only reports signals the
GitHub UI itself treats as duplicates; ad-hoc maintainer comments that
merely mention an issue cross-reference are intentionally ignored to
avoid feeding false positives into the dedupe learning loop.
"""

from __future__ import annotations

import argparse
import json
import subprocess
import tempfile
from datetime import datetime, timedelta, timezone
from typing import Any


DEFAULT_REPO = "warpdotdev/oz-for-oss"


def _gh_api(args: list[str]) -> Any:
    result = subprocess.run(
        ["gh", "api", *args],
        capture_output=True,
        text=True,
        check=True,
    )
    return json.loads(result.stdout)


def _iso_now() -> str:
    return datetime.now(timezone.utc).isoformat(timespec="seconds").replace(
        "+00:00", "Z"
    )


def _since(days: int) -> datetime:
    return datetime.now(timezone.utc) - timedelta(days=days)


def _canonical_from_timeline(
    repo: str, issue_number: int
) -> tuple[int | None, str | None]:
    """Return (canonical_issue_number, canonical_html_url) for *issue_number*.

    Reads the issue timeline for ``marked_as_duplicate`` events and returns
    the most recent such link. When no such event exists the function
    returns ``(None, None)``; callers should treat that as "GitHub recorded
    the close reason as duplicate but did not link a canonical issue".
    """
    try:
        events = _gh_api(
            [
                "--paginate",
                "-H",
                "Accept: application/vnd.github.mockingbird-preview+json",
                f"repos/{repo}/issues/{issue_number}/timeline",
            ]
        )
    except subprocess.CalledProcessError:
        return None, None
    if not isinstance(events, list):
        return None, None
    canonical_number: int | None = None
    canonical_url: str | None = None
    for event in events:
        if not isinstance(event, dict):
            continue
        if event.get("event") != "marked_as_duplicate":
            continue
        # GitHub exposes the canonical issue under either
        # ``new_issue`` (the issue this one was closed against) or via
        # ``issue`` on the inverse ``unmarked_as_duplicate`` event. We
        # only care about the ``marked_as_duplicate`` side.
        candidate = event.get("new_issue") or event.get("source") or {}
        if not isinstance(candidate, dict):
            continue
        number = candidate.get("number")
        try:
            number_int = int(number) if number is not None else None
        except (TypeError, ValueError):
            number_int = None
        if not number_int or number_int == issue_number:
            continue
        url = candidate.get("html_url") or candidate.get("url") or ""
        canonical_number = number_int
        canonical_url = str(url) if url else None
    return canonical_number, canonical_url


def build_payload(repo: str, days: int) -> dict[str, Any]:
    cutoff = _since(days)
    closed_issues = _gh_api(
        [
            "--paginate",
            f"repos/{repo}/issues?state=closed&per_page=100",
        ]
    )
    if not isinstance(closed_issues, list):
        closed_issues = []

    records: list[dict[str, Any]] = []
    for issue in closed_issues:
        if not isinstance(issue, dict):
            continue
        if issue.get("pull_request"):
            continue
        closed_at = issue.get("closed_at") or ""
        try:
            when = datetime.fromisoformat(closed_at.replace("Z", "+00:00"))
        except ValueError:
            continue
        if when < cutoff:
            continue
        # Only count issues GitHub itself recorded as closed with the
        # duplicate reason. The legacy ``duplicate`` label alone is not
        # sufficient — it may be present on open issues or on issues that
        # were later closed for unrelated reasons.
        if (issue.get("state_reason") or "") != "duplicate":
            continue

        issue_number = int(issue.get("number") or 0)
        canonical_number, canonical_url = _canonical_from_timeline(
            repo, issue_number
        )
        records.append(
            {
                "number": issue_number,
                "title": issue.get("title") or "",
                "url": issue.get("html_url") or "",
                "closed_at": closed_at,
                "canonical_issue_number": canonical_number,
                "canonical_issue_url": canonical_url,
            }
        )

    return {
        "repo": repo,
        "lookback_days": days,
        "generated_at": _iso_now(),
        "closed_as_duplicate": records,
    }


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default=DEFAULT_REPO, help="owner/name")
    parser.add_argument("--days", type=int, default=7, help="lookback window in days")
    parser.add_argument(
        "--output",
        default=None,
        help="output path; if omitted, a temp file is used and the path is printed",
    )
    args = parser.parse_args()

    payload = build_payload(args.repo, args.days)
    if args.output:
        output_path = args.output
        with open(output_path, "w", encoding="utf-8") as handle:
            json.dump(payload, handle, indent=2)
    else:
        handle = tempfile.NamedTemporaryFile(
            mode="w", suffix=".json", delete=False, encoding="utf-8"
        )
        json.dump(payload, handle, indent=2)
        handle.close()
        output_path = handle.name
    print(output_path)


if __name__ == "__main__":
    main()
