package api

import (
	"net/url"

	"github.com/maximseshuk/snapr/internal/config"
)

type JobListItem struct {
	Name          string      `json:"name"`
	Schedule      string      `json:"schedule"`
	SourcesCount  int         `json:"sourcesCount"`
	StoragesCount int         `json:"storagesCount"`
	Status        string      `json:"status"` // "idle" or "running"
	Active        bool        `json:"active"`
	LastRun       string      `json:"lastRun,omitempty"`
	NextRun       string      `json:"nextRun,omitempty"`
	LastResult    *LastResult `json:"lastResult,omitempty"`
}

type LastResult struct {
	Success  bool   `json:"success"`
	Duration string `json:"duration,omitempty"`
	Error    string `json:"error,omitempty"`
}

type JobDetail struct {
	Name            string            `json:"name"`
	Schedule        string            `json:"schedule"`
	Sources         []SourceDetail    `json:"sources,omitempty"`
	Storages        []StorageDetail   `json:"storages,omitempty"`
	DefaultStorage  string            `json:"defaultStorage,omitempty"`
	Compression     string            `json:"compression,omitempty"`
	Retention       *RetentionDetail  `json:"retention,omitempty"`
	HasBeforeScript bool              `json:"hasBeforeScript,omitempty"`
	HasAfterScript  bool              `json:"hasAfterScript,omitempty"`
	Encryption      *EncryptionDetail `json:"encryption,omitempty"`
	Notifiers       []NotifierDetail  `json:"notifiers,omitempty"`
	Split           *SplitDetail      `json:"split,omitempty"`
}

type SplitDetail struct {
	ChunkSize string `json:"chunkSize"`
}

type SourceDetail struct {
	Type          string            `json:"type"`
	Path          string            `json:"path,omitempty"`
	Excludes      []string          `json:"excludes,omitempty"`
	Host          string            `json:"host,omitempty"`
	Port          int               `json:"port,omitempty"`
	Username      string            `json:"username,omitempty"`
	Database      string            `json:"database,omitempty"`
	Tables        []string          `json:"tables,omitempty"`
	ExcludeTables []string          `json:"excludeTables,omitempty"`
	AllDatabases  bool              `json:"allDatabases,omitempty"`
	HasURI        bool              `json:"hasUri,omitempty"`
	Oplog         bool              `json:"oplog,omitempty"`
	Endpoint      string            `json:"endpoint,omitempty"`
	ZoneName      string            `json:"zoneName,omitempty"`
	SyncPath      string            `json:"syncPath,omitempty"`
	ExtraParams   map[string]string `json:"extraParams,omitempty"`
}

type StorageDetail struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	Path             string `json:"path,omitempty"`
	Bucket           string `json:"bucket,omitempty"`
	Region           string `json:"region,omitempty"`
	Endpoint         string `json:"endpoint,omitempty"`
	StorageClass     string `json:"storageClass,omitempty"`
	ZoneName         string `json:"zoneName,omitempty"`
	PullZoneHostname string `json:"pullZoneHostname,omitempty"`
	Host             string `json:"host,omitempty"`
	Port             int    `json:"port,omitempty"`
	Username         string `json:"username,omitempty"`
	HasPrivateKey    bool   `json:"hasPrivateKey,omitempty"`
	HasKnownHosts    bool   `json:"hasKnownHosts,omitempty"`
	StrictHostKey    *bool  `json:"strictHostKey,omitempty"`
	HasURL           bool   `json:"hasUrl,omitempty"`
	URLHost          string `json:"urlHost,omitempty"`
}

type RetentionDetail struct {
	Last int `json:"last"`
}

type EncryptionDetail struct {
	Type   string `json:"type,omitempty"`
	Cipher string `json:"cipher,omitempty"`
}

