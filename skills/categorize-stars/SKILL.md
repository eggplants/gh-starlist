---
name: categorize-stars
description: Sort GitHub starred repositories into star lists with gh-starlist. Exports every star to STARS.md, classifies the Uncategorized ones into existing or new lists, writes the plan to RESULT.md, and applies it with `gh starlist`. Also backfills missing list descriptions and consolidates lists to fit GitHub's 32-list cap.
---

# Categorizing starred repositories

Sorting stars is a four-stage pipeline. **Plan first, apply second** — the
classification is written to `RESULT.md` and reviewed before a single API call
mutates anything.

```
0. prepare  →  1. describe lists  →  2. export  →  3. classify  →  4. apply
```

Run stages in order. If the user asks for only part of it (e.g. "apply
RESULT.md"), jump straight to that stage.

## Hard limits

- **A GitHub account may hold at most 32 star lists.** Existing + new must stay
  ≤ 32. When a draft plan exceeds it, merge the closest categories and *propose
  the consolidation to the user before applying it* — never silently drop
  repositories or lists.
- `gh starlist add` rewrites a repository's whole list membership, so it walks
  every list the user owns on each invocation. **Batch all repositories for one
  list into a single `add` call.** One call per repository is unusably slow.
- A repository can belong to several lists, but this workflow assigns exactly
  one destination per repository unless the user says otherwise.

## 0. Prepare

```bash
gh extension install eggplants/gh-starlist   # skip if already installed
gh auth status | grep -qF "'user'" || gh auth refresh -h github.com -s user
```

Reading lists works with default scopes; **creating or editing them needs the
`user` scope.** If a write fails with a permission error, this is why.

## 1. Backfill list descriptions

Lists without a description make classification guesswork, so fix them first.

```bash
gh starlist list --json | jq -r '.[] | select(.description == "") | .slug'
```

For each slug, look at what is actually in it, then write a description:

```bash
gh starlist view <slug> -L 10
gh starlist edit <slug> -d "<one-line description of what belongs here>"
```

Descriptions are the classification rubric — write them as membership rules
("Command-line utilities and terminal user interfaces."), not as labels.

## 2. Export

```bash
gh starlist export -o STARS.md
```

`STARS.md` holds one `## <List name>` section per list plus an
`## Uncategorized` section of starred repositories in no list. Each row is
`| [owner/repo](url) | description | ⭐stars |`.

The file is large (hundreds of KB). Read the list headings and the
`Uncategorized` section rather than the whole file:

```bash
grep -n '^## ' STARS.md                                   # sections
sed -n '/^## Uncategorized/,$p' STARS.md | wc -l          # size of the work
```

## 3. Classify → RESULT.md

Assign every repository under `## Uncategorized` to a destination list, then
write `RESULT.md`. **Do not touch `gh starlist` in this stage** — this is a
plan, and the user reviews it.

`RESULT.md` has exactly two sections, and the applier script parses them:

```markdown
# Classification

## Destination

- owner/repo: <list name>
- owner/repo: <list name>

## New Lists

- <list name>: <english description>
```

Rules:

- One line per repository, in the same order as `STARS.md`. Nothing is skipped;
  if a repository fits nowhere, give it a list rather than dropping the line.
- `<list name>` is the display name (e.g. `CLI/TUI`), matched case-insensitively
  against name or slug at apply time. Every name used under `## Destination`
  must be either an existing list or listed under `## New Lists`.
- New-list descriptions are in English and state what belongs in the list.
- Prefer an existing list over a new one; only create a list when a coherent
  group of ≥ 5 repositories has no home.
- Count `existing + new` before finishing. Over 32 → merge the nearest
  categories (e.g. `Anime/Manga` → `Japanese`, `Nix` → `DevOps/Infra`), record
  what was merged into what at the top of `RESULT.md`, and present the
  consolidation to the user.

Work through the Uncategorized section in chunks and append to `RESULT.md` as
you go — a few thousand repositories will not fit in one pass.

## 4. Apply

Only after the user has reviewed `RESULT.md`.

```bash
.claude/skills/categorize-stars/scripts/apply-result.sh RESULT.md          # dry run
.claude/skills/categorize-stars/scripts/apply-result.sh RESULT.md --apply  # execute
```

The script creates the missing lists, refuses to run if the total would exceed
32, groups repositories by destination, and issues one `gh starlist add` per
list. Show the dry run to the user before passing `--apply`.

It folds a destination written as a slug onto the list of the
same name, skips repeated repositories, and fails loudly when a destination is
neither an existing list nor declared under `## New Lists`. `GH_STARLIST` picks
a different CLI (`GH_STARLIST=./gh-starlist` for a local build) and `BATCH` sets
how many repositories go into one `add` call (default 100).

Doing it by hand instead:

```bash
gh starlist create "AI/LLM" -d "LLM inference engines, coding agents and MCP servers."
gh starlist add "AI/LLM" ggml-org/llama.cpp vllm-project/vllm openai/whisper
```

`add` prints `<repo>: added "<list>"` or `<repo>: no change`, so re-running it
is safe. Afterwards, confirm the result:

```bash
gh starlist export -o STARS.md
grep -c '^| \[' <(sed -n '/^## Uncategorized/,$p' STARS.md)   # what is left
```
