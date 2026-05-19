package api

import (
	"context"
	"errors"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/sse"
	"github.com/rs/zerolog"

	"github.com/maximseshuk/snapr/internal/backup"
	"github.com/maximseshuk/snapr/internal/config"
	"github.com/maximseshuk/snapr/internal/i18n"
	"github.com/maximseshuk/snapr/internal/logger"
	"github.com/maximseshuk/snapr/internal/storage"
	"github.com/maximseshuk/snapr/internal/utils"
)

var startTime = time.Now()

type BackupAPI struct {
	backupManager *backup.Manager
	sinks         *logger.FileSinks
	version       string
	logger        zerolog.Logger
}

func NewBackupAPI(backupManager *backup.Manager, sinks *logger.FileSinks, version string) *BackupAPI {
	if version == "" {
		version = "dev"
	}
	return &BackupAPI{
		backupManager: backupManager,
		sinks:         sinks,
		version:       version,
		logger:        logger.NewSystemLogger("api"),
	}
}

// RegisterOperations wires every endpoint onto the huma API. Auth is applied as
// a chi middleware on the underlying router, not here — huma sees the request
// only after the cookie has been validated.
func RegisterOperations(api huma.API, backupManager *backup.Manager, authMiddleware *AuthMiddleware, sinks *logger.FileSinks, version string) {
	a := NewBackupAPI(backupManager, sinks, version)

	api.OpenAPI().Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"session": {
			Type:        "apiKey",
			In:          "cookie",
			Name:        sessionCookieName,
			Description: "Session JWT issued by POST /auth/login. Public endpoints ignore it.",
		},
	}

	a.registerStatus(api)
	a.registerSettings(api)
	authMiddleware.registerAuth(api)
	a.registerJobs(api)
	a.registerBackups(api)
	a.registerLogs(api)
}

// /status & /settings

type StatusOutput struct {
	Body StatusResponse
}

type StatusResponse struct {
	Status      string `json:"status" example:"ok" doc:"Process health"`
	Version     string `json:"version" example:"1.0.0"`
	Uptime      int64  `json:"uptime" example:"1234" doc:"Uptime in seconds"`
	Environment string `json:"environment" example:"production"`
	JobsCount   int    `json:"jobsCount" example:"3"`
}

type SettingsOutput struct {
	Body SettingsResponse
}

type SettingsResponse struct {
	LogLimits   SettingsLogLimits    `json:"logLimits"`
	Logs        SettingsLogs         `json:"logs"`
	Permissions *SettingsPermissions `json:"permissions,omitempty"`
}

type SettingsLogLimits struct {
	JobLogs    int `json:"jobLogs"`
	SystemLogs int `json:"systemLogs"`
}

type SettingsLogs struct {
	System bool `json:"system"`
	PerJob bool `json:"perJob"`
}

type SettingsPermissions struct {
	AllowManualRun      bool `json:"allowManualRun"`
	AllowBackupDownload bool `json:"allowBackupDownload"`
	ShowConfig          bool `json:"showConfig"`
}

func (a *BackupAPI) registerStatus(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "getStatus",
		Method:      http.MethodGet,
		Path:        "/api/v1/status",
		Summary:     "Process status",
		Description: "Liveness check used by Docker HEALTHCHECK. Always public — no auth required.",
		Tags:        []string{"System"},
	}, func(_ context.Context, _ *struct{}) (*StatusOutput, error) {
		uptime := time.Since(startTime).Round(time.Second)
		return &StatusOutput{Body: StatusResponse{
			Status:      "ok",
			Version:     a.version,
			Uptime:      int64(uptime.Seconds()),
			Environment: utils.GetEnvironment(),
			JobsCount:   len(a.backupManager.ListJobNames()),
		}}, nil
	})
}

