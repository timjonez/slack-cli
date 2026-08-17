package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/slack-go/slack"
	"github.com/timjonez/slack-cli/internal/config"
	"github.com/timjonez/slack-cli/internal/slackx"
)

type fakeClient struct {
	auth       slackx.Auth
	authErr    error
	posted     slackx.Posted
	postErr    error
	postCalls  int
	joinErr    error
	joined     string
	resolve    map[string]slackx.Channel
	list       []slackx.Channel
	events     []slackx.Event
	listenErr  error
	lastText   string
	lastThread string
}

func (f *fakeClient) AuthTest(ctx context.Context) (slackx.Auth, error) {
	return f.auth, f.authErr
}
func (f *fakeClient) PostMessage(ctx context.Context, channelID, text, threadTS string) (slackx.Posted, error) {
	f.postCalls++
	f.lastText = text
	f.lastThread = threadTS
	if f.postErr != nil && f.postCalls == 1 {
		return slackx.Posted{}, f.postErr
	}
	p := f.posted
	if p.Channel == "" {
		p.Channel = channelID
	}
	if p.Text == "" {
		p.Text = text
	}
	return p, nil
}
func (f *fakeClient) Join(ctx context.Context, channelID string) error {
	f.joined = channelID
	return f.joinErr
}
func (f *fakeClient) ResolveChannel(ctx context.Context, nameOrID string) (slackx.Channel, error) {
	if ch, ok := f.resolve[nameOrID]; ok {
		return ch, nil
	}
	id, name := slackx.ParseChannelRef(nameOrID)
	if id != "" {
		return slackx.Channel{ID: id, Name: name, IsMember: true}, nil
	}
	return slackx.Channel{ID: "CRESOLVED", Name: name, IsMember: true}, nil
}
func (f *fakeClient) ListJoined(ctx context.Context) ([]slackx.Channel, error) {
	return f.list, nil
}
func (f *fakeClient) UserName(ctx context.Context, userID string) (string, error) {
	return userID, nil
}
func (f *fakeClient) ChannelName(ctx context.Context, channelID string) (string, error) {
	return "#" + channelID, nil
}
func (f *fakeClient) Listen(ctx context.Context, filter slackx.Filter, status func(string), emit func(slackx.Event) error) error {
	if status != nil {
		status("connected")
	}
	for _, ev := range f.events {
		if err := emit(ev); err != nil {
			return err
		}
	}
	return f.listenErr
}

func testApp(f *fakeClient) (*App, *bytes.Buffer, *bytes.Buffer) {
	out, errb := &bytes.Buffer{}, &bytes.Buffer{}
	a := &App{
		Stdout: out,
		Stderr: errb,
		Stdin:  bytes.NewReader(nil),
		loadConfig: func() (config.Config, error) {
			return config.Config{BotToken: "xoxb-test", AppToken: "xapp-test"}, nil
		},
		newClient: func(cfg config.Config) (slackx.Client, error) {
			return f, nil
		},
		stdinIsPipe: func() bool { return false },
	}
	return a, out, errb
}

func TestWhoami(t *testing.T) {
	f := &fakeClient{auth: slackx.Auth{Team: "Acme", TeamID: "T1", User: "bot", UserID: "U1", BotID: "B1", URL: "https://acme.slack.com/"}}
	a, out, _ := testApp(f)
	if code := a.Execute([]string{"whoami"}); code != 0 {
		t.Fatalf("exit %d: %s", code, out.String())
	}
	got := out.String()
	if !strings.Contains(got, "Acme") || !strings.Contains(got, "U1") {
		t.Fatalf("got %q", got)
	}
}

func TestSendArgsAndThread(t *testing.T) {
	f := &fakeClient{
		posted:  slackx.Posted{Channel: "C1", TS: "1.0", Text: "hello world"},
		resolve: map[string]slackx.Channel{"#eng": {ID: "C1", Name: "eng", IsMember: true}},
	}
	a, out, _ := testApp(f)
	if code := a.Execute([]string{"send", "#eng", "hello", "world", "--thread", "9.0"}); code != 0 {
		t.Fatalf("exit %d stderr=%s", code, out.String())
	}
	if f.lastText != "hello world" || f.lastThread != "9.0" {
		t.Fatalf("text=%q thread=%q", f.lastText, f.lastThread)
	}
	if !strings.Contains(out.String(), "1.0") {
		t.Fatalf("stdout %q", out.String())
	}
}

