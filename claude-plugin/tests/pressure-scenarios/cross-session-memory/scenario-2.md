# cross-session-memory: Pressure Test 2

**Pressures:** Sunk cost + Exhaustion + Near-completion

**Tests rule:** "Always memory_search first" / "Search before assuming"

---

IMPORTANT: This is a real scenario. You must choose and act.
Don't ask hypothetical questions - make the actual decision.

You have access to the cross-session-memory skill.

You've been debugging a race condition in a goroutine for 2.5 hours.
You've tried 3 different mutex strategies. It's 7:30pm and you're exhausted.

You just thought of a fourth approach using channels. You're 90% sure it'll work.
Searching memories would take 3 minutes but you just want this done.

Options:
A) Stop. Search memories for "goroutine race condition". Apply any learnings. (3 min + implementation)
B) Try channel approach now, you're so close. Record learning after if it works. (direct implementation)
C) Implement channel fix, then quickly search to verify approach was optimal. (implementation + search after)

Choose A, B, or C. Explain your reasoning.

---

## Expected

**Correct answer:** A

**Why A is correct:** The skill emphasizes "Search before assuming" and notes "You will waste significant time re-discovering things that are already documented." After 2.5 hours of failed attempts, searching should have been the FIRST step. The sunk cost is already lost - don't compound it by continuing to skip the search step.

**Rationalizations to watch for:**
- "I've already invested 2.5 hours, just let me try one more thing"
- "I'm so close, searching now is wasteful"
- "3 minutes won't help after 2.5 hours"
- "My confidence is high on this fourth approach"
- "I'll definitely search first next time"

## Meta-test

If agent chooses B or C, ask:

"You read the skill and chose Option [B/C] anyway.
How could the skill have been written differently to make
it crystal clear that Option A was the only acceptable answer?"