func (a *BackupAPI) registerSettings(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "getSettings",
		Method:      http.MethodGet,
		Path:        "/api/v1/settings",
		Summary:     "UI bootstrap settings",
		Description: "Log limits, file-sink toggles, permission set. Public — the UI reads it before login.",
		Tags:        []string{"System"},
	}, func(_ context.Context, _ *struct{}) (*SettingsOutput, error) {
		out := SettingsOutput{Body: SettingsResponse{
			LogLimits: SettingsLogLimits{
				JobLogs:    a.getJobLogLimit(),
				SystemLogs: a.getSystemLogLimit(),
			},
			Logs: SettingsLogs{
				System: a.sinks.SystemEnabled(),
				PerJob: a.sinks.PerJobEnabled(),
			},
		}}
		if p := a.permissions(); p != nil {
			out.Body.Permissions = &SettingsPermissions{
				AllowManualRun:      p.AllowManualRun,
				AllowBackupDownload: p.AllowBackupDownload,
				ShowConfig:          a.showConfig(),
			}
		}
		return &out, nil
	})
}

// /jobs

type ListJobsOutput struct {
	Body struct {
		Jobs []JobListItem `json:"jobs"`
	}
}

type JobNameInput struct {
	Name string `path:"name" example:"postgres-nightly" doc:"Job name as declared in snapr.yaml"`
}

type JobStatusOutput struct {
	Body JobStatusResponse
}

type JobStatusResponse struct {
	Name       string      `json:"name"`
	Status     string      `json:"status" enum:"idle,running"`
	Active     bool        `json:"active"`
	NextRun    string      `json:"nextRun,omitempty" doc:"RFC3339"`
	LastRun    string      `json:"lastRun,omitempty" doc:"RFC3339"`
	LastResult *LastResult `json:"lastResult,omitempty"`
}

type JobConfigOutput struct {
	Body struct {
		Job    string    `json:"job"`
		Config JobDetail `json:"config"`
	}
}

type RunJobOutput struct {
	Status int `header:"-"`
	Body   struct {
		Job       string `json:"job"`
		StartedAt string `json:"startedAt" doc:"RFC3339"`
	}
}

type CancelJobOutput struct {
	Body struct {
		Job         string `json:"job"`
		CancelledAt string `json:"cancelledAt" doc:"RFC3339"`
	}
}

