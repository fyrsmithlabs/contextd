---
name: spec-driven-implementation
description: Drive a spec-first workflow for substantial features by writing a product spec before implementation, writing a tech spec when warranted, and keeping both specs updated as implementation evolves. Use when starting a significant feature, planning agent-driven implementation, or when the user wants product and tech specs checked into source control.
---

# spec-driven-implementation

Drive a spec-first workflow for substantial features in this repository.

## Overview

This skill is the local shared spec-first workflow for this repository. Local wrappers and workflows depend on it directly as the canonical spec-first contract.

Use this skill for significant features where a written spec will improve implementation quality, reduce ambiguity, or make review easier. Be pragmatic: not every change needs specs.

Specs should usually live somewhere under `specs/`.

If a repo-specific wrapper skill or explicit prompt provides exact output paths or filenames, follow those instructions.

These specs should largely be written by agents, not by hand, and should be checked into source control so they can be reviewed and kept current with the code.

## When specs are required

Strongly prefer specs when the change is substantial, such as:

- product or architectural ambiguity
- expected implementation size around 1k+ LOC
- deep or cross-cutting stack changes
- risky behavior changes where regressions would be expensive
- work where agent quality will improve materially from clearer inputs

Specs are often unnecessary for:

- small, local bug fixes
- straightforward refactors
- narrow UI tweaks with little ambiguity

For pure UI changes, the product spec is often useful while the tech spec may be unnecessary.

## Workflow

### 1. Decide whether the feature needs specs

Evaluate the size, ambiguity, and risk of the feature. If specs will not meaningfully improve execution or review, skip them and focus on verification instead.

### 2. Write the product spec first

Before implementation, create the product spec describing the desired user-facing behavior.

Use the `write-product-spec` skill to produce it. The product spec should define:

- what problem is being solved
- the desired user experience
- invariants and edge cases
- success criteria
- how the behavior will be validated

If the feature has UI or interaction design, ask for a Figma mock if one exists. If there is no mock, continue but call that out explicitly in the product spec.

### 3. Write the tech spec when warranted

Use the `write-tech-spec` skill for substantial or ambiguous implementation work.

Prefer a tech spec when:

- the implementation spans multiple subsystems
- architecture or extensibility matters
- there are meaningful tradeoffs to document
- reviewers will benefit more from reviewing the plan than the raw code

It is acceptable to write the tech spec after an end-to-end prototype if that leads to a more accurate implementation plan. Do not force a premature tech spec when the implementation details are still too uncertain.

### 4. Implement approved specs

After the specs are approved, use the `implement-specs` skill to build from the approved product spec and tech spec.

The implementation can often be pushed in the same PR or branch as the product and tech specs. As the engineer iterates, keep the specs, code changes, and tests in that same change so the review reflects the feature that will actually ship.

For large features, the implementer may optionally offer:

- `PROJECT_LOG.md` to track explored paths, checkpoints, and current implementation state
- `DECISIONS.md` to capture concrete product and technical decisions made during design and implementation

These are optional aids, not required outputs.

### 5. Keep specs current during implementation

If implementation changes from the spec, update the spec rather than leaving it stale.

Update the product spec when:

- user-facing behavior changes
- success criteria change
- UX details or edge cases change

Update the tech spec when:

- the implementation approach changes
- architectural boundaries move
- risks, dependencies, or rollout details change
- the testing or validation plan changes

The checked-in specs should describe the feature that actually ships, not just the initial intent. Keep those spec updates in the same change as the related code changes whenever practical.

### 6. Verify behavior against the spec

Before considering the work complete, make sure verification maps back to the specs. Prefer tests and artifacts that validate the product behavior directly, using the repository's existing validation workflows.

## Best Practices

- Be pragmatic above all else.
- Write specs to improve input quality for agents, not as ceremony.
- Keep product specs behavior-oriented and implementation-light.
- Keep tech specs implementation-oriented and grounded in current codebase patterns.
- Use review time to validate specs and behavior, not to over-index on code style nits.

## Related Skills

- `implement-specs`
- `write-product-spec`
- `write-tech-spec`
