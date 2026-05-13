#!/usr/bin/env python3
"""Aggregate recent human feedback on agent-authored PR review comments into JSON."""

from __future__ import annotations

import argparse
import json
import tempfile
import re
import subprocess
from datetime import datetime, timedelta, timezone
from pathlib import Path
from typing import Any
from urllib.parse import urlparse


DEFAULT_REPO = "warpdotdev/oz-for-oss"
DEFAULT_AGENT_LOGINS = ("warp-dev-github-integration[bot]",)
SEVERITY_RE = re.compile(r"^\s*(?:[^\[]+\s+)?\[(CRITICAL|IMPORTANT|SUGGESTION|NIT)\]")

PR_FILES_QUERY = """
query($owner: String!, $repo: String!, $number: Int!, $cursor: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      files(first: 100, after: $cursor) {
        nodes {
          path
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}
"""

REVIEW_THREADS_QUERY = """
query($owner: String!, $repo: String!, $number: Int!, $cursor: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      number
      title
      url
      baseRefName
      headRefName
      reviewThreads(first: 50, after: $cursor) {
        nodes {
          id
          isResolved
          isOutdated
          path
          line
          originalLine
          startLine
          originalStartLine
          diffSide
          comments(first: 100) {
            nodes {
              id
              databaseId
              body
              createdAt
              url
              author {
                __typename
                login
              }
              replyTo {
                databaseId
              }
              pullRequestReview {
                id
                url
                state
              }
            }
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}
"""

ISSUE_COMMENTS_QUERY = """
query($owner: String!, $repo: String!, $number: Int!, $cursor: String) {
  repository(owner: $owner, name: $repo) {
    pullRequest(number: $number) {
      comments(first: 100, after: $cursor) {
        nodes {
          id
          databaseId
          body
          createdAt
          url
          author {
            __typename
            login
          }
        }
        pageInfo {
          hasNextPage
          endCursor
        }
      }
    }
  }
}
"""

def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description=(
            "Aggregate human replies to agent-authored pull request review comments "
            "into a structured JSON file."
        )
    )
    parser.add_argument(
        "--repo",
        help="GitHub repository in owner/name form. Defaults to the current git remote, then warpdotdev/oz-for-oss.",
    )
    parser.add_argument(
        "--pr",
        type=int,
        action="append",
        help="Pull request number to inspect. May be supplied more than once.",
    )
    parser.add_argument(
        "--days",
        type=int,
        default=7,
        help="Look back this many days for recently updated pull requests when --pr is omitted.",
    )
    parser.add_argument(
        "--agent-login",
        action="append",
        help=(
            "GitHub login used by the coding agent whose comments should be analyzed. "
            "May be supplied more than once. Defaults to warp-dev-github-integration[bot]."
        ),
    )
    parser.add_argument(
        "--output",
        help="Path to write the aggregated JSON file. Defaults to a temporary file in /tmp.",
    )
    parser.add_argument(
        "--include-bots",
        action="store_true",
        help="Include bot-authored replies from accounts other than the agent.",
    )
    parser.add_argument(
        "--include-without-replies",
        action="store_true",
        help="Keep agent comments even when they received no qualifying human replies.",
    )
    return parser.parse_args()


def run_command(args: list[str]) -> str:
    try:
        completed = subprocess.run(
            args,
            check=True,
            text=True,
            capture_output=True,
        )
    except FileNotFoundError as exc:
        raise SystemExit(f"Command not found: {args[0]}") from exc
    except subprocess.CalledProcessError as exc:
        message = exc.stderr.strip() or exc.stdout.strip() or str(exc)
        raise SystemExit(message) from exc
    return completed.stdout


def infer_repo() -> str:
    try:
        remote_url = run_command(["git", "remote", "get-url", "origin"]).strip()
    except SystemExit:
        return DEFAULT_REPO
    if remote_url.startswith("git@github.com:"):
        repo = remote_url.split("git@github.com:", 1)[1]
    else:
        parsed = urlparse(remote_url)
        repo = parsed.path.lstrip("/")
    if repo.endswith(".git"):
        repo = repo[:-4]
    if repo.count("/") != 1:
        raise SystemExit(f"Could not infer owner/name from git remote: {remote_url}")
    return repo


