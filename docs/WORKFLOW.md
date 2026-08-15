# Development Workflow

The rules every goxsd8 session obeys, whatever command it is running. The
commands themselves live in `.claude/commands/`; this file does not repeat
them.

goxsd8 is developed by Claude Code sessions — scheduled routines and
on-demand local runs execute the **same slash commands**, so automated and
interactive behavior are identical by construction. A session is short;
the repo and its GitHub issues are the brain.

## What survives a session

**The container is ephemeral.** A run may start from a fresh clone: local
git state — stashes, dirty trees, local-only branches, scratch dirs — does
not survive. Hence **anything not pushed does not exist** (PRINCIPLES 28).

Three things are durable, and everything a later session needs must be in
one of them:

- **The issue thread** — groundings, verdicts, RESUME notes, decisions.
- **`docs/LOG/<year>-<month>.md`** — what happened and what it cost.
- **Pushed branches and `main`** — the code.

A transcript is not durable. Neither is a local path: never point at one
in a comment.

For issue operations use whichever GitHub channel the session has
(docs/ROUTINES.md ranks them): the platform's built-in GitHub tools, the
GitHub MCP server, or the `gh` CLI.

## The branch scheme (the WIP discovery index)

The remote branch namespace is a machine-readable index. Any session
reconstructs the in-flight state with one command:

```sh
git ls-remote --heads origin 'refs/heads/wip/*' 'refs/heads/parked/*'
```

**Read the remote, not the local remote-tracking refs.** GitHub deletes a
branch at merge, but `refs/remotes/origin/wip/*` keeps a stale entry until
pruned, so `git for-each-ref` invents in-flight work. Fetch with
`--prune`; `go tool wipsurvey` reads the remote regardless.

| Branch | Meaning | Lifecycle |
|---|---|---|
| `main` | always green; receives only squash-merged PRs | permanent |
| `wip/issue-<N>` | THE work branch for issue #N — at most one, name is stable | created when work starts; auto-deleted by GitHub on merge; retired in place if abandoned |
| `parked/untriaged-<YYYYMMDD-HHMMSS>` | unattributable work found in a dirty local tree | kept for human triage |

A maintenance command lands from a short-lived branch of its own; it opens
and squash-merges its PR in the same session, so it holds no lease and
never appears in a survey.

Invariants:

