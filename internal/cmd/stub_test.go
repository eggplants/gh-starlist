package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
	"github.com/eggplants/gh-starlist/internal/starlist"
)

// stubTransport answers each GraphQL request with the next canned body and
// records what was sent, so tests can assert on the requests a command makes.
type stubTransport struct {
	mutex  sync.Mutex
	bodies []string
	byList map[string][]string
	calls  int
	sent   []gqlRequest
}

type gqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

func (s *stubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var recorded gqlRequest
	if req.Body != nil {
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(payload, &recorded); err != nil {
			return nil, err
		}
	}

	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.sent = append(s.sent, recorded)
	body, err := s.answer(recorded)
	if err != nil {
		return nil, err
	}
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Request:    req,
	}, nil
}

// stubGitHub points the commands at a client answering with bodies, in order,
// for the rest of the test. A listItems body is answered whenever the items of
// that star list are asked for, whatever the order.
func stubGitHub(t *testing.T, bodies ...interface{}) *stubTransport {
	t.Helper()
	ordered, byList := split(bodies)
	transport := &stubTransport{bodies: ordered, byList: byList}
	client, err := starlist.NewClientWithOptions(api.ClientOptions{
		Host:      "github.com",
		AuthToken: "test",
		Transport: transport,
	})
	if err != nil {
		t.Fatal(err)
	}

	original := newClient
	newClient = func() (*starlist.Client, error) { return client, nil }
	t.Cleanup(func() { newClient = original })
	return transport
}

// output is what a command wrote to the two standard streams.
type output struct {
	stdout string
	stderr string
}

// run executes the command tree with args, capturing the real os.Stdout and
// os.Stderr the commands write to directly.
func run(t *testing.T, args ...string) (output, error) {
	t.Helper()
	// A pipe is not a terminal, so the tables render as tab separated values.
	t.Setenv("GH_FORCE_TTY", "")

	outReader, outWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errReader, errWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}

	originalOut, originalErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outWriter, errWriter
	defer func() { os.Stdout, os.Stderr = originalOut, originalErr }()

	// Both pipes are drained concurrently so neither can fill and block the
	// command mid-write.
	stdoutText := make(chan string, 1)
	stderrText := make(chan string, 1)
	go func() {
		text, _ := io.ReadAll(outReader)
		stdoutText <- string(text)
	}()
	go func() {
		text, _ := io.ReadAll(errReader)
		stderrText <- string(text)
	}()

	root := NewRoot("test")
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs(args)
	runErr := root.Execute()

	if err := outWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := errWriter.Close(); err != nil {
		t.Fatal(err)
	}
	return output{stdout: <-stdoutText, stderr: <-stderrText}, runErr
}

// decodeJSON parses a command's JSON output into value.
func decodeJSON(t *testing.T, payload string, value interface{}) {
	t.Helper()
	if err := json.Unmarshal([]byte(payload), value); err != nil {
		t.Fatalf("output is not valid JSON (%v):\n%s", err, payload)
	}
}

// listItems binds a canned response to the star list it answers for. Star
// lists are read in parallel, so a response that belongs to one of them cannot
// be matched by arrival order.
type listItems struct {
	id   string
	body string
}

// answer picks the response for a request: the queue of the star list it asks
// about when the test bound one, the next positional body otherwise. The
// caller holds the mutex.
func (s *stubTransport) answer(request gqlRequest) (string, error) {
	if id, ok := request.Variables["listId"].(string); ok && len(s.byList) > 0 {
		queue := s.byList[id]
		if len(queue) == 0 {
			return "", fmt.Errorf("no canned response left for list %s", id)
		}
		s.byList[id] = queue[1:]
		return queue[0], nil
	}
	if s.calls >= len(s.bodies) {
		return "", errors.New("unexpected extra request")
	}
	body := s.bodies[s.calls]
	s.calls++
	return body, nil
}

// split sorts canned responses into the positional queue and the per-list
// ones, so a test can mix both.
func split(bodies []interface{}) ([]string, map[string][]string) {
	ordered := make([]string, 0, len(bodies))
	byList := map[string][]string{}
	for _, body := range bodies {
		switch body := body.(type) {
		case listItems:
			byList[body.id] = append(byList[body.id], body.body)
		case string:
			ordered = append(ordered, body)
		default:
			panic(fmt.Sprintf("unsupported canned response %T", body))
		}
	}
	return ordered, byList
}
