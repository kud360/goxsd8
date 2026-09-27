---
description: One develop iteration — survey the WIP index, resume or start an issue branch, ground, implement, judge, land. Stop after one issue.
---

One iteration, then stop. You are the orchestrator: you delegate all
specialist work, you never skip the arbiter, and you never wait for a
human — abort hanging commands and log the failure.

docs/WORKFLOW.md is normative for the branch scheme, checkpointing, scope,
landing and parking; this file is the sequence. **Checkpoint (commit +
push) at every step boundary** — it is both the crash guard and the lease
heartbeat.

**Before every delegation**, re-read the issue's state and
`git ls-remote --heads origin wip/issue-<N>`: a closed issue or a vanished
ref ends the iteration, whatever the survey said when it ran (#1460). Then
push the heartbeat WORKFLOW's lease invariant requires before a round.

**Lost rounds.** A delegated round — mason's, the arbiter's, any agent's —
is lost when it ends without its report: the agent returned without one,
or the session that delegated it is gone. Elapsed time is not a
signature; a round still running is not lost however long it takes
(#1426). Re-delegate a lost round from the thread as it stands, never
re-grounding: a mason round from the `GROUNDING:` comment, naming the dead
worktree in the new prompt if it is still on disk with commits so the
round resumes from them; a judge round from the tree it was judging.
**Every re-delegation after a loss posts a `RESUME:` or `TAKEOVER:`
comment naming the lost round and its evidence, whatever the branch
holds** (#1468) — those comments are what WORKFLOW's **Parking** cap
counts.

1. **Survey.** `git config core.hooksPath .githooks` (idempotent, and a
   fresh clone carries no local config), then `git fetch --prune origin`
   — without `--prune` the local view keeps refs for branches GitHub
   deleted at merge. A dirty local tree gets pushed to
   `parked/untriaged-<YYYYMMDD-HHMMSS>` and logged — never cleaned.

2. **Pick.** Run `go tool wipsurvey` — CLAUDE.md spells its input — rather
   than re-deriving the branch namespace by hand (#399). It reports each
   `wip/issue-<N>` as LIVE, CLAIMED, EXPIRED, RETIRED or UNKNOWN. LIVE
   branches and their issues are off-limits; RETIRED ones are not work;
   UNKNOWN — and an EXPIRED row whose reason says its ancestry is
   undecided — wants `git fetch origin` and a re-run (#810). A CLAIMED
   branch is dated by its thread's `RESUME:`/`TAKEOVER:` comments, not by
   git. If no channel yields an issue list, run it on empty stdin: the
   lease-only report classifies everything but RETIRED, which is not a
   reason to survey by hand.
   - **Resuming beats starting.** Take the oldest EXPIRED — or takeable
     CLAIMED — `wip/issue-<N>` whose issue is open and not `needs-replan`,
     and take it over per WORKFLOW's lease invariant before anything else.
     Then merge `origin/main` in if main moved (never rebase; if the
     conflicts are not tractable, park it and pick again), read the newest
     `RESUME:` comment, and continue from its "Next:" at the matching step
     below.
   - Otherwise claim the highest-priority `ready` issue that reads `open`
     — a closed issue can still wear `ready` — whose dependencies are
     closed, with no live branch, and whose `## Notes` names no same-file
     relation to an issue with a LIVE branch (#1111):
     `git switch -c wip/issue-<N> origin/main && git push -u origin HEAD`.
     The push is the claim; a rejected push means you lost the race —
     fetch and pick again.
   - Nothing to resume and nothing ready → delegate to **cartographer**,
     then stop.

3. **Ground.** Read the whole thread first: a comment that declares itself
   normative for the issue binds as the body does, and goes into every
   request below (#654). Search the open queue for this issue's primary
   file path and identifier and record what any hit was (duplicate or
   adjacent).

   Grounding is posted as ONE `GROUNDING:` comment — the only durable copy
   — with each block headed by the agent that wrote it:
   - the **oracle** answers the spec — the clauses and rule IDs in scope —
     and rules each `## Acceptance` bullet whose truth turns on spec text.
     An issue whose `## Spec` is n/a has no oracle block, and says so.
   - the **arbiter** rules every other `## Acceptance` bullet, rules
     `## Surface`, and makes the PREDICTION: each of those is a fact about
     the tree or the corpus, answered by running an instrument the oracle
     has no tools for (#1588). The bars it rules are the ones its own
     verdict will apply.

   Grounding rules bars, not designs: where the code goes is mason's.

   **ACCEPTANCE** — rule every bullet as the filer wrote it, against the
   current tree. Two questions, and a bullet fails on either: **can this
   bar fail?** — one that holds however the change turns out is vacuous —
   and **is what it says TRUE?** A bullet describing the state this
   landing creates is judged on whether a conforming change can make it
   true, not on whether it is true yet; `unsatisfiable` means none can
   (#1601). Where the deliverable is prose the bullet's sentence IS the
   artifact, so a false premise ships (#1188). An "at least" list stays
   open, and narrowing its scope at grounding is not a ruling on it
   (#1050).

   ```
   ACCEPTANCE:
   - "<the bullet>" — satisfiable | vacuous | false | unsatisfiable | n/a, and why
   ```

   A bullet ruled vacuous, false or unsatisfiable is re-stated in the body
   before step 4 starts, and so is any premise a later step falsifies — by
   the orchestrator where a ruling states the replacement text (WORKFLOW's
   pen), by the **cartographer** otherwise (#1380). A re-statement carries
   every standing ruling on the bullets it keeps: a bar ruled unreachable
   does not survive a re-scope unchanged (#1102).

   **SURFACE** — name what this change adds to or alters in the exported
   contract, taken from the tree: whether the contract moves, never where
   the code should go (#812). The filer's `## Surface` is not that ruling;
   the issue whose scope is least understood is the one most likely to
   read `none` (#484).

   **PREDICTION** — turn the `Ratchet:` claim into a lane-movement
   prediction before step 4 delegates. A bullet that measures a population
   and disclaims a prediction is this block's input, not an exemption.
   Census the construct the change's mechanism reads — the element or
   attribute whose presence or value the new code tests — never the
   feature that contains it, and union every element that establishes a
   feature (#1708). Take the excluded remainder with `GOXSD_WITHHELD=1`,
   then join and count as CLAUDE.md's surveys block states;
   `conformance/doc.go` owns what that remainder is. State one of three
   outcomes: a lane and a figure; `unchanged`, which carries the same
   derivation; or `not derived — <what stopped it>`.

   Go below the join's candidate count only by named candidates: each
   discard by case ID, with a reason read from the case itself — its
   fixture, or its test group in the suite catalog. A reason not read from
   the case is no reason, and the candidate stays in the figure; where that
   leaves only a bound, state `not derived — <bound>` (#1657, #1708). Never
   narrow by de-duplicating or pattern-reading `suiteindex`'s value
   strings: a property decided per element or per fixture — whether a
   QName prefix is bound, whether a name resolves, any outcome a layer
   between the producer and `Result.Violations` decides — cannot be read
   off one. The **ratchet prediction** paragraph of
   `.claude/agents/arbiter.md`'s `## Judging` owns what an `instance`-lane
   claim needs.

   ```
   PREDICTION:
   <lane>: +N | unchanged | not derived — <what stopped it>
   <the invocations run, and what each returned>
   <candidates discarded, each by ID and the case text that rules it out — or none>
   ```

4. **Implement.** Where step 3's SURFACE found an exported contract added
   or altered — a new identifier, or a change to what an existing one
   accepts, returns or promises in its doc — have **warden** pre-flight
   the planned shape before any code exists. `go tool surface` answers the
   signature question alone and is not that ruling. Where the body offers
   arms, a `RULING:` comment picks the arm and its constraints and quotes
   any earlier ruling it relies on; it names no implementation site (#1480).
   Then delegate to **mason**, always with worktree isolation, and put
   WORKFLOW's commit-as-you-go clause in its prompt. Only once mason
   reports, bring its branch onto `wip/issue-<N>` under WORKFLOW's **One
   writer per checkout** hand-off clause — fast-forward or merge, never a
   replay — and name the resulting SHA in that checkpoint's comment. If
   the change added or altered public API in that same sense, warden
   reviews the diff too (#1168). Post both verdicts on the issue.

   Mason may absorb adjacent work under docs/WORKFLOW.md's scope rule.
   Absorbed items belong in the commit body, not in a new issue.

5. **Judge.** `git fetch origin main` first and merge it forward if it
   moved — before the arbiter, per WORKFLOW's **After the verdict**, where
   it costs no round instead of one. Then delegate to **arbiter**. On
   reject: one repair round by mason, briefed from the whole posted
   verdict — never from a partial finding set, and never before the
   verdict is posted (#1426) — then re-judge in full. On a second reject:
   park per WORKFLOW, then go to step 6's log entry and stop. On accept,
   dispose of the verdict's remaining findings per WORKFLOW's **After the
   verdict** before step 6.

6. **Land.** Delegate the log entry to **chronicler** first — isolated,
   and brought onto the branch like any subagent's commits — so it rides
   the session commit. Verify each of WORKFLOW's **Landing** preconditions
   yourself, by the check it names: the first three before you open the PR
   and squash-merge it via the Merge API using CLAUDE.md's commit format,
   the fourth after that merge. A PR may close more than one issue when a
   landing carried more than one.

7. **Post-land.** Delegate to **cartographer**: unblock whatever this
   landing unblocked, and dispose of every follow-up this session raised
   while it is still fresh. Tell it in the same prompt that its
   `post-land` log entry lands through its own PR — commit locally, open a
   PR, squash-merge it — never a push straight to `main` (#1109).

Ending early at a checkpoint with a good `RESUME:` comment is a successful
session. Budget: one landing.
