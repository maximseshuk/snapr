package i18n

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var initOnce sync.Once

func mustInit(t *testing.T) {
	t.Helper()
	initOnce.Do(func() {
		require.NoError(t, Init("en"))
	})
}

func TestTKnownKey(t *testing.T) {
	mustInit(t)

	assert.Equal(t, "Job not found", T("en", "error.job_not_found"))
	assert.Equal(t, "Задача не найдена", T("ru", "error.job_not_found"))
}

func TestTMissingKeyReturnsKey(t *testing.T) {
	mustInit(t)

	assert.Equal(t, "error.does_not_exist", T("en", "error.does_not_exist"))
}

func TestTUnknownLanguageFallsBackToDefault(t *testing.T) {
	mustInit(t)

	assert.Equal(t, T("en", "error.job_not_found"), T("xx", "error.job_not_found"))
}

func TestTEmptyLanguage(t *testing.T) {
	mustInit(t)

	assert.Equal(t, T("en", "error.unauthorized"), T("", "error.unauthorized"))
}

func TestDetectLanguage(t *testing.T) {
	mustInit(t)

	cases := []struct{ accept, want string }{
		{"", "en"},
		{"en", "en"},
		{"en-US,en;q=0.9", "en"},
		{"ru-RU,ru;q=0.9,en;q=0.8", "ru"},
		{"ru", "ru"},
		{"de", "en"},
		{"  ru-RU  ", "ru"},
	}
	for _, c := range cases {
		assert.Equalf(t, c.want, DetectLanguage(c.accept), "DetectLanguage(%q)", c.accept)
	}
}