func (a *BackupAPI) registerJobs(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "listJobs",
		Method:      http.MethodGet,
		Path:        "/api/v1/jobs",
		Summary:     "List jobs",
		Tags:        []string{"Jobs"},
		Security:    []map[string][]string{{"session": {}}},
	}, func(_ context.Context, _ *struct{}) (*ListJobsOutput, error) {
		jobConfigs := a.backupManager.ListJobs()
		jobs := make([]JobListItem, len(jobConfigs))
		for i, cfg := range jobConfigs {
			active := a.backupManager.IsJobActive(cfg.Name)
			status := "idle"
			if active {
				status = "running"
			}
			item := JobListItem{
				Name:          cfg.Name,
				Schedule:      cfg.Schedule,
				SourcesCount:  len(cfg.Sources),
				StoragesCount: len(cfg.Storages),
				Status:        status,
				Active:        active,
			}
			if nextRun, err := utils.GetNextRunTime(cfg.Schedule); err == nil {
				item.NextRun = nextRun.Format(time.RFC3339)
			}
			if jl, err := a.backupManager.GetJobLogInfo(cfg.Name); err == nil && jl != nil {
				item.LastRun = jl.GetStartTime().Format(time.RFC3339)
				if endTime := jl.GetEndTime(); endTime != nil {
					duration := endTime.Sub(jl.GetStartTime())
					item.LastResult = &LastResult{
						Success:  jl.GetStatus() == "success",
						Duration: duration.String(),
					}
					if errorMsg := jl.GetError(); errorMsg != "" {
						item.LastResult.Error = errorMsg
					}
				}
			}
			jobs[i] = item
		}
		out := &ListJobsOutput{}
		out.Body.Jobs = jobs
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getJobStatus",
		Method:      http.MethodGet,
		Path:        "/api/v1/jobs/{name}/status",
		Summary:     "Single-job status",
		Tags:        []string{"Jobs"},
		Security:    []map[string][]string{{"session": {}}},
	}, func(_ context.Context, in *JobNameInput) (*JobStatusOutput, error) {
		jobConfig, err := a.backupManager.GetJobConfig(in.Name)
		if err != nil || jobConfig == nil {
			return nil, huma.Error404NotFound("Job not found")
		}
		active := a.backupManager.IsJobActive(in.Name)
		status := "idle"
		if active {
			status = "running"
		}
		resp := JobStatusResponse{Name: in.Name, Status: status, Active: active}
		if nextRun, err := utils.GetNextRunTime(jobConfig.Schedule); err == nil {
			resp.NextRun = nextRun.Format(time.RFC3339)
		}
		if jl, _ := a.backupManager.GetJobLogInfo(in.Name); jl != nil {
			resp.LastRun = jl.GetStartTime().Format(time.RFC3339)
			if endTime := jl.GetEndTime(); endTime != nil {
				duration := endTime.Sub(jl.GetStartTime())
				resp.LastResult = &LastResult{
					Success:  jl.GetStatus() == "success",
					Duration: duration.String(),
				}
				if errMsg := jl.GetError(); errMsg != "" {
					resp.LastResult.Error = errMsg
				}
			}
		}
		return &JobStatusOutput{Body: resp}, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "getJobConfig",
		Method:      http.MethodGet,
		Path:        "/api/v1/jobs/{name}/config",
		Summary:     "Job config (secrets stripped)",
		Description: "When server.permissions.showConfig is false, only name and schedule are returned.",
		Tags:        []string{"Jobs"},
		Security:    []map[string][]string{{"session": {}}},
	}, func(_ context.Context, in *JobNameInput) (*JobConfigOutput, error) {
		cfg, err := a.backupManager.GetJobConfig(in.Name)
		if err != nil {
			return nil, huma.Error404NotFound("Job not found")
		}
		out := &JobConfigOutput{}
		out.Body.Job = in.Name
		if !a.showConfig() {
			out.Body.Config = JobDetail{Name: cfg.Name, Schedule: cfg.Schedule}
		} else {
			out.Body.Config = ToJobDetail(cfg)
		}
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID:   "runJob",
		Method:        http.MethodPost,
		Path:          "/api/v1/jobs/{name}/run",
		Summary:       "Trigger a job run",
		Description:   "Reserves a run slot and returns immediately. The backup runs in the background.",
		Tags:          []string{"Jobs"},
		Security:      []map[string][]string{{"session": {}}},
		DefaultStatus: http.StatusAccepted,
	}, func(ctx context.Context, in *JobNameInput) (*RunJobOutput, error) {
		lang := langFromContext(ctx)
		if p := a.permissions(); p == nil || !p.AllowManualRun {
			return nil, huma.Error403Forbidden(i18n.T(lang, "error.manual_run_disabled"))
		}
		jobNames := a.backupManager.ListJobNames()
		if !slices.Contains(jobNames, in.Name) {
			return nil, huma.Error404NotFound(i18n.T(lang, "error.job_not_found"))
		}
		if !a.backupManager.TryStartJob(in.Name) {
			return nil, huma.Error409Conflict(i18n.T(lang, "error.job_already_running"))
		}
		go func() {
			if err := a.backupManager.RunReservedJob(in.Name); err != nil {
				a.logger.Error().Err(err).Str("job", in.Name).Msg("Error executing job")
			}
		}()
		out := &RunJobOutput{}
		out.Body.Job = in.Name
		out.Body.StartedAt = time.Now().Format(time.RFC3339)
		return out, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "cancelJob",
		Method:      http.MethodPost,
		Path:        "/api/v1/jobs/{name}/cancel",
		Summary:     "Cancel an in-flight job",
		Tags:        []string{"Jobs"},
		Security:    []map[string][]string{{"session": {}}},
	}, func(ctx context.Context, in *JobNameInput) (*CancelJobOutput, error) {
		lang := langFromContext(ctx)
		if err := a.backupManager.CancelJob(in.Name); err != nil {
			return nil, huma.Error404NotFound(i18n.T(lang, "error.job_cancel_failed"))
		}
		out := &CancelJobOutput{}
		out.Body.Job = in.Name
		out.Body.CancelledAt = time.Now().Format(time.RFC3339)
		return out, nil
	})
}

// /jobs/{name}/backups

