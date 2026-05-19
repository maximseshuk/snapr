package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

//go:embed locales/*.json
var localesFS embed.FS

var (
	translations = make(map[string]map[string]string)
	mu           sync.RWMutex
	defaultLang  = "en"
)

func Init(defaultLanguage string) error {
	mu.Lock()
	defer mu.Unlock()

	if defaultLanguage != "" {
		defaultLang = defaultLanguage
	}

	entries, err := localesFS.ReadDir("locales")
	if err != nil {
		return fmt.Errorf("read locales dir: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") {
			continue
		}
		lang := strings.TrimSuffix(name, ".json")

		data, err := localesFS.ReadFile("locales/" + name)
		if err != nil {
			return fmt.Errorf("read locale %s: %w", name, err)
		}

		var msgs map[string]string
		if err := json.Unmarshal(data, &msgs); err != nil {
			return fmt.Errorf("parse locale %s: %w", name, err)
		}

		translations[lang] = msgs
	}

	return nil
}

func T(lang, key string) string {
	if lang == "" {
		lang = defaultLang
	}

	mu.RLock()
	defer mu.RUnlock()

	if msgs, ok := translations[lang]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}

	if msgs, ok := translations[defaultLang]; ok {
		if msg, ok := msgs[key]; ok {
			return msg
		}
	}

	return key
}

func DetectLanguage(acceptLanguage string) string {
	if acceptLanguage == "" {
		return defaultLang
	}

	langs := strings.Split(acceptLanguage, ",")
	for _, lang := range langs {
		lang = strings.TrimSpace(lang)
		if idx := strings.Index(lang, ";"); idx != -1 {
			lang = lang[:idx]
		}

		lang = strings.ToLower(lang)
		if strings.HasPrefix(lang, "ru") {
			return "ru"
		}
		if strings.HasPrefix(lang, "en") {
			return "en"
		}
	}

	return defaultLang
}