def gh_graphql(query: str, variables: dict[str, Any]) -> dict[str, Any]:
    args = ["gh", "api", "graphql", "-f", f"query={query}"]
    for key, value in variables.items():
        if value is None:
            continue
        args.extend(["-F", f"{key}={value}"])
    output = run_command(args)
    try:
        payload = json.loads(output)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"gh api returned invalid JSON: {exc}") from exc
    if payload.get("errors"):
        raise SystemExit(json.dumps(payload["errors"], indent=2))
    return payload


def recent_cutoff_iso(days: int) -> str:
    cutoff = datetime.now(timezone.utc) - timedelta(days=days)
    return cutoff.replace(microsecond=0).isoformat()


def parse_github_datetime(value: str) -> datetime:
    normalized = value.replace("Z", "+00:00")
    return datetime.fromisoformat(normalized)


def fetch_recent_pull_requests(owner: str, repo: str, days: int) -> list[dict[str, Any]]:
    cutoff = recent_cutoff_iso(days)
    cutoff_dt = parse_github_datetime(cutoff)
    output = run_command(
        [
            "gh",
            "pr",
            "list",
            "--repo",
            f"{owner}/{repo}",
            "--state",
            "all",
            "--limit",
            "100",
            "--search",
            f"updated:>={cutoff}",
            "--json",
            "number,title,url,updatedAt",
        ]
    )
    try:
        pulls = json.loads(output)
    except json.JSONDecodeError as exc:
        raise SystemExit(f"gh pr list returned invalid JSON: {exc}") from exc
    return [pr for pr in pulls if parse_github_datetime(pr["updatedAt"]) >= cutoff_dt]


def fetch_review_threads(owner: str, repo: str, number: int) -> tuple[dict[str, Any], list[dict[str, Any]]]:
    pr_meta: dict[str, Any] | None = None
    threads: list[dict[str, Any]] = []
    cursor: str | None = None

    while True:
        payload = gh_graphql(
            REVIEW_THREADS_QUERY,
            {"owner": owner, "repo": repo, "number": number, "cursor": cursor},
        )
        pr = payload["data"]["repository"]["pullRequest"]
        if pr is None:
            raise SystemExit(f"Pull request #{number} was not found in {owner}/{repo}")
        if pr_meta is None:
            pr_meta = {
                "number": pr["number"],
                "title": pr["title"],
                "url": pr["url"],
                "base_ref": pr["baseRefName"],
                "head_ref": pr["headRefName"],
            }
        thread_connection = pr["reviewThreads"]
        threads.extend(thread_connection["nodes"])
        if not thread_connection["pageInfo"]["hasNextPage"]:
            return pr_meta, threads
        cursor = thread_connection["pageInfo"]["endCursor"]


def fetch_pr_files(owner: str, repo: str, number: int) -> list[str]:
    """Fetch all changed file paths for a pull request."""
    paths: list[str] = []
    cursor: str | None = None

    while True:
        payload = gh_graphql(
            PR_FILES_QUERY,
            {"owner": owner, "repo": repo, "number": number, "cursor": cursor},
        )
        pr = payload["data"]["repository"]["pullRequest"]
        if pr is None:
            raise SystemExit(f"Pull request #{number} was not found in {owner}/{repo}")
        file_connection = pr["files"]
        paths.extend(node["path"] for node in file_connection["nodes"])
        if not file_connection["pageInfo"]["hasNextPage"]:
            return paths
        cursor = file_connection["pageInfo"]["endCursor"]


def classify_review_type(changed_files: list[str]) -> str:
    """Return 'spec' if all changed files are under specs/, otherwise 'code'."""
    if changed_files and all(f.startswith("specs/") for f in changed_files):
        return "spec"
    return "code"