type NotifierDetail struct {
	Name      string   `json:"name,omitempty"`
	Type      string   `json:"type"`
	OnSuccess bool     `json:"onSuccess"`
	OnFailure bool     `json:"onFailure"`
	URLHost   string   `json:"urlHost,omitempty"`  // webhook host only
	ChatID    string   `json:"chatId,omitempty"`   // telegram
	From      string   `json:"from,omitempty"`     // email
	To        []string `json:"to,omitempty"`       // email
	SMTPHost  string   `json:"smtpHost,omitempty"` // email
}

// ToJobDetail returns the full job view without secrets. Callers must gate by ShowConfig.
func ToJobDetail(cfg *config.JobConfig) JobDetail {
	sources := make([]SourceDetail, len(cfg.Sources))
	for i, src := range cfg.Sources {
		sources[i] = SourceDetail{
			Type:          src.Type,
			Path:          src.Path,
			Excludes:      src.Excludes,
			Host:          src.Host,
			Port:          src.Port,
			Username:      src.Username,
			Database:      src.Database,
			Tables:        src.Tables,
			ExcludeTables: src.ExcludeTables,
			AllDatabases:  src.AllDatabases,
			HasURI:        src.URI != "",
			Oplog:         src.Oplog,
			Endpoint:      src.Endpoint,
			ZoneName:      src.ZoneName,
			SyncPath:      src.SyncPath,
			ExtraParams:   src.ExtraParams,
		}
	}

	storages := make([]StorageDetail, len(cfg.Storages))
	for i, stor := range cfg.Storages {
		storages[i] = StorageDetail{
			Type:             stor.Type,
			Name:             stor.Name,
			Path:             stor.Path,
			Bucket:           stor.Bucket,
			Region:           stor.Region,
			Endpoint:         stor.Endpoint,
			StorageClass:     stor.StorageClass,
			ZoneName:         stor.ZoneName,
			PullZoneHostname: stor.PullZoneHostname,
			Host:             stor.Host,
			Port:             stor.Port,
			Username:         stor.Username,
			HasPrivateKey:    stor.PrivateKey != "",
			HasKnownHosts:    stor.KnownHosts != "",
			StrictHostKey:    stor.StrictHostKey,
			HasURL:           stor.URL != "",
			URLHost:          urlHost(stor.URL),
		}
	}

	var enc *EncryptionDetail
	if cfg.Encryption != nil {
		enc = &EncryptionDetail{Type: cfg.Encryption.Type, Cipher: cfg.Encryption.Cipher}
	}

	var split *SplitDetail
	if cfg.Split != nil {
		split = &SplitDetail{ChunkSize: cfg.Split.ChunkSize}
	}

	notifiers := make([]NotifierDetail, len(cfg.Notifiers))
	for i, n := range cfg.Notifiers {
		nd := NotifierDetail{
			Name:      n.Name,
			Type:      n.Type,
			OnSuccess: n.OnSuccess,
			OnFailure: n.OnFailure,
		}
		switch n.Type {
		case "webhook":
			nd.URLHost = urlHost(n.URL)
		case "telegram":
			nd.ChatID = n.ChatID
		case "email":
			nd.From = n.From
			nd.To = n.To
			nd.SMTPHost = n.SMTPHost
		}
		notifiers[i] = nd
	}

	return JobDetail{
		Name:            cfg.Name,
		Schedule:        cfg.Schedule,
		Sources:         sources,
		Storages:        storages,
		DefaultStorage:  cfg.DefaultStorage,
		Compression:     cfg.Compression,
		Retention:       &RetentionDetail{Last: cfg.Retention.Last},
		HasBeforeScript: cfg.BeforeScript != "",
		HasAfterScript:  cfg.AfterScript != "",
		Encryption:      enc,
		Notifiers:       notifiers,
		Split:           split,
	}
}

func urlHost(s string) string {
	if s == "" {
		return ""
	}
	u, err := url.Parse(s)
	if err != nil {
		return ""
	}
	return u.Host
}
