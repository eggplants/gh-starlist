# gh-starlist

[![Go Reference](
  <https://pkg.go.dev/badge/github.com/eggplants/gh-starlist.svg>
)](
  <https://pkg.go.dev/github.com/eggplants/gh-starlist>
) [![ci](
  <https://github.com/eggplants/gh-starlist/actions/workflows/ci.yml/badge.svg>
)](
  <https://github.com/eggplants/gh-starlist/actions/workflows/ci.yml>
) [![release](
  <https://github.com/eggplants/gh-starlist/actions/workflows/release.yml/badge.svg>
)](
  <https://github.com/eggplants/gh-starlist/actions/workflows/release.yml>
) 

A [GitHub CLI](https://cli.github.com) extension for [GitHub Star List](https://docs.github.com/en/get-started/exploring-projects-on-github/saving-repositories-with-stars#organizing-starred-repositories-with-lists)

_For information on the GitHub API we're using, see [QUERIES.md](QUERIES.md)._

## Installation

```bash
gh extension install eggplants/gh-starlist
gh auth status | grep -qF "'user'" || gh auth refresh -h github.com -s user
```

## Usage

```console
$ gh starlist list
NAME           SLUG           REPOS  VISIBILITY  DESCRIPTION
CLI/TUI        cli-tui        7      public
Python Utils   python-utils   9      public      Handy python libraries

$ gh starlist view cli-tui
REPOSITORY              STARS  LANGUAGE  DESCRIPTION
dandavison/delta        31830  Rust      A syntax-highlighting pager for git, diff, ...
tomnomnom/gf            2139   Go        A wrapper around grep, to help you grep for things

$ gh starlist create "Rust Tools" -d "Things written in Rust" --private
$ gh starlist add rust-tools BurntSushi/ripgrep sharkdp/fd
$ gh starlist remove rust-tools sharkdp/fd
$ gh starlist edit rust-tools --name "Rust CLI" --public
$ gh starlist delete rust-cli
```

<details><summary>Commands</summary>

| Command | Description |
| --- | --- |
| `gh starlist list` | List star lists |
| `gh starlist view <list>` | Show the repositories in a list |
| `gh starlist create <name>` | Create a list (`-d`, `--private`) |
| `gh starlist edit <list>` | Change name, description or visibility |
| `gh starlist delete <list>` | Delete a list (`--yes` to skip the prompt) |
| `gh starlist add <list> <repo>...` | Add repositories to a list |
| `gh starlist remove <list> <repo>...` | Remove repositories from a list |
| `gh starlist export` | Export every list as Markdown or JSON |

`list` and `view` take `--json` for scripting, and `--user` to read someone else's lists (their private lists are never visible):

```bash
gh starlist list --user zhuozhiyongde --json | jq -r '.[].slug'
gh starlist view rdf --json | jq -r '.repos[].nameWithOwner'
```

</details>

### Export as a Markdown

`export` renders every list, plus the starred repositories that are in no list, as a Markdown document with a table of contents:

```bash
gh starlist export -o STARS.md
gh starlist export --sort name --no-uncategorized
gh starlist export --format json | jq '.[] | {name, count: (.repos | length)}'
```

With `--template`, the generated Markdown replaces the `<!-- gh-starlist-export -->` placeholder of a template file, so the output can be dropped into an existing README:

```bash
gh starlist export --template template/template.md -o README.md
```

## License

[MIT](LICENSE)