def fetch_issue_comments(owner: str, repo: str, number: int) -> list[dict[str, Any]]:
    comments: list[dict[str, Any]] = []
    cursor: str | None = None

    while True:
        payload = gh_graphql(
            ISSUE_COMMENTS_QUERY,
            {"owner": owner, "repo": repo, "number": number, "cursor": cursor},
        )
        pr = payload["data"]["repository"]["pullRequest"]
        if pr is None:
            raise SystemExit(f"Pull request #{number} was not found in {owner}/{repo}")
        comment_connection = pr["comments"]
        comments.extend(comment_connection["nodes"])
        if not comment_connection["pageInfo"]["hasNextPage"]:
            return comments
        cursor = comment_connection["pageInfo"]["endCursor"]


def author_login(comment: dict[str, Any]) -> str | None:
    author = comment.get("author")
    if not author:
        return None
    return author.get("login")


def author_type(comment: dict[str, Any]) -> str | None:
    author = comment.get("author")
    if not author:
        return None
    return author.get("__typename")


def is_human_comment(comment: dict[str, Any], excluded_logins: set[str], include_bots: bool) -> bool:
    login = author_login(comment)
    if not login or login in excluded_logins:
        return False
    if include_bots:
        return True
    return author_type(comment) != "Bot"


def extract_severity(body: str) -> str | None:
    stripped = body.strip()
    first_line = stripped.splitlines()[0] if stripped else ""
    match = SEVERITY_RE.match(first_line)
    if not match:
        return None
    return match.group(1)


def iso_now() -> str:
    return datetime.now(timezone.utc).replace(microsecond=0).isoformat()


def normalize_comment(comment: dict[str, Any]) -> dict[str, Any]:
    return {
        "id": comment.get("databaseId"),
        "node_id": comment.get("id"),
        "author": author_login(comment),
        "author_type": author_type(comment),
        "created_at": comment.get("createdAt"),
        "url": comment.get("url"),
        "reply_to_id": (comment.get("replyTo") or {}).get("databaseId"),
        "body": comment.get("body", ""),
    }


def build_feedback_items(
    threads: list[dict[str, Any]],
    agent_logins: set[str],
    include_bots: bool,
    include_without_replies: bool,
) -> list[dict[str, Any]]:
    items: list[dict[str, Any]] = []

    for thread in threads:
        thread_comments = thread.get("comments", {}).get("nodes", [])
        normalized_thread_comments = [normalize_comment(comment) for comment in thread_comments]

        for index, comment in enumerate(thread_comments):
            if author_login(comment) not in agent_logins:
                continue

            later_comments = []
            for reply in thread_comments[index + 1 :]:
                if author_login(reply) in agent_logins:
                    break
                later_comments.append(reply)

            replies = [
                normalize_comment(reply)
                for reply in later_comments
                if is_human_comment(reply, agent_logins, include_bots)
            ]
            if not replies and not include_without_replies:
                continue

            agent_comment = normalize_comment(comment)
            agent_comment["severity"] = extract_severity(agent_comment["body"])
            agent_comment["has_suggestion_block"] = "```suggestion" in agent_comment["body"]
            items.append(
                {
                    "thread": {
                        "id": thread.get("id"),
                        "path": thread.get("path"),
                        "line": thread.get("line"),
                        "start_line": thread.get("startLine"),
                        "original_line": thread.get("originalLine"),
                        "original_start_line": thread.get("originalStartLine"),
                        "diff_side": thread.get("diffSide"),
                        "is_resolved": thread.get("isResolved"),
                        "is_outdated": thread.get("isOutdated"),
                    },
                    "agent_comment": agent_comment,
                    "human_replies": replies,
                    "human_reply_summary": {
                        "count": len(replies),
                        "authors": sorted({reply["author"] for reply in replies if reply["author"]}),
                    },
                    "thread_comments": normalized_thread_comments,
                }
            )

    return items


