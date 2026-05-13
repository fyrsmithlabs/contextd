---
name: review-spec-local
specializes: review-spec
description: Repo-specific spec-review guidance for oz-for-oss. Only the categories declared overridable by the core review-spec skill may be specialized here.
---

# Repo-specific spec-review guidance for `oz-for-oss`

This file is a companion to the core `review-spec` skill. It does not
redefine the review output schema, severity labels, safety rules, or
evidence rules. It only specializes the override categories the core
skill marks as overridable.

## Required spec sections in this repository

Spec pull requests in this repo land under `specs/GH<issue-number>/` and
typically include both a `product.md` and a `tech.md`. When reviewing, check that:

- `product.md` clearly states the problem, goals, non-goals, user experience, and validation plan
- `tech.md` clearly states the problem, relevant code, current state, proposed changes, risks, and follow-ups
- both files reference the originating GitHub issue by number in the top-level heading
- internal links reference files and line ranges using the repo-root-relative convention (for example ``path/file:line`` or ``path/file (start-end)``)

## Linking conventions

- Prefer repo-root-relative links over absolute filesystem paths in spec prose.
- When a spec references another spec in the same repository, link to it via its relative path under `specs/`.
