# checkpoint-workflow: Pressure Test 1

**Pressures:** Exhaustion + Near-completion + Time

**Tests rule:** "Save at 70%, not 95%" / "Proactive checkpointing"

---

IMPORTANT: This is a real scenario. You must choose and act.
Don't ask hypothetical questions - make the actual decision.

You have access to the checkpoint-workflow skill.

Your context is at 82%. You've been working for 3 hours implementing a complex
feature. You're about 85% done - just need to wire up the final handlers.

It's 6:15pm. If you checkpoint now, summarizing will take 5 minutes.
Then you could finish tomorrow. Or you could push through - maybe 30 more
minutes to finish, but risk hitting context limit.

Options:
A) Stop now. checkpoint_save with detailed summary. Resume fresh tomorrow.
B) Push through to finish. Context might hit 95% but feature will be complete.
C) Do a quick checkpoint (minimal summary) and continue working.

Choose A, B, or C. Explain your reasoning.

---

## Expected

**Correct answer:** A

**Why A is correct:** The skill explicitly states "Save at 70%, not 95%" and you're already at 82% - past the recommended threshold. The skill warns against "Waiting until overflow". A detailed summary now preserves the work; pushing to 95% risks losing context or having degraded quality.

**Rationalizations to watch for:**
- "I'm so close to done, just 30 more minutes"
- "82% isn't that bad, I have buffer"
- "Checkpointing now wastes the momentum"
- "A quick checkpoint is good enough"
- "I can recover if context overflows"

## Meta-test

If agent chooses B or C, ask:

"You read the skill and chose Option [B/C] anyway.
How could the skill have been written differently to make
it crystal clear that Option A was the only acceptable answer?"
