#!/usr/bin/env bash
# Apply a RESULT.md classification plan with gh-starlist.
#
#   apply-result.sh [RESULT.md] [--apply]
#
# Without --apply nothing is mutated: the plan is validated and the commands
# that would run are printed. Override the CLI with GH_STARLIST=./gh-starlist.
set -euo pipefail

RESULT=RESULT.md
APPLY=0
BATCH=${BATCH:-100}
LIST_CAP=32

for arg in "$@"; do
  case "$arg" in
    --apply) APPLY=1 ;;
    -h|--help) awk 'NR > 1 { if ($0 !~ /^#/) exit; sub(/^# ?/, ""); print }' "$0"; exit 0 ;;
    -*) echo "unknown flag: $arg" >&2; exit 2 ;;
    *) RESULT=$arg ;;
  esac
done

STARLIST=${GH_STARLIST:-gh starlist}
[ -f "$RESULT" ] || { echo "no such file: $RESULT" >&2; exit 1; }
command -v jq >/dev/null || { echo "jq is required" >&2; exit 1; }

run() {
  if [ "$APPLY" = 1 ]; then
    $STARLIST "$@" </dev/null
  else
    printf 'would run: %s' "$STARLIST"
    printf ' %q' "$@"
    printf '\n'
  fi
}

TMPDIR_=$(mktemp -d "${TMPDIR:-/tmp}/apply-result.XXXXXX")
trap 'rm -rf "$TMPDIR_"' EXIT

# --- reconcile against the account ------------------------------------------
$STARLIST list --json | jq -r '.[] | [.name, .slug] | @tsv' > "$TMPDIR_/existing.tsv"

# --- parse RESULT.md and build the plan --------------------------------------
# Everything below is deliberately awk rather than bash: associative arrays,
# ${var,,} and mapfile are all bash 4+, and macOS ships bash 3.2.
# Section headings are accepted in English and Japanese.
# The summary goes to stdout; the plan is written to $plan as a tab separated
# stream of CREATE / HEADER / ADD records for the executor loop below.
awk -v batch="$BATCH" -v cap="$LIST_CAP" -v result="$RESULT" -v plan="$TMPDIR_/plan.tsv" '
BEGIN { FS = "\t" }

# first file (ARGV[1]): existing lists on the account, as name<TAB>slug.
# Comparing FILENAME beats the FNR==NR idiom, which misfires on an empty file.
FILENAME == ARGV[1] {
  exists[tolower($1)] = $1
  exists[tolower($2)] = $1
  ecount++
  next
}

/^## / {
  insec = ""
  if ($0 ~ /^## New Lists/) insec = "new"
  else if ($0 ~ /^## (Destination|Destinations)/) insec = "dest"
  next
}
insec == "" || $0 !~ /^- / { next }

{
  line = substr($0, 3)
  p = index(line, ": ")
  if (p) { head = substr(line, 1, p - 1); tail = substr(line, p + 2) }
  else   { head = line; tail = "" }

  if (insec == "new") {
    key = tolower(head)
    if (!(key in newname)) {
      newname[key] = head; newdesc[key] = tail; neworder[++nn] = key
    }
    next
  }

  # insec == "dest": "owner/repo: List Name"
  if (!p || index(head, "/") == 0) {
    printf "malformed line under Destination: - %s\n", line > "/dev/stderr"
    bad = 1; exit 1
  }
  rk = tolower(head)
  if (rk in seen) { dupes++; next }
  seen[rk] = 1; nrepos++
  key = tolower(tail)
  if (!(key in destname)) { destname[key] = tail; destorder[++nd] = key }
  repolist[key] = repolist[key] head " "
}

END {
  if (bad) exit 1
  if (nd == 0) {
    printf "no '"'"'## Destination'"'"' entries in %s\n", result > "/dev/stderr"
    exit 1
  }

  # A destination may be spelled as a name or as a slug; fold both onto the name
  # so one list never gets two `add` calls.
  for (i = 1; i <= nd; i++) {
    k = destorder[i]
    if (k in exists) destname[k] = exists[k]
    lk = tolower(destname[k])
    if (!(lk in uniq)) { uniq[lk] = 1; nuniq++ }
  }

  missing = 0
  for (i = 1; i <= nd; i++) {
    k = destorder[i]
    if (!(k in exists) && !(k in newname)) {
      if (!missing++) print "these destinations are neither existing lists nor under '"'"'## New Lists'"'"':" > "/dev/stderr"
      printf "  - %s\n", destname[k] > "/dev/stderr"
    }
  }
  if (missing) exit 1

  ncreate = 0
  for (i = 1; i <= nn; i++) {
    k = neworder[i]
    if (!(k in exists)) create[++ncreate] = k
  }

  total = ecount + ncreate
  printf "existing lists: %d   to create: %d   total: %d (cap %d)\n", ecount, ncreate, total, cap
  note = dupes > 0 ? sprintf("   duplicate lines skipped: %d", dupes) : ""
  printf "repositories:   %d across %d lists%s\n", nrepos, nuniq, note
  if (total > cap) {
    fflush()
    printf "aborting: %d lists exceeds GitHub'"'"'s cap of %d. Consolidate %s first.\n", total, cap, result > "/dev/stderr"
    exit 1
  }
  print ""

  for (i = 1; i <= ncreate; i++) {
    k = create[i]
    printf "CREATE\t%s\t%s\n", newname[k], newdesc[k] > plan
  }

  for (i = 1; i <= nd; i++) {
    nm = destname[destorder[i]]
    lk = tolower(nm)
    if (lk in merged) continue
    merged[lk] = 1
    all = ""
    for (j = 1; j <= nd; j++) if (destname[destorder[j]] == nm) all = all repolist[destorder[j]]
    n = split(all, repo, " ")
    printf "HEADER\t%s\t%d\n", nm, n > plan
    for (s = 1; s <= n; s += batch) {
      chunk = ""
      for (t = s; t < s + batch && t <= n; t++) chunk = chunk (chunk == "" ? "" : " ") repo[t]
      printf "ADD\t%s\t%s\n", nm, chunk > plan
    }
  }
}
' "$TMPDIR_/existing.tsv" "$RESULT"

# --- execute -----------------------------------------------------------------
[ -f "$TMPDIR_/plan.tsv" ] || : > "$TMPDIR_/plan.tsv"
while IFS=$'\t' read -r tag a b; do
  case "$tag" in
    CREATE) if [ -n "$b" ]; then run create "$a" -d "$b"; else run create "$a"; fi ;;
    HEADER) echo "== $a ($b repos)" ;;
    ADD)    IFS=' ' read -r -a repos <<<"$b"; run add "$a" "${repos[@]}" ;;
  esac
done < "$TMPDIR_/plan.tsv"

if [ "$APPLY" = 0 ]; then
  echo
  echo "dry run only — re-run with --apply to execute"
fi