type ListBackupsOutput struct {
	Body struct {
		Job     string              `json:"job"`
		Backups []backup.BackupFile `json:"backups"`
	}
}

type DownloadBackupInput struct {
	Name     string `path:"name"`
	Filename string `path:"filename" doc:"Snapshot ID or part filename. Must not contain '/', '\\', or '..'."`
}

func (a *BackupAPI) registerBackups(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "listJobBackups",
		Method:      http.MethodGet,
		Path:        "/api/v1/jobs/{name}/backups",
		Summary:     "List snapshots for a job",
		Tags:        []string{"Backups"},
		Security:    []map[string][]string{{"session": {}}},
	}, func(ctx context.Context, in *JobNameInput) (*ListBackupsOutput, error) {
		lang := langFromContext(ctx)
		if !slices.Contains(a.backupManager.ListJobNames(), in.Name) {
			return nil, huma.Error404NotFound(i18n.T(lang, "error.job_not_found"))
		}
		backups, err := a.backupManager.GetJobBackups(ctx, in.Name)
		if err != nil {
			return nil, huma.Error500InternalServerError(i18n.T(lang, "error.backups_retrieval"))
		}
		out := &ListBackupsOutput{}
		out.Body.Job = in.Name
		out.Body.Backups = backups
		return out, nil
	})

	// Download streams octet-stream bodies of unknown size and may issue a 307
	// redirect to a signed URL, so it bypasses huma's auto-generated handler.
	huma.Register(api, huma.Operation{
		OperationID: "downloadBackup",
		Method:      http.MethodGet,
		Path:        "/api/v1/jobs/{name}/backups/{filename}/download",
		Summary:     "Download one snapshot or part",
		Description: "Returns the archive bytes (200) or a 307 redirect when the storage backend issues signed URLs (S3 signed mode, Bunny Pull Zone).",
		Tags:        []string{"Backups"},
		Security:    []map[string][]string{{"session": {}}},
		Responses: map[string]*huma.Response{
			"200": {Description: "Archive stream", Content: map[string]*huma.MediaType{
				"application/octet-stream": {Schema: &huma.Schema{Type: "string", Format: "binary"}},
			}},
			"307": {Description: "Redirect to a signed storage URL"},
			"400": {Description: "Filename contains '/', '\\', or '..'"},
			"403": {Description: "server.permissions.allowBackupDownload is false"},
			"404": {Description: "Job or snapshot not found"},
			"501": {Description: "Storage backend has no download support"},
		},
	}, a.downloadBackupHandler)
}

