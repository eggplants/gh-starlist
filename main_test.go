package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// main exits the process, so it cannot be called in-process: these tests drive
// the real binary instead, which also covers the wiring gh relies on — the exit
// code, the stream each message lands on, and the version stamped at build time.

// cliPath is the binary built once for the whole package.
var cliPath string

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "gh-starlist-test")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer os.RemoveAll(dir)

	cliPath, err = build(dir, "")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

// build compiles the extension into dir and returns the binary path. ldflags is
// passed to the compiler when it is not empty.
func build(dir, ldflags string) (string, error) {
	name := "gh-starlist"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)

	args := []string{"build", "-o", path}
	if ldflags != "" {
		args = append(args, "-ldflags="+ldflags)
	}
	args = append(args, ".")

	command := exec.Command("go", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("go %s: %w\n%s", strings.Join(args, " "), err, output)
	}
	return path, nil
}

// result is one run of the binary.
type result struct {
	stdout   string
	stderr   string
	exitCode int
}

// runCLI runs binary with args and reports what it wrote and how it exited.
func runCLI(t *testing.T, binary string, args ...string) result {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, binary, args...)
	// The commands under test never reach the network, but a stray token in the
	// environment should not change what they print either.
	command.Env = append(os.Environ(), "NO_COLOR=1", "GH_FORCE_TTY=")

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	err := command.Run()
	if ctx.Err() != nil {
		t.Fatalf("%s %v timed out", binary, args)
	}

	code := 0
	var exitErr *exec.ExitError
	switch {
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	case err != nil:
		t.Fatalf("running %s %v: %v", binary, args, err)
	}
	return result{stdout: stdout.String(), stderr: stderr.String(), exitCode: code}
}

// run is runCLI against the binary built for this package.
func run(t *testing.T, args ...string) result {
	t.Helper()
	return runCLI(t, cliPath, args...)
}

func TestVersionDefaultsToDev(t *testing.T) {
	// An unstamped build has to say something, and the release workflow
	// replaces it.
	if version != "dev" {
		t.Errorf("version = %q, want dev", version)
	}
}

func TestVersionFlag(t *testing.T) {
	got := run(t, "--version")

	if got.exitCode != 0 {
		t.Errorf("exit code = %d, want 0", got.exitCode)
	}
	if want := "starlist version dev\n"; got.stdout != want {
		t.Errorf("stdout = %q, want %q", got.stdout, want)
	}
	if got.stderr != "" {
		t.Errorf("stderr = %q, want it empty", got.stderr)
	}
}

func TestVersionIsStampedAtBuildTime(t *testing.T) {
	// gh-extension-precompile builds the release with -X main.version=<tag>, so
	// the variable has to stay a package level string in package main. Making
	// it a constant, renaming it or moving it would silently ship "dev".
	stamped, err := build(t.TempDir(), "-X main.version=v1.2.3")
	if err != nil {
		t.Fatal(err)
	}

	got := runCLI(t, stamped, "--version")
	if want := "starlist version v1.2.3\n"; got.stdout != want {
		t.Errorf("stdout = %q, want %q", got.stdout, want)
	}
}

func TestHelpFlag(t *testing.T) {
	got := run(t, "--help")

	if got.exitCode != 0 {
		t.Errorf("exit code = %d, want 0", got.exitCode)
	}
	if got.stderr != "" {
		t.Errorf("stderr = %q, want it empty", got.stderr)
	}
	for _, want := range []string{
		"GitHub star list CLI.",
		"gh auth refresh -h github.com -s user",
		"Usage:",
		"add", "create", "delete", "edit", "export", "list", "remove", "view",
	} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("help should mention %q, got:\n%s", want, got.stdout)
		}
	}
}

func TestNoArgumentsPrintsHelp(t *testing.T) {
	// The root command has nothing to run on its own, so a bare invocation is
	// not an error.
	got := run(t)

	if got.exitCode != 0 {
		t.Errorf("exit code = %d, want 0", got.exitCode)
	}
	if !strings.Contains(got.stdout, "Usage:") {
		t.Errorf("stdout should hold the help, got:\n%s", got.stdout)
	}
}

