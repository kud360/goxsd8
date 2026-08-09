# Development Workflow

goxsd8 is developed by Claude Code sessions — scheduled Claude routines
and on-demand local runs execute the **same slash commands**, so automated
and interactive behavior are identical by construction. A session is
short; the repo and its GitHub issues are the brain.

**Long-horizon memory lives in GitHub issues** (plans, groundings,
verdicts, hand-off notes — all as issue comments) **and docs/LOG/**
(history). Anything a future cold-start session needs must be on the issue
thread or in the log, never only in a transcript. For issue operations
use whichever GitHub channel the session has (docs/ROUTINES.md ranks
them): the cloud platform's built-in GitHub tools, the GitHub MCP server
(needs `GITHUB_PAT` when headless), or the `gh` CLI.

**The container is ephemeral.** A scheduled routine may start from a
fresh clone every run: local git state — stashes, dirty trees, local-only
branches, `.agent/` scratch — does NOT survive between sessions. The rule
that follows: **anything not pushed does not exist** (PRINCIPLES 28).
All work therefore happens on pushed WIP branches under a fixed naming
scheme; the scheme itself is how a cold-start session discovers in-flight
work.

## The branch scheme (the WIP discovery index)

The remote branch namespace is a machine-readable index. Three kinds of
branches carry develop-loop state; nothing else is ever pushed except the
short-lived branch a maintenance trigger lands from (see "Other
triggers" — it opens and squash-merges its PR in the same session, so it
holds no lease and never appears in a survey):

| Branch | Meaning | Lifecycle |
|---|---|---|
| `main` | always green; receives only squash-merged PRs — develop-loop `wip/issue-` work and the maintenance triggers' `meta:`/`conformance:` commits (/backlog, /retro, /ratchet) land the same way | permanent |
| `wip/issue-<N>` | THE work branch for issue #N — at most one, its name is stable | created when work starts; auto-deleted by GitHub when its PR squash-merges; retired in place if the attempt is abandoned |
| `parked/untriaged-<YYYYMMDD-HHMMSS>` | unattributable work found in a dirty local tree | kept for human triage |

Sessions only ever CREATE refs; the one deletion in the system is
GitHub's auto-delete on merge. Any session can reconstruct the entire
in-flight state with one command, no issue archaeology required:

```sh
git ls-remote --heads origin 'refs/heads/wip/*' 'refs/heads/parked/*'
```

Invariants the scheme encodes:

- **The name is the claim; the tip time is the lease.** `wip/issue-<N>`
  existing means issue #N has an in-flight attempt, and its stable name
  makes a second branch for the same issue impossible to create by
  accident. But existence alone cannot distinguish "being worked right
  now" from "abandoned by a dead session" — that is the tip's committer
  timestamp (`git log -1 --format=%cI origin/wip/issue-<N>`):
  - tip newer than the **claim TTL (2 hours)** → LIVE: another session
    presumably holds it. Do not resume it, and do not start issue #N.
  - tip older than the TTL → EXPIRED: the owner is gone (a healthy
    session checkpoints far more often than that) → resumable.
  Checkpoint pushes are therefore also the lease heartbeat: a step
  expected to run long pushes intermediate commits rather than letting
  its lease lapse mid-work.
- **Races are settled by git's atomic ref updates — never by force.**
  Two sessions claiming or checkpointing the same branch: the second
  push is rejected (non-fast-forward). A rejected push to `wip/*` means
  you lost the race — fetch, abandon your local copy of that attempt,
  and pick a different issue. Force-pushing any `wip/*` or `parked/*`
  ref is forbidden; it is the one way sessions could actually stomp
  each other.
- **A `wip/` branch is work only while its issue is open and not
  `needs-replan`.** Otherwise it is retired: landed branches vanish on
  merge, abandoned ones stay put as re-planning evidence — never
  resumed, never deleted by a session. A retired name is never
  contended, because an issue is never re-attempted under its own
  number: re-planning supersedes it with a new issue, and the fresh
  attempt starts as `wip/issue-<M>` from `origin/main`.
- **Checkpoint = commit + push.** Work is committed on the WIP branch at
  every step boundary (grounding done, implementation done, each verdict)
  with message `wip #<N>: <step>`, and pushed immediately. A session that
  dies loses at most the work since its last checkpoint — and its lease
  expires on its own, so the next session recovers the branch without
  human help.
- **Freshly-fetched `origin/main` is the only base.** Every diff, merge,
  and branch point is taken against `origin/main` after an explicit
  `git fetch origin main` in THIS session. A local `main` is never
  authoritative — sessions run in ephemeral containers whose refs are
  stale from clone time, and even `origin/main` is only as fresh as the
  last fetch. Judging or branching against a stale base invents changes
  that are not there and hides changes that are (`git diff main..HEAD`
  is the shape that fails; `git diff origin/main...HEAD` after a fetch is
  the shape that works). The same rule in the other direction: judge the
  COMMITTED tree, not the working tree — if `git status --porcelain` is
  non-empty, what was verified is not what will land.
- **Landing is atomic.** Accept → open a PR for the WIP branch and
  squash-merge it via the GitHub Merge API: the squash is the session's
  ONE commit (code + log entry), `Closes #<N>` in the PR body closes
  the issue, and GitHub auto-deletes the head branch (repo setting
  "Automatically delete head branches" — keep it ON). A `wip/issue-<N>`
  whose issue is closed is retired — survey skips it; the cartographer
  verifies its content is in main (`git log`/diff) and supersedes the
  issue if it isn't.

## The cast

| Agent | File | Model | Role |
|---|---|---|---|
| **mason** | `.claude/agents/mason.md` | opus | Implements one issue at a time |
| **arbiter** | `.claude/agents/arbiter.md` | opus | Judges changes; owns the ratchet verdict |
| **oracle** | `.claude/agents/oracle.md` | sonnet | Spec exegesis; answers only from `docs/specs/md` |
| **warden** | `.claude/agents/warden.md` | opus | API/type-safety review; illegal states unrepresentable |
| **cartographer** | `.claude/agents/cartographer.md` | opus | Long-horizon planning; owns GitHub issues/milestones |
| **steward** | `.claude/agents/steward.md` | opus | Long-horizon architecture steward; audit (Part 2 of /retro); pre/post-1.0 mobility policy |
| **chronicler** | `.claude/agents/chronicler.md` | opus | Session logs; meta-process retrospectives |
| **libuser** | `.claude/agents/libuser.md` | sonnet | Role-plays a library consumer; works only from godoc + README |
| **cliuser** | `.claude/agents/cliuser.md` | sonnet | Role-plays a CLI user; works only from README + `-help` |

Roles do not blur: mason never re-baselines the ratchet, arbiter never
implements fixes, oracle never writes code, libuser/cliuser never read
the source, steward never touches Go code (it files refactor issues;
warden judges individual diffs, steward judges the whole). The
orchestrating session delegates and coordinates; it does no specialist
work itself and never skips the arbiter.

## The develop loop (`/develop`, the default scheduled trigger)

1. **Survey** — activate the repo's git hooks, then list the WIP index:

   ```sh
   git config core.hooksPath .githooks   # idempotent; run it EVERY session —
                                         # a fresh clone carries no local config
   git fetch origin
   git ls-remote --heads origin 'refs/heads/wip/*'
   ```

   The hooks are the only guard that reads the git INDEX rather than the
   working tree (see "Merge-conflict resolution"), and local git config
   does not survive an ephemeral container, so activation rides this
   always-executed step instead of being its own skippable one. If the
   local tree is somehow dirty (persistent local checkout only; a routine
   container starts clean), push it to `parked/untriaged-<ts>` first and
   log it — never clean it (PRINCIPLES 28).
2. **Pick** — partition the `wip/*` branches by lease
   (tip timestamp vs the 2h claim TTL — see the branch scheme):
   - **LIVE** branches (and their issues) are off-limits this session.
   - **Resuming beats starting**: if any EXPIRED `wip/issue-<N>` exists
     whose issue is open and not `needs-replan`, take the oldest —
     switch to it, merge `origin/main` in if main has moved (never
     rebase — a rewritten branch cannot be pushed without force), read
     the issue's newest `RESUME:` comment, and continue from its
     "Next:". (A merge with non-trivial conflicts → park the branch,
     comment, pick again.)
   - Otherwise take the highest-priority `ready` issue with closed
     dependencies and no live branch, and claim it:

     ```sh
     git switch -c wip/issue-<N> origin/main
     git push -u origin HEAD     # the push IS the claim
     ```

     Push rejected → another session claimed it between your survey and
     now; fetch and pick again.
   - Nothing to resume, nothing ready → run the cartographer instead
     and stop.
3. **Ground** — before delegating, search the open issue queue for this
   issue's primary file path and identifier (the filing-discipline bullet
   under "GitHub conventions"): a hit is a duplicate to close or an
   adjacent issue to cross-reference, recorded either way, never silent
   overlap. Then ask the **oracle** for the exact spec clauses and rule
   IDs in scope. Post the answer verbatim as a comment on the issue
   (`GROUNDING:` prefix) — that comment is the grounding's only durable
   copy, so never point at a session-local path instead. The citation
   goes in the commit. **Checkpoint.**
4. **Implement** — if the issue's `## Surface` section is non-"none",
   **warden** pre-flights the planned surface (sketch + intended type
   shapes) before any code is written; post it on the issue. Then
   **mason** makes the smallest change that closes the issue, per
   docs/STYLE.md — spawned WITH worktree isolation, never into the
   orchestrator's own checkout, so it commits on the isolated
   worktree's local branch (see "One writer per checkout"; mandatory
   for every mason invocation). Only after mason reports completion —
   never mid-flight, never because a stop hook warned about
   uncommitted changes — fast-forward-merge that local branch into
   `wip/issue-<N>`. New/changed public API → **warden** reviews the
   diff before proceeding (post the verdict on the issue).
   **Checkpoint.**
5. **Judge** — **arbiter** runs the gate
   (`go build ./... && go test ./... && go vet ./...` + the lint gate +
   `go tool commentwrap ./...` + the conformance run), reviews the
   branch's committed diff against
   freshly-fetched `origin/main` (never a local `main` — see the branch
   scheme) per STYLE.md including the exported-surface diff (T5), and
   posts a verdict on the issue. **Checkpoint after each verdict.**
   The gate is exactly that list — CLAUDE.md's "Commands" block is its
   single source of truth, and a step absent from that block is not a
   gate step no matter which brief, LOG entry, or issue names one (the
   standing example is `go tool logguard`, which has never existed). A
   phantom step is a defect in the text that names it, not a gate
   failure, and confirming its absence from scratch is not the
   arbiter's job.
   - *accept* → arbiter runs the ratchet (`GOXSD_RATCHET=1`, upward
     only). Running it and banking it are ONE step. Immediately after the
     run — before anything else, no branch switch, no ending the session
     — check `git status --porcelain -- conformance/testdata/expectations/`:
     - non-empty → `git add` those files and commit them on the CURRENT
       WIP branch right then, as their own checkpoint, naming the lane
       movement in the commit message; only then post the verdict.
     - empty → the run found no movement; say so.

     The posted verdict states exactly one of three things: "ratchet run,
     write committed as `<sha>`" (the real short SHA of that commit);
     "ratchet run, tree clean, nothing to bank" — the run left
     `git status --porcelain -- conformance/testdata/expectations/` empty,
     so there was no upward movement to commit; or "ratchet not run,
     because `<X>`" — the last only when the gate's conformance run was
     itself inapplicable to the change, and it must name that reason.
     There is no fourth state: running the ratchet, discarding the write,
     and reporting `Ratchet: unchanged` is the defect this rule exists to
     prevent — it is how #202 stranded six `schema`-lane flips for a later
     branch to absorb.
   - *reject* → one repair round by mason (edit the flagged lines, don't
     rewrite), then re-judge. Second reject → **park** (see below),
     comment findings, relabel `needs-replan`. **Two rejections is the
     hard cap** (PRINCIPLES 30).
6. **Land** — **chronicler** appends to `docs/LOG/<year>-<month>.md` on
   the WIP branch FIRST (PRINCIPLES 29) and checkpoints; then land
   through GitHub, never a local merge:

   0. **Two verified preconditions, in this order, before the PR is
      opened.** Both are checks the orchestrating session performs and
      states; neither is anyone else's to volunteer.

      **(a) The LOG entry is IN the branch's diff.** Not "the chronicler
      was invoked" — the entry is present in
      `git diff origin/main...HEAD -- docs/LOG/`, and that command is
      the check. The entry rides the session commit or the session does
      not land (PRINCIPLES 29). *Ruling, 2026-08-09 /retro, on seven
      arbiter-raised sightings and two counter-examples:* an arbiter
      verdict line reporting a missing LOG entry is a **symptom of this
      step being skipped**, not a role boundary working as designed —
      the earlier reading recorded in the 2026-08-01 #331 entry does not
      survive. The arbiter may still say it; it is under no obligation
      to, and a session that relies on the reminder has already skipped
      the step. Do NOT add a LOG check to `.claude/agents/arbiter.md`,
      and do not build a mechanical guard here (that is #304's subject).

      **(b) `origin/main` has not moved past the verdict's base.** Fetch
      and check `git log HEAD..origin/main` is EMPTY. If it is not, the
      base has drifted and the branch owes, in order:

      - **Merge `origin/main` forward** (never rebase — WORKFLOW forbids
        force-push on a WIP branch, so a merge is the only mechanism);
        follow the merge-conflict ritual below, and **name the absorbed
        SHAs in the log entry and the PR body**.
      - **Re-run the FULL gate on the COMMITTED merged tree — always,
        a conflict-free auto-merge included.** A clean merge is not
        evidence of a compatible merge: #392's fourth merge type-checked
        and introduced a false reject that only the gate found. If an
        absorbed commit changed the gate itself, re-read CLAUDE.md's
        "Commands" block and run the gate it now names, not the one you
        remember (#329 added `go tool commentwrap ./...` mid-branch).
      - **Then buy a second, gate-only arbiter round IF AND ONLY IF the
        verdict rests on a measurement the moved base can invalidate** —
        a banked ratchet figure, a withheld-set count, a lane delta.
        A green gate the session has just re-run is not such a
        measurement, so a doc-only or comment-only landing lands on the
        verdict it already has. (Both halves are required: "always buy a
        round" taxes every cheap landing for a risk it cannot run;
        "re-run and land" silently drops the case where the moved base
        changes what a banked figure means.)

      Re-verify `git log HEAD..origin/main` is empty after the merge —
      main can drift again while the PR is open, and one branch paid
      four merge cycles in a single session.

   1. Open a PR from `wip/issue-<N>` to `main`; the body carries
      `Closes #<N>` plus a pointer to the arbiter's accept verdict.
   2. Squash-merge it via the Merge API (MCP `merge_pull_request` with
      `merge_method: "squash"`, or the platform's built-in PR tools),
      supplying the CLAUDE.md commit format as the squash title
      (`<area>: <what changed> (#<N>)`) and body (`Spec:`/`Ratchet:`
      lines).
   3. GitHub finishes server-side: main gets the ONE session commit,
      `Closes #<N>` closes the issue, and the head branch is
      auto-deleted.

   Nothing else is ever committed directly to main.

7. **Post-land pass** — the **cartographer**, twofold: (a) **unblock**
   — scan `blocked` issues whose `Depends on:` names the just-closed
   issue; any whose dependencies are now ALL closed is relabeled
   `ready`, with a one-line comment naming the landing that unblocked
   it; (b) **harvest follow-ups from this landing** — the session log
   entry's "Next:"/deferred items and the issue thread's advisory
   verdict notes are each filed as an issue or explicitly dismissed in
   a comment, while the context is fresh. The dependency graph and the
   follow-up ledger react to landings immediately instead of waiting
   for the next /backlog.

   **A hand-off is not a disposition.** An advisory that is real but too
   small to carry an issue of its own has exactly two dispositions:
   **filed**, or **written into a comment on the specific issue that
   will absorb it, in the session that raises it**. "Handed to the
   post-land pass as a candidate", "its right home is whichever issue
   next touches X", and "recorded here so a later session can pick it
   up" are NOT dispositions — the post-land pass keeps no ledger to
   check them against, so an item disposed of that way is untracked the
   moment the session ends. #330's two sub-threshold doc-citation
   advisories went out through that phrasing, the pass then ran, and it
   carried neither; two later entries re-stated them as an open leak.
   If no issue is a plausible home, the advisory is above threshold
   after all — file it.

Budget: one issue per session. Nothing works? A checkpointed WIP branch
+ a good RESUME comment is a successful session. Never wait for a human;
abort hanging commands and log the failure.

## Checkpoints, hand-off, and parking

**Checkpoint** (at every step boundary, and before ending any session):

```sh
git add -A && git commit -m "wip #<N>: <step completed>"
git push origin wip/issue-<N>
```

plus a `RESUME:` comment on the issue whenever the next action isn't
obvious from the branch alone:

```
RESUME: <last completed step, e.g. "implementation done, warden passed">
Next: <the exact next action, e.g. "arbiter verdict round 2 — prior
findings were X, Y">
Grounding: see the GROUNDING comment above (re-ask the oracle if absent)
```

The branch carries the CONTENT; the RESUME comment carries the INTENT.
Discovery never depends on the comment — `wip/issue-<N>` is found by
listing the namespace — but a good "Next:" saves the resuming session
from re-deriving where things stood.

The orchestrator's transcript is disposable (compaction may summarize it
at any moment) and so is the container. ALL durable state lives on
GitHub: the issue thread, the pushed WIP branch, and main. Neither
compaction nor a recycled container may be able to eat anything that
can't be rebuilt from those. Wrapping up early at a checkpoint (time
budget hit, second reject) is a first-class outcome, not a failure.

**One writer per checkout** — a `wip/issue-<N>` checkout has exactly one
writer, and it is the orchestrating session. Two writers in one tree is
not a discipline failure that more care avoids; it is undefined
behaviour, because every gate command and every `git commit` honestly
reports on whatever happens to be in the tree at that instant, and the
two writers interleave at moments neither chooses. An auto-commit fired
into a shared checkout while a mason was mid mutation-check (deliberately
breaking lines to confirm a test notices — `.claude/agents/mason.md`,
"Before handoff") and pushed `206ea51` with two of those lines still
broken, costing a full arbiter round (#350). The rules:

- **Only the orchestrating session's own git commands write directly
  into a `wip/issue-<N>` checkout.**
- **Any subagent spawned to mutate code is spawned with worktree
  isolation** — the harness's `isolation: "worktree"` option, which runs
  the subagent in its own git worktree on its own local branch. This is
  mandatory for every **mason** invocation, because a mason may run a
  mutation-check at any point without announcing it. The isolated
  worktree lives in the harness's own worktree area and its branch is
  LOCAL-ONLY, never pushed to `origin`, so it mints no second ref under
  `refs/heads/wip/` (see the branch scheme: one working branch per
  issue).
- **The orchestrator never commits on a live subagent's behalf, for any
  reason** — including in reaction to a stop-hook "uncommitted changes"
  warning, which fires on an in-progress edit exactly as readily as on a
  finished one and is therefore not evidence that a tree is commit-ready
  (that trigger, with no mutation-check running at all, is how the same
  hazard recurred in #296). A subagent's tree is commit-ready only after
  the subagent itself reports completion.
- **After that report**, the orchestrator fast-forward-merges the
  isolated worktree's local branch into its `wip/issue-<N>` checkout and
  discards the worktree — the harness removes it by itself when the
  subagent changed nothing — then checkpoints as normal. The worktree
  never survives the step boundary it was spawned for.

This is a different hazard from the stale index that
`.githooks/pre-commit` guards (see "Merge-conflict resolution" below):
there, one writer commits content it can no longer see; here, a second
writer changes the content under the first. Neither guard substitutes
for the other, and they are not merged.

**Merge-conflict resolution** — the one path where the checkpoint
ritual's `git add -A` is not what actually happens. Conflict resolution
stages file by file, and a merge also stages the files it brought in
cleanly, so any fix made *after* that staging is invisible to
`git commit`:

```sh
git merge origin/main       # step 2's resume merge, or the pre-land merge
# resolve every conflict — AND make the follow-up fixes the merge implies
# in files it staged for you (a symbol the other side renamed, a helper
# that became a method, an import that moved)
git add -u                  # re-stage EVERY touched file, not just the
                            # conflicted ones — always after the LAST edit
git diff --cached --stat    # mandatory: this is the tree that will land
git commit
```

`git diff --cached --stat` is a named step of the ritual, not an optional
sanity check. The whole gate — `go build`, `go test`, `go vet`,
`golangci-lint run`, `go tool commentwrap ./...`, the conformance
suite — plus the arbiter's review
read the WORKING TREE; nothing in them reads the index. A green gate
therefore says nothing about what is actually staged: a fix made after
`git add` gets silently reverted by the commit while every check honestly
reports green. That is how a red `main` landed at `547b42f` through a
squash-merged PR (#179). `.githooks/pre-commit` now refuses any commit
where a path has both staged and unstaged changes, which catches this
mechanically — but only in a session that ran step 1's
`git config core.hooksPath .githooks`, so the discipline stays yours too.

**The `docs/LOG` tail is the conflict every pre-land merge hits**, because
every session appends to the same month file at the same point, and it is
the one file where a plausible-looking resolution can silently delete
someone else's landed history (the log is append-only — PRINCIPLES 29).
Both sides are almost always PURE APPENDS, so the resolution is
positional and nothing else: base, then the other side's entries in the
other side's own order, then yours, each byte-identical to its authored
form. Nothing is reflowed, reordered, merged or "tidied". Then prove it
rather than counting headings — reconstruct each parent's version of the
file from the resolved one and compare:

```sh
git show :2:docs/LOG/<yyyy>-<mm>.md > /tmp/ours   # or the merge parents'
git show :3:docs/LOG/<yyyy>-<mm>.md > /tmp/theirs # blobs, pre-resolution
# each must appear in the resolved file as an unbroken, unedited prefix/run
```

Heading arithmetic passes while an entry is silently dropped; a
byte-for-byte reconstruction of both parents cannot. Do not read the
scale of the drift off the expectations diff either — a landing can
append a LOG entry while touching no lane file at all, so a resolver
reasoning from `conformance/testdata/expectations/` under-counts what it
must preserve. (That the month file is ONE append point, and therefore
forces this merge on finished branches with zero logic overlap, is a
layout question tracked separately as #600.)

**Park** (second reject, or a resume whose merge won't resolve):
checkpoint the branch one final time, label the issue `needs-replan`,
and comment the findings that killed the attempt. Nothing is renamed or
deleted — the `needs-replan` label alone retires the branch in place,
where it stays as re-planning evidence, not resumable work. After
re-planning, the cartographer closes the issue as superseded and files
a replacement; the fresh attempt starts as `wip/issue-<M>` under the
new number, from `origin/main`.

## Other triggers

Every maintenance trigger lands the same way `wip/issue-` work does: it
commits on whatever branch it is on, then opens a PR and squash-merges it
via the GitHub Merge API in the SAME session, before that session ends. A
trigger session must NOT end with a commit sitting on an unmerged branch —
that is how commits get stranded off `main`. These sessions land
same-session, so they need no `wip/*`-style lease or branch-discovery
machinery; just don't leave the merge undone.

- **`/ratchet`** — arbiter only: run conformance, report movement per
  lane, ratchet upward, investigate & file issues for any regression.
- **`/backlog`** — cartographer: reconcile GitHub issues with reality (close
  stale, split oversized, order by dependency). No numeric `ready` cap
  (#347) — the dependency-ordered band published in docs/PLAN.md is the
  working queue; `.claude/agents/cartographer.md` states the rule.
  **The launching session — not the cartographer — runs
  libuser/cliuser** when the pass covers API- or CLI-facing milestones,
  and hands their stories to the cartographer to fold into issue bodies.
  A cartographer subagent has no way to spawn them, and role-playing a
  persona it could not isolate itself from is worse than skipping the
  step (#416). Also **reconcile the branch namespace**: classify every
  `wip/*` branch by its issue's state (live / resumable / retired); a
  `wip/` branch stale for several days with no RESUME comment gets its
  issue flagged `needs-replan`; a closed issue's leftover branch is
  verified landed (superseded if it isn't); retired branches and
  `parked/untriaged-*` are listed in the plan summary for human triage
  — never deleted by an agent.
- **`/story`** — cartographer interviews libuser and cliuser (feeding
  them only the current README and `go doc` output) to produce user
  stories with acceptance criteria, filed as issues.
- **`/retro`** — chronicler: read the last ~2 weeks of LOG + git history +
  `needs-replan` issues + verdict comments on issue threads; find
  recurring friction and classify it by pipeline entry point; audit the
  follow-up ledger (every promised follow-up filed or dismissed); apply
  the smallest durable fix to WORKFLOW/STYLE/agent prompts in a
  `meta: retro` commit; log metric trends vs the previous retro.
  Then Part 2, the **architecture audit** — delegate to the
  **steward**: import graph and exported surface vs
  docs/ARCHITECTURE.md; placement, duplicate concepts/representations
  (judged by upkeep cost), exported-symbol usage vs godoc intent, doc
  drift. Files `kind/refactor` issues ranked by cost-of-delay;
  pre-1.0 movement is encouraged, post-1.0 the audit guards the
  surface (docs/PLAN.md defines the line).
  **The steward cannot file its own issues or merge its own PR.** Its
  tool set is deliberately Read/Grep/Glob/Bash only (it must not touch
  Go code or the tracker directly), and GitHub has been unreachable from
  that context on both audits run so far (2026-07-26, 2026-08-02). So the
  hand-off is the intended seam, not a workaround: the steward returns
  issue-ready write-ups and pushes its `meta: audit <date>` doc commit to
  a branch, and the **orchestrating session** files the issues, opens the
  PR and squash-merges it. Budget for that step when delegating; an audit
  that "filed nothing" because the subagent had no GitHub is an
  orchestration miss, not a steward finding.
  The ratchet-integrity rules (CLAUDE.md's one rule, arbiter.md's
  ratchet section) change only via a human-filed issue.

See docs/ROUTINES.md for the schedule. Every trigger is a slash command
you can also run locally on demand.

## GitHub conventions

- **Labels**: `ready` (unblocked, sized for one session), `blocked`,
  `needs-replan`, `epic`; areas
  `area/{model,xsderr,parser,value,builtin,xpath,validate,codegen,codec,regex,loader,conformance,cli,meta}`;
  kinds `kind/{feature,gap,bug,refactor,process,tooling,story}`.
- **Milestones** mirror docs/PLAN.md (M1, M2, …).
- Issue body must contain: goal, spec references (rule IDs), acceptance
  criteria (which conformance cases / tests prove it), and dependencies
  (`Depends on #N`). If an agent can't start it from the body alone, the
  body is incomplete.
- For an issue that regenerates a spec-derived table (`xsderr/catalog.go`,
  the spec `.md` files, any generated Go table), do NOT make "regenerated
  output is byte-identical to the committed file" an acceptance criterion
  unless a spec grep first confirms the committed table already contains
  every member the widened generator will admit. Regeneration frequently
  and correctly ADDS previously-missed entries — catching those is often
  the whole point — so a byte-identical criterion is usually vacuous or
  self-contradictory. The right criterion is "the regenerated diff is
  additive-only and every added entry is a genuine spec member," verified
  by grep.
- **Filing discipline — a defective issue BODY outlives the session that
  filed it** (#315; six-plus landings paid for each clause below). Before
  filing a new issue, and again before grounding one already filed:
  - **A stale or wrong premise is corrected IN THE BODY.** When someone
    finds a body asserting something no longer true — or never true — the
    fix edits that body; a thread comment alone does not retire the
    premise, because the next reader starts from the body. The comment
    that found it stays as provenance, it just isn't the only place the
    correction lives. When rewriting a body, keep angle-bracket tags and
    bare-URL autolinks out of the replacement text: GitHub
    entity-escapes both when a body is stored, and an edit round-trips
    unchanged as long as the new text introduces neither.
  - **A runtime-mechanism claim states whether it was reproduced.** A
    body asserting what the code currently does (a mechanism in the tree,
    not a spec rule) either says the assertion was reproduced against the
    tree, or is written as a hypothesis pending verification. A memory of
    prior discussion is not reproduction and must not be filed as settled
    fact.
  - **Every static citation is checked against the tree before filing.**
    That covers every rule ID — a docs/STYLE.md `T`/`D` number, an XSD
    rule name, a spec section — and every file path named as a consumer
    or a target. Three sharpenings, each of which caught a real defect a
    generic "check your citations" habit missed: (a) a citation must
    resolve to the right section by KIND, not merely by number — a
    `cvc-*`-style Validation Rule ID belongs in the spec's "… Validation
    Rules" subsection and a Schema Component Constraint in the sibling
    "Constraints on … Schema Components" subsection, so the right number
    under the wrong-kind subsection is still a wrong citation; (b)
    CLAUDE.md's headline-bullet numbering and docs/STYLE.md's T-series
    are two different numbering spaces over overlapping content — cite
    the docs/STYLE.md letter+number ID (`STYLE T1`), never the CLAUDE.md
    headline bullet number, and never conflate them; (c) to point at a
    specific site, prefer a `GAP(...)` marker's TEXT or the enclosing
    identifier's name over a line number, which later unrelated edits
    move.
  - **Search the open queue for overlap, and record what the hit was.**
    Search open issues for the primary file path and the primary
    identifier the change is about. A hit means one of exactly two
    things: the same change — close one as duplicate and say which in a
    comment; or an adjacent change — post a one-line cross-reference on
    both issues. Never proceed past a hit without recording which of the
    two it was.
  - **An `## Acceptance` ratchet promise names its condition.** A clause
    promising a lane will move states what that promise depends on
    ("moves the `schema` lane **provided** #N has landed"), so that a
    later planning pass chartering #N's prerequisites differently leaves
    the staleness one grep away instead of requiring planning history to
    be reconstructed.
- `// GAP(...)` comments and fail-open sites get tracking issues
  (`kind/gap`) so nothing fails open silently forever.

## Commit format

```
<area>: <what changed> (#<issue>)

Spec: <rule ids, or "n/a">
Ratchet: <lane movement, or "unchanged">
```

Small, focused, independently revertible. Ratchet expectation updates are
part of the same commit as the fix that earned them. The `Ratchet:` line
of the LANDED commit is the arbiter's figure, not the branch's prediction
— see "The ratchet" below.

## The ratchet (the heart of the process)

- `conformance/testdata/expectations/*.txt`: one line per W3C test case,
  `<case-id> <expected-outcome>`, sorted, committed, one lane per file.
- `go test ./conformance -run TestConformance -count=1` fails if any case
  does worse than its expectation.
- The same run under `GOXSD_RATCHET=1` rewrites expectations for cases
  that now do better, refuses to write anything worse.
- Every expectation change must be explainable; "it flipped and I don't
  know why" blocks the commit and becomes an issue.
- **A line leaves a lane file only as a sanctioned applicability
  removal** (issue #576, repo-owner ruling): discovery stopped producing
  the case because the W3C suite's own `@version` metadata scopes it away
  from an XSD 1.1 processor, so it was never ours to score. That is not a
  `Vanished` regression — but it is banked only when the runner itself
  withheld the case AND the arbiter's ratchet run asserted the per-lane
  count, `GOXSD_RATCHET_REMOVALS=schema=34,instance=65`. Any other number
  refuses the whole merge; a read-only run only logs removals. Genuine
  `Regressed` and `Vanished` cases still abort the merge whatever the
  assertion says. Details in `conformance/testdata/expectations/README.md`.
- **A read-only conformance PASS is evidence of NO REGRESSION only, never
  of the whole ratchet write.** `Compare` fails on `Regressed` cases;
  `Improved` cases still pass, but the read-only run now LOGS them, so
  `go test ./conformance -run TestConformance -count=1 -v` lets a
  non-arbiter measure the improvement half of the movement rather than
  guess at it (issue #303). What remains invisible read-only is the `New`
  set — cases carrying no expectation line yet, which a `GOXSD_RATCHET=1`
  run also writes. So a `Ratchet:` line written by anyone but the arbiter
  is still a **prediction** of the write as a whole, even when its
  improvement half is measured; only the arbiter's ratchet run over a
  clean tree observes all of it.
- **The landed `Ratchet:` line carries only what the arbiter's accept
  verdict states, in the verdict's own terms.** Branch commits may and
  should predict — a prediction that names the cases it expects to flip
  and why is what hands the arbiter its evidence — but at squash time the
  prediction is REPLACED, not confirmed, and any figure the verdict does
  not state (per-lane totals in particular) is dropped rather than
  carried forward. "Unchanged because the diff touches no lane's code
  path" is an inference; the arbiter's ratchet run over a clean tree is
  an observation, and only the second belongs in the squash message.

## Debugging playbook (for agents)

- Reproduce one failing conformance case in isolation before touching code
  (the harness supports single-case runs; see conformance's doc.go).
- Turn on scoped debug logs (`GOXSD_DEBUG=validate,xpath go test ...`) —
  messages carry rule IDs and locations by design.
- For bulk failure analysis, write an env-gated throwaway diagnostic test
  (`zz_diag_test.go`, gated on `DIAG=1`), harvest the pattern, delete it
  (PRINCIPLES 23).
- Grep the spec (`docs/specs/md/`), not your memory. Quote clauses in
  issues and commits.
- Friction with a manual process twice in a row? File a `kind/tooling`
  issue proposing a repo tool (PRINCIPLES 27).
- Never snapshot or copy the working tree with ad-hoc `cp`/backup commands
  into or beside the repo root during conformance review — a stray
  `cp conformance/testdata/expectations/* .` once clobbered the tracked
  root `README.md` and littered lane `*.txt` files at repo root (#104). If
  you genuinely need a working copy, put it well outside the repo (the
  scratchpad dir) or use `git worktree`.