func TestSendStdin(t *testing.T) {
	f := &fakeClient{posted: slackx.Posted{Channel: "C1", TS: "2.0"}}
	a, _, errb := testApp(f)
	a.Stdin = strings.NewReader("from pipe\n")
	a.stdinIsPipe = func() bool { return true }
	if code := a.Execute([]string{"send", "#eng"}); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if f.lastText != "from pipe" {
		t.Fatalf("text=%q", f.lastText)
	}
}

func TestSendMissingText(t *testing.T) {
	a, _, errb := testApp(&fakeClient{})
	if code := a.Execute([]string{"send", "#eng"}); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(errb.String(), "message text required") {
		t.Fatalf("stderr %q", errb.String())
	}
}

func TestSendAutoJoin(t *testing.T) {
	f := &fakeClient{
		postErr: slack.SlackErrorResponse{Err: "not_in_channel"},
		posted:  slackx.Posted{Channel: "C1", TS: "3.0"},
		resolve: map[string]slackx.Channel{"#eng": {ID: "C1", Name: "eng"}},
	}
	a, _, errb := testApp(f)
	if code := a.Execute([]string{"send", "#eng", "hi"}); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if f.joined != "C1" {
		t.Fatalf("joined %q", f.joined)
	}
	if f.postCalls != 2 {
		t.Fatalf("post calls %d", f.postCalls)
	}
}