func TestUnknownCommandFails(t *testing.T) {
	got := run(t, "bogus")

	if got.exitCode != 1 {
		t.Errorf("exit code = %d, want 1", got.exitCode)
	}
	if got.stdout != "" {
		t.Errorf("stdout = %q, want it empty", got.stdout)
	}
	if want := `gh-starlist: unknown command "bogus" for "starlist"`; !strings.Contains(got.stderr, want) {
		t.Errorf("stderr = %q, want %q", got.stderr, want)
	}
}

func TestErrorsArePrefixedAndSentToStderr(t *testing.T) {
	cases := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "a missing argument",
			args: []string{"view"},
			want: "gh-starlist: accepts 1 arg(s), received 0",
		},
		{
			name: "too many arguments",
			args: []string{"list", "extra"},
			want: "gh-starlist: unknown command",
		},
		{
			name: "an error raised by the command itself",
			args: []string{"edit", "my-list"},
			want: "gh-starlist: nothing to edit: pass --name, --description, --private or --public",
		},
		{
			name: "contradictory flags",
			args: []string{"edit", "my-list", "--private", "--public"},
			want: "gh-starlist: --private and --public are mutually exclusive",
		},
		{
			name: "an unknown flag",
			args: []string{"list", "--nope"},
			want: "gh-starlist: unknown flag: --nope",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := run(t, tc.args...)

			if got.exitCode != 1 {
				t.Errorf("exit code = %d, want 1", got.exitCode)
			}
			if got.stdout != "" {
				t.Errorf("stdout = %q, want errors on stderr only", got.stdout)
			}
			if !strings.Contains(got.stderr, tc.want) {
				t.Errorf("stderr = %q, want %q", got.stderr, tc.want)
			}
			if !strings.HasSuffix(got.stderr, "\n") {
				t.Errorf("stderr = %q, want it to end with a newline", got.stderr)
			}
		})
	}
}

func TestErrorsAreReportedOnce(t *testing.T) {
	// The root command silences cobra's own reporting, so the message must not
	// be printed a second time, and the usage block must stay out of the way.
	got := run(t, "edit", "my-list")

	if count := strings.Count(got.stderr, "nothing to edit"); count != 1 {
		t.Errorf("the error is reported %d times, want once:\n%s", count, got.stderr)
	}
	for _, unwanted := range []string{"Usage:", "Available Commands:", "Flags:"} {
		if strings.Contains(got.stderr, unwanted) {
			t.Errorf("stderr should not dump %q, got:\n%s", unwanted, got.stderr)
		}
	}
}

func TestSubcommandHelpExitsZero(t *testing.T) {
	// Every subcommand has to document itself without reaching for a token.
	for _, name := range []string{"list", "view", "create", "edit", "delete", "add", "remove", "export"} {
		t.Run(name, func(t *testing.T) {
			got := run(t, name, "--help")

			if got.exitCode != 0 {
				t.Errorf("exit code = %d, want 0", got.exitCode)
			}
			if got.stderr != "" {
				t.Errorf("stderr = %q, want it empty", got.stderr)
			}
			if !strings.Contains(got.stdout, "Usage:") {
				t.Errorf("stdout should hold the usage, got:\n%s", got.stdout)
			}
		})
	}
}

func TestAliases(t *testing.T) {
	// The README documents `ls` and `rm`; they have to survive on the binary.
	for alias, canonical := range map[string]string{"ls": "list", "rm": "delete"} {
		t.Run(alias, func(t *testing.T) {
			got := run(t, alias, "--help")

			if got.exitCode != 0 {
				t.Errorf("exit code = %d, want 0", got.exitCode)
			}
			if !strings.Contains(got.stdout, "starlist "+canonical) {
				t.Errorf("%s should be the alias of %s, got:\n%s", alias, canonical, got.stdout)
			}
		})
	}
}
