package notifier

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/maximseshuk/snapr/internal/config"
)

func TestBuild_KnownTypes(t *testing.T) {
	cases := []struct {
		typ  string
		cfg  config.NotifierConfig
		want string
	}{
		{"webhook", config.NotifierConfig{Type: "webhook", URL: "https://example.com"}, "webhook"},
		{"telegram", config.NotifierConfig{Type: "telegram", BotToken: "t", ChatID: "1"}, "telegram"},
		{"email", config.NotifierConfig{Type: "email", SMTPHost: "h", SMTPPort: 25, From: "f@x", To: []string{"t@x"}}, "email"},
	}
	for _, c := range cases {
		t.Run(c.typ, func(t *testing.T) {
			n, err := build(c.cfg)
			require.NoError(t, err)
			assert.Equal(t, c.want, n.GetType())
		})
	}
}

func TestBuild_UnknownTypeErrors(t *testing.T) {
	_, err := build(config.NotifierConfig{Type: "carrier-pigeon"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown notifier type")
}

// fakeNotifier tracks Dispatch routing decisions.
type fakeNotifier struct {
	typ    string
	called []Event
	err    error
}

func (f *fakeNotifier) GetType() string { return f.typ }
func (f *fakeNotifier) Notify(_ context.Context, ev Event) error {
	f.called = append(f.called, ev)
	return f.err
}

func TestDispatcher_RoutesByOnSuccessOnFailure(t *testing.T) {
	successOnly := &fakeNotifier{typ: "x"}
	failureOnly := &fakeNotifier{typ: "y"}
	both := &fakeNotifier{typ: "z"}

	d := &Dispatcher{
		notifiers: []boundNotifier{
			{cfg: config.NotifierConfig{OnSuccess: true}, n: successOnly},
			{cfg: config.NotifierConfig{OnFailure: true}, n: failureOnly},
			{cfg: config.NotifierConfig{OnSuccess: true, OnFailure: true}, n: both},
		},
	}

	d.Dispatch(context.Background(), Event{JobName: "j", Success: true})
	assert.Len(t, successOnly.called, 1)
	assert.Empty(t, failureOnly.called)
	assert.Len(t, both.called, 1)

	d.Dispatch(context.Background(), Event{JobName: "j", Success: false})
	assert.Len(t, successOnly.called, 1, "successOnly must not fire on failure")
	assert.Len(t, failureOnly.called, 1)
	assert.Len(t, both.called, 2)
}

func TestDispatcher_SwallowsNotifyErrors(t *testing.T) {
	fail := &fakeNotifier{typ: "x", err: assert.AnError}
	d := &Dispatcher{notifiers: []boundNotifier{
		{cfg: config.NotifierConfig{OnFailure: true}, n: fail},
	}}
	// Must not panic / propagate.
	d.Dispatch(context.Background(), Event{Success: false})
	assert.Len(t, fail.called, 1)
}