- **The name is the claim; the tip time is the lease.** `wip/issue-<N>`
  existing means #N has an in-flight attempt. Whether that attempt is
  alive is the tip's timestamp
  (`git log -1 --format=%cI origin/wip/issue-<N>`) against the **2h claim
  TTL**: newer is LIVE — off-limits, and so is its issue; older is
  EXPIRED — resumable. Checkpoint pushes are therefore the lease
  heartbeat; a long step pushes intermediate commits rather than letting
  its lease lapse. A branch that has pushed **no commits of its own** has
  no tip time of its own — its tip is the landing it branched from, and
  only its issue thread dates the claim, so it is never EXPIRED and never
  resumable on age (#722).
- **Races are settled by git's atomic ref updates, never by force.** A
  rejected push to `wip/*` means you lost the race: fetch, abandon the
  local attempt, pick something else. Force-pushing `wip/*` or `parked/*`
  is forbidden — it is the one way sessions could stomp each other.
  Sessions only ever CREATE refs; the single deletion in the system is
  GitHub's auto-delete on merge.
- **A `wip/` branch is work only while its issue is open and not
  `needs-replan`.** Otherwise it is retired: landed branches vanish on
  merge, abandoned ones stay as re-planning evidence. A retired name is
  never contended, because an issue is never re-attempted under its own
  number — re-planning supersedes it with a new issue, and the fresh
  attempt starts as `wip/issue-<M>` from `origin/main`.
- **Freshly-fetched `origin/main` is the only base.** Every diff, merge
  and branch point is taken against `origin/main` after an explicit
  `git fetch origin main` in THIS session; a local `main` in an ephemeral
  container is stale, and diffing against it invents changes that are not
  there while hiding ones that are. `git diff origin/main...HEAD` is the
  shape that works. Equally: judge the COMMITTED tree — if
  `git status --porcelain` is non-empty, what was verified is not what
  will land.

## Checkpointing and hand-off

**Checkpoint at every step boundary, and before ending any session:**

```sh
git add -A && git commit -m "wip #<N>: <step completed>"
git push origin wip/issue-<N>
```

Add a `RESUME:` comment on the issue whenever the next action is not
obvious from the branch alone — last completed step, the exact next
action, and where the grounding is. The branch carries the CONTENT; the
comment carries the INTENT. Discovery never depends on the comment.

Never destroy uncommitted work: `git clean`, `git restore .`,
`git checkout -- <file>` and stashing of any kind are forbidden. A dirty
local tree is pushed to `parked/untriaged-<ts>` and logged.

Ending early at a checkpoint — time budget, second reject, a wall — is a
first-class outcome, not a failure.

## One writer per checkout

A `wip/issue-<N>` checkout has exactly one writer: the orchestrating
session. Two writers in one tree is undefined behavior, not a discipline
problem — every gate command and every commit reports whatever is in the
tree at that instant, and the writers interleave at moments neither
chooses (#350).

- Only the orchestrating session's own git commands write into the
  `wip/issue-<N>` checkout.
- **Any subagent spawned to mutate code gets worktree isolation** — the
  harness's `isolation: "worktree"` option. Mandatory for every **mason**
  invocation, because a mason may break lines deliberately at any moment
  to check that a test notices. Its branch is local-only and never pushed.
- The orchestrator never commits on a live subagent's behalf, for any
  reason — including a stop-hook "uncommitted changes" warning, which
  fires on an in-progress edit exactly as readily as a finished one
  (#296). A subagent's tree is commit-ready only once it reports.
- After that report, fast-forward-merge the isolated branch into
  `wip/issue-<N>` and discard the worktree, then checkpoint.

The orchestrator holds the pen itself only when the edit carries no design
content and its scope is provable: a change a review verdict specifies
verbatim, or text no compiler reads. Everything else is a mason round.

**Every orchestrator edit is a new commit.** Amend and force-push of a
pushed `wip/*` or `parked/*` ref are forbidden without exception —
including one's own last commit, including to fix a commit message. An
amended pushed ref leaves the judge no baseline to check the account
against once the container is gone (#649). Disclose a breach on the thread
and record it in the log; it is a process note the orchestrator owns, not
a finding against the diff.

## Scope: what one landing carries

A landing carries its issue's change plus whatever that change makes
necessary. Finding adjacent work mid-implementation is normal and
expected. The question is never "was this in the issue body" but **"does
this need its own grounding, its own surface review, or its own ratchet
attribution?"**

- Needs a rule ID the thread's GROUNDING does not cover → separate issue.
- Adds exported surface the warden has not seen → warden pass now, or
  separate issue.
- Could move a conformance lane on its own → separate issue, so the
  movement stays attributable (PRINCIPLES 22).
- None of those → **absorb it**, and name it in the commit body so the
  arbiter reads it as intended scope rather than creep.

Deferring is not free: a follow-up costs a whole landing and opens a
hand-off where premises decay. A rename, an unexport, a stale comment, a
call site the change itself breaks — all are cheaper absorbed than filed.
Judge the diff by whether a reviewer can hold it at once, not by whether
it matches the body.

**Ordering is not splitting.** A review may find that one part of a change
must precede another. That is satisfied by commit order on one branch.
Split into a second issue only when the later part needs its own grounding
or will not fit the session; otherwise land both and close both from the
one PR.

## Landing

Landing is atomic. Accept → open a PR from the branch and squash-merge it
via the GitHub Merge API. The squash is the session's ONE commit (code
plus log entry), `Closes #<N>` in the body closes the issue, and GitHub
auto-deletes the head branch (keep the repo's "Automatically delete head
branches" setting ON). Nothing is ever committed directly to `main`.

Two preconditions, verified and stated by the orchestrating session before
the PR is opened — neither is anyone else's to volunteer:

1. **The LOG entry is in the branch's diff** — literally
   `git diff origin/main...HEAD -- docs/LOG/`, not "the chronicler was
   invoked". The entry rides the session commit or the session does not
   land (PRINCIPLES 29).
2. **`origin/main` has not moved past the verdict's base** —
   `git log HEAD..origin/main` is empty. If it is not, the branch owes, in
   order: **merge `origin/main` forward** (never rebase — force-push is
   forbidden, so a merge is the only mechanism), naming absorbed SHAs in
   the log entry and PR body; **re-run the full gate on the committed
   merged tree**, auto-merges included, because a clean merge is not
   evidence of a compatible one (#392); then re-judge per **After the
   verdict**. Re-verify afterwards: main can drift again while the PR is
   open.

If an absorbed commit changed the gate itself, re-read CLAUDE.md's gate
block and run what it names now, not what you remember.

### After the verdict

A verdict measures a tree, and the tree that lands must be that tree.
Anything that moves it — a merge forward, a late finding, a fix turned up
while writing the log — is judged by one question: can this change what
the verdict measured?

- **The diff changed** → a full new round.
- **Only the base moved** → a gate-only round, and only when the verdict
  rests on a measurement that base can invalidate: a banked ratchet
  figure, a lane delta. A green gate just re-run is not such a
  measurement.
- **Provably nothing measured changed** → land it. Only text no compiler
  reads qualifies, proved mechanically: the diff since the accepted commit
  touches no line outside a comment or a `.md` file. It is its own commit,
  carrying that proof in its body; re-run gate parts 1–3 and move no
  ratchet figure.

What no class admits goes to the post-land ledger.

## Merge-conflict resolution

The one path where the checkpoint ritual's `git add -A` is not what
happens. Conflict resolution stages file by file, and a merge also stages
what it brought in cleanly, so any fix made *after* that staging is
invisible to `git commit`:

```sh
git merge origin/main
# resolve every conflict — AND make the follow-up fixes the merge implies
# in files it staged for you (a renamed symbol, a moved import)
git add -u                  # re-stage EVERY touched file, after the LAST edit
git diff --cached --stat    # mandatory: this is the tree that will land
git commit
```

`git diff --cached --stat` is a named step, not a sanity check: the whole
gate reads the working tree and nothing reads the index, so a fix made
after `git add` is silently reverted by the commit while every check
reports green (#179). `.githooks/pre-commit` refuses a commit where a path
has both staged and unstaged changes — activate it every session, since
local git config does not survive a fresh clone:

```sh
git config core.hooksPath .githooks
```

**The `docs/LOG` tail is the conflict every pre-land merge hits**, and the
one file where a plausible resolution silently deletes landed history.
Both sides are almost always pure appends, so the resolution is positional
and nothing else: base, then the other side's entries in their own order,
then yours, each byte-identical to its authored form — nothing reflowed,
reordered or tidied. Then prove it by reconstructing each parent from the
resolved file (`git show :2:<path>`, `git show :3:<path>`); each must
appear as an unbroken, unedited run. Heading arithmetic passes while an
entry is silently dropped. (#600 tracks the single-append-point layout.)

## Parking

On a second arbiter rejection, or a resume whose merge will not resolve:
checkpoint the branch one final time, label the issue `needs-replan`, and
comment the findings that killed the attempt. Nothing is renamed or
deleted — the label alone retires the branch in place as re-planning
evidence. **Two rejections is the hard cap** (PRINCIPLES 30); never
solicit a third round. After re-planning, the cartographer closes the
issue as superseded and files a replacement.

## GitHub conventions

**Labels**: `ready` (unblocked, sized for one session), `blocked`,
`needs-replan`, `epic`; `area/{model,xsderr,parser,value,builtin,xpath,`
`validate,codegen,codec,regex,loader,conformance,cli,meta}`;
`kind/{feature,gap,bug,refactor,process,tooling,story}`. Milestones mirror
docs/PLAN.md. `blocked` means waiting on a named dependency recorded in
`## Depends on` — an issue or a trigger, not only an open issue.

**The body states the decision; the thread holds the reasoning.** A body
carries goal, spec rule IDs, acceptance, surface and dependencies — enough
that an agent can start from it alone — and stays short enough to read at
once. Verdicts, groundings and analysis belong in comments; transcribing
them into the body is what produces issues longer than this document.

**Filing discipline** — a defective body outlives the session that filed
it (#315). Before filing, and again before grounding something already
filed:

- **Correct a stale or wrong premise in the body**, not only in a comment;
  the next reader starts from the body. The comment stays as provenance.
- **State whether a runtime-mechanism claim was reproduced** against the
  tree, or write it as a hypothesis. Memory of prior discussion is not
  reproduction.
- **Check every citation against the tree** — rule IDs, STYLE letter IDs,
  file paths. A citation must resolve by KIND as well as number (a `cvc-*`
  Validation Rule lives in a different subsection from a Schema Component
  Constraint), and CLAUDE.md's headline numbers are not STYLE IDs. To
  point at a site, prefer a `GAP(...)` marker's text or the enclosing
  identifier over a line number.
- **Search the open queue** for the primary file path and identifier. A
  hit is either a duplicate (close one, say which) or an adjacent issue
  (cross-reference both). Never pass a hit silently.
- **An `## Acceptance` ratchet promise names its condition** ("moves the
  `schema` lane **provided** #N has landed"), so a later re-plan leaves the
  staleness one grep away.

`// GAP(...)` and fail-open sites get `kind/gap` tracking issues, so
nothing fails open silently forever.

For a table that regenerates spec-derived output, do not make
"byte-identical regeneration" an acceptance criterion unless a grep first
confirms the committed table already holds every member the widened
generator admits — regeneration correctly ADDS missed entries, which is
usually the point. The right criterion is "additive-only, and every added
entry is a genuine spec member."
