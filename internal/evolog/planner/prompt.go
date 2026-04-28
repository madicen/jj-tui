package planner

// AtomicSplitSystemPrompt is the LLM system prompt for JSON-shaped atomic commit plans.
// Kept in code so DryRun / Refine flows can share one canonical string.
const AtomicSplitSystemPrompt = `Analyze the provided diff. Propose a sequence of atomic, dependent commits. For each commit, include a 1-line summary and the relevant file hunks. Ensure the sequence respects topological dependency: commits must build upon each other logically. Return a single JSON array only (no markdown fences): [{"message": string, "hunks": [string], "sequence_id": number}]. Use sequence_id 0,1,2,... in application order.`