def build_human_review_comments(
    threads: list[dict[str, Any]],
    agent_logins: set[str],
    include_bots: bool,
) -> list[dict[str, Any]]:
    comments: list[dict[str, Any]] = []

    for thread in threads:
        for comment in thread.get("comments", {}).get("nodes", []):
            if not is_human_comment(comment, agent_logins, include_bots):
                continue
            comments.append(
                {
                    "thread": {
                        "id": thread.get("id"),
                        "path": thread.get("path"),
                        "line": thread.get("line"),
                        "start_line": thread.get("startLine"),
                        "original_line": thread.get("originalLine"),
                        "original_start_line": thread.get("originalStartLine"),
                        "diff_side": thread.get("diffSide"),
                        "is_resolved": thread.get("isResolved"),
                        "is_outdated": thread.get("isOutdated"),
                    },
                    "comment": normalize_comment(comment),
                }
            )

    return comments


def build_issue_context(
    comments: list[dict[str, Any]],
    agent_logins: set[str],
    include_bots: bool,
) -> list[dict[str, Any]]:
    return [
        normalize_comment(comment)
        for comment in comments
        if author_login(comment) not in agent_logins
        and is_human_comment(comment, agent_logins, include_bots)
    ]


def default_output_path() -> Path:
    tmpdir = Path(tempfile.gettempdir())
    timestamp = datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ")
    return tmpdir / f"update-pr-review-feedback-{timestamp}.json"


def main() -> None:
    args = parse_args()
    repo = args.repo or infer_repo()
    if "/" not in repo:
        raise SystemExit(f"--repo must be in owner/name format, got: {repo}")
    owner, repo_name = repo.split("/", 1)
    agent_logins = set(args.agent_login or DEFAULT_AGENT_LOGINS)

    pr_numbers = args.pr or [pr["number"] for pr in fetch_recent_pull_requests(owner, repo_name, args.days)]
    pull_requests: list[dict[str, Any]] = []
    total_review_thread_count = 0
    total_issue_comment_count = 0
    total_human_issue_comment_count = 0
    total_feedback_item_count = 0
    total_human_review_comment_count = 0

    for pr_number in pr_numbers:
        pr_meta, review_threads = fetch_review_threads(owner, repo_name, pr_number)
        changed_files = fetch_pr_files(owner, repo_name, pr_number)
        pr_meta["changed_files"] = changed_files
        pr_meta["review_type"] = classify_review_type(changed_files)
        issue_comments = fetch_issue_comments(owner, repo_name, pr_number)
        feedback_items = build_feedback_items(
            review_threads,
            agent_logins=agent_logins,
            include_bots=args.include_bots,
            include_without_replies=args.include_without_replies,
        )
        human_review_comments = build_human_review_comments(
            review_threads,
            agent_logins=agent_logins,
            include_bots=args.include_bots,
        )
        issue_context = build_issue_context(
            issue_comments,
            agent_logins=agent_logins,
            include_bots=args.include_bots,
        )

        pull_requests.append(
            {
                "pull_request": pr_meta,
                "feedback_items": feedback_items,
                "human_review_comments": human_review_comments,
                "issue_comments": issue_context,
                "stats": {
                    "review_thread_count": len(review_threads),
                    "feedback_item_count": len(feedback_items),
                    "human_review_comment_count": len(human_review_comments),
                    "issue_comment_count": len(issue_comments),
                    "human_issue_comment_count": len(issue_context),
                },
            }
        )
        total_review_thread_count += len(review_threads)
        total_issue_comment_count += len(issue_comments)
        total_human_issue_comment_count += len(issue_context)
        total_feedback_item_count += len(feedback_items)
        total_human_review_comment_count += len(human_review_comments)

    output = {
        "repository": repo,
        "agent_logins": sorted(agent_logins),
        "generated_at": iso_now(),
        "lookback_days": args.days,
        "pull_request_count": len(pull_requests),
        "pull_requests": pull_requests,
        "stats": {
            "review_thread_count": total_review_thread_count,
            "feedback_item_count": total_feedback_item_count,
            "human_review_comment_count": total_human_review_comment_count,
            "issue_comment_count": total_issue_comment_count,
            "human_issue_comment_count": total_human_issue_comment_count,
        },
    }

    output_path = Path(args.output) if args.output else default_output_path()
    output_path.parent.mkdir(parents=True, exist_ok=True)
    output_path.write_text(json.dumps(output, indent=2) + "\n", encoding="utf-8")
    print(output_path)


if __name__ == "__main__":
    main()