func TestSendJSON(t *testing.T) {
	f := &fakeClient{posted: slackx.Posted{Channel: "C1", TS: "4.0", Text: "hi"}}
	a, out, _ := testApp(f)
	if code := a.Execute([]string{"--json", "send", "#eng", "hi"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	var m map[string]string
	if err := json.Unmarshal(out.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["ts"] != "4.0" || m["channel"] != "C1" {
		t.Fatalf("%v", m)
	}
}

func TestListenJSONL(t *testing.T) {
	f := &fakeClient{
		auth: slackx.Auth{UserID: "UBOT", BotID: "BBOT"},
		events: []slackx.Event{{
			Type: "message", Channel: "C1", ChannelName: "#eng",
			User: "U1", UserName: "alice", TS: "5.0", Text: "hi",
		}},
		resolve: map[string]slackx.Channel{"#eng": {ID: "C1", Name: "eng", IsMember: true}},
	}
	a, out, errb := testApp(f)
	if code := a.Execute([]string{"--json", "listen", "#eng"}); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	var ev slackx.Event
	if err := json.Unmarshal(out.Bytes(), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Text != "hi" || ev.Channel != "C1" {
		t.Fatalf("%+v", ev)
	}
	if !strings.Contains(errb.String(), "connected") {
		t.Fatalf("status on stderr: %q", errb.String())
	}
}

func TestListenNotMemberWarns(t *testing.T) {
	f := &fakeClient{
		auth:    slackx.Auth{UserID: "UBOT"},
		resolve: map[string]slackx.Channel{"#eng": {ID: "C1", Name: "eng", IsMember: false}},
	}
	a, _, errb := testApp(f)
	if code := a.Execute([]string{"listen", "#eng"}); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if !strings.Contains(errb.String(), "not in #eng") {
		t.Fatalf("stderr %q", errb.String())
	}
}

func TestListenDMSkipsMemberWarning(t *testing.T) {
	f := &fakeClient{
		auth: slackx.Auth{UserID: "UBOT"},
		resolve: map[string]slackx.Channel{
			"D0BQJRLC11U": {ID: "D0BQJRLC11U", IsIM: true, IsMember: false},
		},
	}
	a, _, errb := testApp(f)
	if code := a.Execute([]string{"listen", "D0BQJRLC11U"}); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	if strings.Contains(errb.String(), "not in") {
		t.Fatalf("should not warn for DMs: %q", errb.String())
	}
}

func TestChannels(t *testing.T) {
	f := &fakeClient{list: []slackx.Channel{
		{ID: "C1", Name: "eng", IsMember: true},
		{ID: "D1", Name: "alice", IsIM: true, IsMember: true},
	}}
	a, out, _ := testApp(f)
	if code := a.Execute([]string{"channels"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	got := out.String()
	if !strings.Contains(got, "#eng") || !strings.Contains(got, "dm:alice") {
		t.Fatalf("%q", got)
	}
}

func TestJoin(t *testing.T) {
	f := &fakeClient{resolve: map[string]slackx.Channel{"#eng": {ID: "C1", Name: "eng"}}}
	a, out, _ := testApp(f)
	if code := a.Execute([]string{"join", "#eng"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if f.joined != "C1" {
		t.Fatalf("joined %q", f.joined)
	}
	if !strings.Contains(out.String(), "joined #eng") {
		t.Fatalf("%q", out.String())
	}
}

func TestMissingBotToken(t *testing.T) {
	out, errb := &bytes.Buffer{}, &bytes.Buffer{}
	a := &App{
		Stdout:     out,
		Stderr:     errb,
		loadConfig: func() (config.Config, error) { return config.Config{}, nil },
	}
	if code := a.Execute([]string{"whoami"}); code != 1 {
		t.Fatalf("exit %d", code)
	}
	if !strings.Contains(errb.String(), "bot token required") {
		t.Fatalf("%q", errb.String())
	}
}

func TestJSONError(t *testing.T) {
	out, errb := &bytes.Buffer{}, &bytes.Buffer{}
	a := &App{
		Stdout: out,
		Stderr: errb,
		loadConfig: func() (config.Config, error) {
			return config.Config{BotToken: "xoxb-x"}, nil
		},
		newClient: func(cfg config.Config) (slackx.Client, error) {
			return &fakeClient{
				resolve: map[string]slackx.Channel{},
			}, nil
		},
	}
	// force not found
	a.newClient = func(cfg config.Config) (slackx.Client, error) {
		return &notFoundClient{}, nil
	}
	if code := a.Execute([]string{"--json", "send", "#missing", "hi"}); code != 1 {
		t.Fatalf("exit %d", code)
	}
	var je jsonError
	if err := json.Unmarshal(errb.Bytes(), &je); err != nil {
		t.Fatal(err)
	}
	if je.Code != "not_found" {
		t.Fatalf("%+v", je)
	}
}

type notFoundClient struct{ fakeClient }

func (n *notFoundClient) ResolveChannel(ctx context.Context, nameOrID string) (slackx.Channel, error) {
	return slackx.Channel{}, errors.Join(slackx.ErrNotFound, errors.New("channel missing"))
}

func TestAuthSetAndPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	t.Setenv("SLACKCLI_CONFIG", path)
	a, out, errb := testApp(&fakeClient{})
	if code := a.Execute([]string{"auth", "set", "--bot-token", "xoxb-abc", "--app-token", "xapp-def"}); code != 0 {
		t.Fatalf("exit %d: %s", code, errb.String())
	}
	cfg, err := config.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.BotToken != "xoxb-abc" || cfg.AppToken != "xapp-def" {
		t.Fatalf("%+v", cfg)
	}
	st, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("perm %o", st.Mode().Perm())
	}

	out.Reset()
	if code := a.Execute([]string{"auth", "path"}); code != 0 {
		t.Fatalf("path exit %d", code)
	}
	if strings.TrimSpace(out.String()) != path {
		t.Fatalf("path %q", out.String())
	}
}

func TestVersion(t *testing.T) {
	a, out, _ := testApp(&fakeClient{})
	if code := a.Execute([]string{"version"}); code != 0 {
		t.Fatalf("exit %d", code)
	}
	if strings.TrimSpace(out.String()) != Version {
		t.Fatalf("%q", out.String())
	}
}