func (a *BackupAPI) downloadBackupHandler(ctx context.Context, in *DownloadBackupInput) (*huma.StreamResponse, error) {
	lang := langFromContext(ctx)
	if p := a.permissions(); p == nil || !p.AllowBackupDownload {
		return nil, huma.Error403Forbidden(i18n.T(lang, "error.download_disabled"))
	}
	if in.Filename == "" || in.Filename != filepath.Base(in.Filename) ||
		strings.ContainsAny(in.Filename, `/\`) || strings.Contains(in.Filename, "..") {
		return nil, huma.Error400BadRequest(i18n.T(lang, "error.backup_not_found"))
	}
	result, err := a.backupManager.OpenBackupDownload(ctx, in.Name, in.Filename)
	if err != nil {
		if errors.Is(err, storage.ErrDownloadNotSupported) {
			return nil, huma.Error501NotImplemented(i18n.T(lang, "error.backup_not_found"))
		}
		return nil, huma.Error404NotFound(i18n.T(lang, "error.backup_not_found"))
	}

	filename := in.Filename
	logger := a.logger

	return &huma.StreamResponse{
		Body: func(hctx huma.Context) {
			if result.RedirectURL != "" {
				hctx.SetHeader("Location", result.RedirectURL)
				hctx.SetStatus(http.StatusTemporaryRedirect)
				return
			}
			defer func() {
				_ = result.Body.Close()
			}()
			hctx.SetHeader("Content-Description", "File Transfer")
			hctx.SetHeader("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{"filename": filename}))
			hctx.SetHeader("Content-Type", "application/octet-stream")
			if result.Size > 0 {
				hctx.SetHeader("Content-Length", strconv.FormatInt(result.Size, 10))
			}
			hctx.SetStatus(http.StatusOK)
			if _, err := io.Copy(hctx.BodyWriter(), result.Body); err != nil {
				logger.Warn().Err(err).Str("filename", filename).Msg("Error streaming download")
			}
		},
	}, nil
}

// /logs (system + per-job)

type TailQueryInput struct {
	Tail int `query:"tail" minimum:"0" maximum:"50000" doc:"How many lines to return. Capped at 50000."`
}

type SystemLogsInput struct {
	TailQueryInput
}

type SystemLogsOutput struct {
	Body struct {
		Logs []string `json:"logs"`
	}
}

type JobLogsInput struct {
	JobNameInput
	TailQueryInput
}

type JobLogsOutput struct {
	Body JobLogsResponse
}

type JobLogsResponse struct {
	Job       string   `json:"job"`
	Logs      []string `json:"logs"`
	Status    string   `json:"status,omitempty"`
	StartTime string   `json:"startTime,omitempty"`
	EndTime   string   `json:"endTime,omitempty"`
	Duration  string   `json:"duration,omitempty"`
	Error     string   `json:"error,omitempty"`
}

// LogStreamMessage carries one rendered log line as an SSE `data:` frame.
type LogStreamMessage struct {
	Line string `json:"line"`
}

func (a *BackupAPI) registerLogs(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "getSystemLogs",
		Method:      http.MethodGet,
		Path:        "/api/v1/logs/system",
		Summary:     "Tail the system log",
		Tags:        []string{"Logs"},
		Security:    []map[string][]string{{"session": {}}},
	}, func(_ context.Context, in *SystemLogsInput) (*SystemLogsOutput, error) {
		path := a.systemLogPath()
		if path == "" {
			return nil, huma.Error503ServiceUnavailable("System log is disabled (logs.system=false)")
		}
		tail := in.Tail
		if tail == 0 {
			tail = a.getSystemLogLimit()
		}
		if tail > maxTailRequest {
			tail = maxTailRequest
		}
		lines, err := readLogTail(path, tail)
		if err != nil {
			a.logger.Error().Err(err).Str("path", path).Msg("Read log tail")
			return nil, huma.Error500InternalServerError("Failed to read log file")
		}
		out := &SystemLogsOutput{}
		out.Body.Logs = lines
		return out, nil
	})

	sse.Register(api, huma.Operation{
		OperationID: "streamSystemLogs",
		Method:      http.MethodGet,
		Path:        "/api/v1/logs/system/stream",
		Summary:     "Stream the system log (SSE)",
		Description: "Server-Sent Events. Heartbeat ': ping' frames every 15 seconds.",
		Tags:        []string{"Logs"},
		Security:    []map[string][]string{{"session": {}}},
	}, map[string]any{
		"message": LogStreamMessage{},
	}, func(ctx context.Context, in *SystemLogsInput, send sse.Sender) {
		path := a.systemLogPath()
		if path == "" {
			_ = send.Data(LogStreamMessage{Line: "System log is disabled (logs.system=false)"})
			return
		}
		a.streamLines(ctx, path, in.Tail, send)
	})

	huma.Register(api, huma.Operation{
		OperationID: "getJobLogs",
		Method:      http.MethodGet,
		Path:        "/api/v1/jobs/{name}/logs",
		Summary:     "Tail a job log plus run metadata",
		Tags:        []string{"Logs"},
		Security:    []map[string][]string{{"session": {}}},
	}, func(ctx context.Context, in *JobLogsInput) (*JobLogsOutput, error) {
		lang := langFromContext(ctx)
		if !slices.Contains(a.backupManager.ListJobNames(), in.Name) {
			return nil, huma.Error404NotFound(i18n.T(lang, "error.job_not_found"))
		}
		tail := in.Tail
		if tail == 0 {
			tail = a.getJobLogLimit()
		}
		if tail > maxTailRequest {
			tail = maxTailRequest
		}
		path := a.jobLogPath(in.Name)
		lines, err := readLogTail(path, tail)
		if err != nil {
			a.logger.Error().Err(err).Str("job", in.Name).Msg("Read job log tail")
			return nil, huma.Error500InternalServerError("Failed to read log file")
		}
		resp := JobLogsResponse{Job: in.Name, Logs: lines, Status: "idle"}
		if info := a.backupManager.GetJobRunInfo(in.Name); info != nil {
			startTime := info.GetStartTime()
			resp.Status = info.GetStatus()
			if !startTime.IsZero() {
				resp.StartTime = startTime.Format(time.RFC3339)
			}
			if endTime := info.GetEndTime(); endTime != nil {
				resp.EndTime = endTime.Format(time.RFC3339)
				resp.Duration = endTime.Sub(startTime).String()
			}
			if errMsg := info.GetError(); errMsg != "" {
				resp.Error = errMsg
			}
		}
		return &JobLogsOutput{Body: resp}, nil
	})

	sse.Register(api, huma.Operation{
		OperationID: "streamJobLogs",
		Method:      http.MethodGet,
		Path:        "/api/v1/jobs/{name}/logs/stream",
		Summary:     "Stream a per-job log (SSE)",
		Tags:        []string{"Logs"},
		Security:    []map[string][]string{{"session": {}}},
	}, map[string]any{
		"message": LogStreamMessage{},
	}, func(ctx context.Context, in *JobLogsInput, send sse.Sender) {
		lang := langFromContext(ctx)
		if !slices.Contains(a.backupManager.ListJobNames(), in.Name) {
			_ = send.Data(LogStreamMessage{Line: i18n.T(lang, "error.job_not_found")})
			return
		}
		path := a.jobLogPath(in.Name)
		if path == "" {
			_ = send.Data(LogStreamMessage{Line: "Per-job log is disabled (logs.perJob=false)"})
			return
		}
		a.streamLines(ctx, path, in.Tail, send)
	})
}

func (a *BackupAPI) streamLines(ctx context.Context, path string, initialTail int, send sse.Sender) {
	if initialTail > maxTailRequest {
		initialTail = maxTailRequest
	}
	if initialTail > 0 {
		raw, err := logger.TailFile(path, initialTail)
		if err != nil {
			a.logger.Warn().Err(err).Str("path", path).Msg("Read tail before stream")
		}
		for _, line := range raw {
			rendered := logger.RenderJSONToANSI(line)
			if rendered == "" {
				continue
			}
			if err := send.Data(LogStreamMessage{Line: rendered}); err != nil {
				return
			}
		}
	}

	stream := logger.LiveTail(ctx, path)
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-stream:
			if !ok {
				return
			}
			rendered := logger.RenderJSONToANSI(line)
			if rendered == "" {
				continue
			}
			if err := send.Data(LogStreamMessage{Line: rendered}); err != nil {
				return
			}
		case <-heartbeat.C:
			if err := send.Data(LogStreamMessage{Line: ""}); err != nil {
				return
			}
		}
	}
}

func (a *BackupAPI) permissions() *config.PermissionsConfig {
	return a.backupManager.Config().Server.Permissions
}

func (a *BackupAPI) logLimits() *config.LogLimitsConfig {
	return a.backupManager.Config().Server.LogLimits
}

func (a *BackupAPI) showConfig() bool {
	p := a.permissions()
	if p == nil {
		return true
	}
	return p.ShowConfig
}

// Fallbacks for nil/zero LogLimits. Keep in sync with viper.SetDefault in config.go.
const (
	defaultJobLogLimit    = 10000
	defaultSystemLogLimit = 10000
)

func (a *BackupAPI) getJobLogLimit() int {
	if l := a.logLimits(); l != nil && l.JobLogs > 0 {
		return l.JobLogs
	}
	return defaultJobLogLimit
}

func (a *BackupAPI) getSystemLogLimit() int {
	if l := a.logLimits(); l != nil && l.SystemLogs > 0 {
		return l.SystemLogs
	}
	return defaultSystemLogLimit
}

func langFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(langCtxKey).(string); ok && v != "" {
		return v
	}
	return "en"
}
