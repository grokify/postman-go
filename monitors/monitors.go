// Package monitors provides a high-level client for Postman's Monitors API.
//
// Monitors run a collection on a schedule (optionally from multiple regions)
// and report on the results, so teams can be alerted when an API starts
// failing or degrading.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	monitors, _ := client.Monitors().List(ctx, &monitors.ListInput{WorkspaceID: wsID})
package monitors

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-faster/jx"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Region is a region a monitor can run from.
type Region string

// Region values.
const (
	RegionUsEast         Region = "us-east"
	RegionUsWest         Region = "us-west"
	RegionApSoutheast    Region = "ap-southeast"
	RegionCaCentral      Region = "ca-central"
	RegionEuCentral      Region = "eu-central"
	RegionSaEast         Region = "sa-east"
	RegionUk             Region = "uk"
	RegionUsEastStaticip Region = "us-east-staticip"
	RegionUsWestStaticip Region = "us-west-staticip"
)

// Service is the high-level Monitors client. Obtain one via
// postman.Client.Monitors.
type Service struct {
	api *api.Client
}

// New creates a Monitors service over the given generated API client. Most
// callers should use postman.Client.Monitors instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// Schedule controls when and in which timezone a monitor runs.
type Schedule struct {
	// Cron is a cron expression controlling how often the monitor runs.
	Cron string
	// Timezone is the IANA timezone the schedule is evaluated in.
	Timezone string
	// NextRun is the time of the monitor's next scheduled run. Read-only:
	// populated by Get, ignored by Create and Update.
	NextRun string
}

// RetrySettings configures how many times a monitor retries a failed run.
type RetrySettings struct {
	// Attempts is the number of retry attempts. Zero is treated as unset.
	Attempts int
}

// Options are monitor run options.
type Options struct {
	// FollowRedirects, when true, follows HTTP redirects during the run.
	FollowRedirects bool
	// StrictSsl, when true, fails requests with invalid SSL certificates.
	StrictSsl bool
	// RequestDelay is the delay, in milliseconds, between requests. Zero is
	// treated as unset.
	RequestDelay int
	// RequestTimeout is the per-request timeout, in milliseconds. Zero is
	// treated as unset.
	RequestTimeout int
}

// NotificationEmail is a single email-based notification target.
type NotificationEmail struct {
	Email string
}

// Notifications configures who is emailed when a monitor errors or fails.
type Notifications struct {
	OnError   []NotificationEmail
	OnFailure []NotificationEmail
}

// RunStats summarizes assertion, request, and error counts for a monitor run.
type RunStats struct {
	AssertionsTotal  int
	AssertionsFailed int
	RequestsTotal    int
	RequestsFailed   int
	RunCount         int
	ErrorCount       int
	AbortedCount     int
	ResponseLatency  int
	ResponseSize     int
}

// LastRun summarizes the most recent execution of a monitor.
type LastRun struct {
	Status     string
	StartedAt  string
	FinishedAt string
	Stats      RunStats
}

// Monitor is a summary of a monitor, as returned by List.
type Monitor struct {
	ID             string
	Name           string
	Active         bool
	UID            string
	Owner          int
	CollectionUID  string
	EnvironmentUID string
}

// MonitorDetail is the full detail of a monitor, as returned by Get.
type MonitorDetail struct {
	ID    string
	Name  string
	UID   string
	Owner int
	// Active reports whether the monitor is scheduled to run.
	Active bool
	// NotificationLimit caps how many notification emails Postman sends per
	// day. Zero means unset.
	NotificationLimit int
	CollectionUID     string
	EnvironmentUID    string
	JobID             string
	Options           Options
	Notifications     Notifications
	Distribution      []Region
	Schedule          Schedule
	Retry             RetrySettings
	LastRun           LastRun
}

// MonitorRef is minimal monitor info returned by Create and Update.
type MonitorRef struct {
	ID     string
	Name   string
	Active bool
	UID    string
}

// --- List --------------------------------------------------------------

// ListInput holds the filters and pagination options for List.
type ListInput struct {
	// WorkspaceID limits results to the given workspace.
	WorkspaceID string
	// Active, when true, returns only active monitors.
	Active bool
	// Owner limits results to monitors owned by the given user ID.
	Owner int
	// CollectionUID limits results to monitors that run the given collection.
	CollectionUID string
	// EnvironmentUID limits results to monitors that use the given environment.
	EnvironmentUID string
	// Cursor is the pagination cursor (use ListResult.NextCursor to page).
	Cursor string
	// Limit is the maximum number of rows to return, up to 25.
	Limit int
}

// ListResult is the paginated set of monitors.
type ListResult struct {
	Monitors   []Monitor
	Limit      int
	NextCursor string
}

// List returns the monitors visible to the caller.
func (s *Service) List(ctx context.Context, in *ListInput) (*ListResult, error) {
	if in == nil {
		in = &ListInput{}
	}
	params := api.GetMonitorsParams{}
	if in.WorkspaceID != "" {
		params.Workspace = api.NewOptString(in.WorkspaceID)
	}
	if in.Active {
		params.Active = api.NewOptBool(true)
	}
	if in.Owner > 0 {
		params.Owner = api.NewOptInt(in.Owner)
	}
	if in.CollectionUID != "" {
		params.CollectionUid = api.NewOptString(in.CollectionUID)
	}
	if in.EnvironmentUID != "" {
		params.EnvironmentUid = api.NewOptString(in.EnvironmentUID)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}

	res, err := s.api.GetMonitors(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetMonitorsOkResponse:
		out := &ListResult{}
		for _, m := range r.Monitors {
			out.Monitors = append(out.Monitors, Monitor{
				ID:             m.ID.Or(""),
				Name:           m.Name.Or(""),
				Active:         m.Active.Or(false),
				UID:            m.UID.Or(""),
				Owner:          m.Owner.Or(0),
				CollectionUID:  m.CollectionUid.Or(""),
				EnvironmentUID: m.EnvironmentUid.Or(""),
			})
		}
		if meta, ok := r.Meta.Get(); ok {
			out.Limit = meta.Limit.Or(0)
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.Monitors400Errors:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetAuditLogEventActionsClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Create --------------------------------------------------------------

// CreateInput holds the fields for creating a monitor.
//
// **Note:** you cannot create monitors for collections added to an API
// definition. If WorkspaceID is empty, Postman creates the monitor in the
// oldest personal Internal workspace you own.
type CreateInput struct {
	// WorkspaceID is the workspace to create the monitor in.
	WorkspaceID string

	// Name is the monitor's name. Required.
	Name string
	// Active activates the monitor immediately after it is created.
	Active bool
	// NotificationLimit caps how many notification emails Postman sends per
	// day. Zero means unset.
	NotificationLimit int
	// Collection is the ID of the collection the monitor runs. Required.
	Collection string
	// Environment is the ID of the environment the monitor's requests run
	// against.
	Environment string
	// Retry configures automatic retries of failed runs.
	Retry *RetrySettings
	// Options configures monitor run behavior.
	Options *Options
	// Schedule controls when the monitor runs. Required.
	Schedule Schedule
	// Distribution runs the monitor from multiple regions.
	Distribution []Region
	// Notifications configures who is emailed on errors and failures.
	Notifications *Notifications
}

// Create creates a monitor.
func (s *Service) Create(ctx context.Context, in *CreateInput) (*MonitorRef, error) {
	if in == nil {
		in = &CreateInput{}
	}
	monitor := api.CreateMonitorMonitor{
		Name:              in.Name,
		NotificationLimit: intToRaw(in.NotificationLimit),
		Collection:        in.Collection,
		Schedule: api.MonitorSchedule{
			Cron:     api.NewOptString(in.Schedule.Cron),
			Timezone: api.NewOptString(in.Schedule.Timezone),
		},
	}
	if in.Active {
		monitor.Active = api.NewOptBool(true)
	}
	if in.Environment != "" {
		monitor.Environment = api.NewOptString(in.Environment)
	}
	if in.Retry != nil {
		monitor.Retry = api.NewOptMonitorRetrySettings(api.MonitorRetrySettings{
			Attempts: intToRaw(in.Retry.Attempts),
		})
	}
	if in.Options != nil {
		monitor.Options = api.NewOptMonitorOptions(optionsToAPI(*in.Options))
	}
	for _, r := range in.Distribution {
		monitor.Distribution = append(monitor.Distribution, api.MonitorDistribution{
			Region: api.NewOptRegion(api.Region(r)),
		})
	}
	if in.Notifications != nil {
		monitor.Notifications = api.NewOptMonitorNotifications(notificationsToAPI(*in.Notifications))
	}

	req := &api.CreateMonitor{Monitor: api.NewOptCreateMonitorMonitor(monitor)}
	params := api.CreateMonitorParams{Workspace: in.WorkspaceID}

	res, err := s.api.CreateMonitor(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateUpdateMonitorResponse:
		return monitorRefFromAPI(r), nil
	case *api.Monitors400Errors:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func monitorRefFromAPI(r *api.CreateUpdateMonitorResponse) *MonitorRef {
	out := &MonitorRef{}
	if m, ok := r.Monitor.Get(); ok {
		out.ID = m.ID.Or("")
		out.Name = m.Name.Or("")
		out.Active = m.Active.Or(false)
		out.UID = m.UID.Or("")
	}
	return out
}

// --- Get --------------------------------------------------------------

// Get returns information about a monitor.
func (s *Service) Get(ctx context.Context, monitorID string) (*MonitorDetail, error) {
	params := api.GetMonitorParams{MonitorId: monitorID}

	res, err := s.api.GetMonitor(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetMonitorOkResponse:
		out := &MonitorDetail{}
		if m, ok := r.Monitor.Get(); ok {
			out.ID = m.ID.Or("")
			out.Name = m.Name.Or("")
			out.UID = m.UID.Or("")
			out.Owner = m.Owner.Or(0)
			out.Active = m.Active.Or(false)
			out.NotificationLimit = rawToInt(m.NotificationLimit)
			out.CollectionUID = m.CollectionUid.Or("")
			out.EnvironmentUID = m.EnvironmentUid.Or("")
			out.JobID = m.JobId.Or("")
			if opts, ok := m.Options.Get(); ok {
				out.Options = optionsFromAPI(opts)
			}
			if n, ok := m.Notifications.Get(); ok {
				out.Notifications = notificationsFromAPI(n)
			}
			for _, d := range m.Distribution {
				out.Distribution = append(out.Distribution, Region(d.Region.Or("")))
			}
			if sched, ok := m.Schedule.Get(); ok {
				out.Schedule = Schedule{
					Cron:     sched.Cron.Or(""),
					Timezone: sched.Timezone.Or(""),
					NextRun:  sched.NextRun.Or(""),
				}
			}
			if retry, ok := m.Retry.Get(); ok {
				out.Retry = RetrySettings{Attempts: rawToInt(retry.Attempts)}
			}
			if lastRun, ok := m.LastRun.Get(); ok {
				out.LastRun = LastRun{
					Status:     lastRun.Status.Or(""),
					StartedAt:  lastRun.StartedAt.Or(""),
					FinishedAt: lastRun.FinishedAt.Or(""),
				}
				if stats, ok := lastRun.Stats.Get(); ok {
					out.LastRun.Stats = runStatsFromAPI(stats)
				}
			}
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetAuditLogEventActionsClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

func runStatsFromAPI(stats api.MonitorRunStats) RunStats {
	out := RunStats{
		RunCount:        stats.RunCount.Or(0),
		ErrorCount:      stats.ErrorCount.Or(0),
		AbortedCount:    stats.AbortedCount.Or(0),
		ResponseLatency: stats.ResponseLatency.Or(0),
		ResponseSize:    stats.ResponseSize.Or(0),
	}
	if a, ok := stats.Assertions.Get(); ok {
		out.AssertionsTotal = a.Total.Or(0)
		out.AssertionsFailed = a.Failed.Or(0)
	}
	if r, ok := stats.Requests.Get(); ok {
		out.RequestsTotal = r.Total.Or(0)
		out.RequestsFailed = r.Failed.Or(0)
	}
	return out
}

func optionsFromAPI(o api.MonitorOptions) Options {
	return Options{
		FollowRedirects: o.FollowRedirects.Or(false),
		StrictSsl:       o.StrictSsl.Or(false),
		RequestDelay:    rawToInt(o.RequestDelay),
		RequestTimeout:  rawToInt(o.RequestTimeout),
	}
}

func optionsToAPI(o Options) api.MonitorOptions {
	out := api.MonitorOptions{
		RequestDelay:   intToRaw(o.RequestDelay),
		RequestTimeout: intToRaw(o.RequestTimeout),
	}
	if o.FollowRedirects {
		out.FollowRedirects = api.NewOptBool(true)
	}
	if o.StrictSsl {
		out.StrictSsl = api.NewOptBool(true)
	}
	return out
}

func notificationsFromAPI(n api.MonitorNotifications) Notifications {
	out := Notifications{}
	for _, e := range n.OnError {
		out.OnError = append(out.OnError, NotificationEmail{Email: e.Email.Or("")})
	}
	for _, e := range n.OnFailure {
		out.OnFailure = append(out.OnFailure, NotificationEmail{Email: e.Email.Or("")})
	}
	return out
}

func notificationsToAPI(n Notifications) api.MonitorNotifications {
	out := api.MonitorNotifications{}
	for _, e := range n.OnError {
		out.OnError = append(out.OnError, api.OnError{Email: api.NewOptString(e.Email)})
	}
	for _, e := range n.OnFailure {
		out.OnFailure = append(out.OnFailure, api.OnFailure{Email: api.NewOptString(e.Email)})
	}
	return out
}

// --- Update --------------------------------------------------------------

// UpdateInput holds the fields to update on a monitor's
// [configuration](https://learning.postman.com/docs/monitoring-your-api/setting-up-monitor/#configure-a-monitor).
// Zero-valued fields (and a nil Schedule/Retry/Options/Notifications) are left
// unchanged; use Active to explicitly activate or deactivate the monitor.
type UpdateInput struct {
	// Name, if non-empty, renames the monitor.
	Name string
	// Active, if non-nil, activates or deactivates the monitor.
	Active *bool
	// NotificationLimit, if non-zero, updates the daily notification cap.
	NotificationLimit int
	Retry             *RetrySettings
	Options           *Options
	Schedule          *Schedule
	Distribution      []Region
	Notifications     *Notifications
}

// Update updates a monitor's configuration.
func (s *Service) Update(ctx context.Context, monitorID string, in *UpdateInput) (*MonitorRef, error) {
	if in == nil {
		in = &UpdateInput{}
	}
	monitor := api.UpdateMonitorMonitor{
		NotificationLimit: intToRaw(in.NotificationLimit),
	}
	if in.Name != "" {
		monitor.Name = api.NewOptString(in.Name)
	}
	if in.Active != nil {
		monitor.Active = api.NewOptBool(*in.Active)
	}
	if in.Retry != nil {
		monitor.Retry = api.NewOptMonitorRetrySettings(api.MonitorRetrySettings{
			Attempts: intToRaw(in.Retry.Attempts),
		})
	}
	if in.Options != nil {
		monitor.Options = api.NewOptMonitorOptions(optionsToAPI(*in.Options))
	}
	if in.Schedule != nil {
		monitor.Schedule = api.NewOptMonitorSchedule(api.MonitorSchedule{
			Cron:     api.NewOptString(in.Schedule.Cron),
			Timezone: api.NewOptString(in.Schedule.Timezone),
		})
	}
	for _, r := range in.Distribution {
		monitor.Distribution = append(monitor.Distribution, api.MonitorDistribution{
			Region: api.NewOptRegion(api.Region(r)),
		})
	}
	if in.Notifications != nil {
		monitor.Notifications = api.NewOptMonitorNotifications(notificationsToAPI(*in.Notifications))
	}

	req := &api.UpdateMonitor{Monitor: api.NewOptUpdateMonitorMonitor(monitor)}
	params := api.UpdateMonitorParams{MonitorId: monitorID}

	res, err := s.api.UpdateMonitor(ctx, req, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.CreateUpdateMonitorResponse:
		return monitorRefFromAPI(r), nil
	case *api.Monitors400Errors:
		return nil, postmanerr.Empty(http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetAuditLogEventActionsClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Delete --------------------------------------------------------------

// DeleteResult confirms the deletion of a monitor.
type DeleteResult struct {
	ID  string
	UID string
}

// Delete deletes a monitor.
func (s *Service) Delete(ctx context.Context, monitorID string) (*DeleteResult, error) {
	params := api.DeleteMonitorParams{MonitorId: monitorID}

	res, err := s.api.DeleteMonitor(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.MonitorDeleted:
		out := &DeleteResult{}
		if m, ok := r.Monitor.Get(); ok {
			out.ID = m.ID.Or("")
			out.UID = m.UID.Or("")
		}
		return out, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetAuditLogEventActionsClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Run --------------------------------------------------------------

// RunInput holds options for Run.
type RunInput struct {
	// Async, when true, runs the monitor asynchronously from the created
	// monitor run task; the server returns immediately rather than waiting up
	// to 300 seconds for the run to finish.
	Async bool
}

// RunResult acknowledges that a monitor run was started.
//
// Known limitation: Postman's generated schema for this endpoint's response
// body is a union ogen cannot resolve to a concrete shape, so it collapses to
// an empty object; run statistics are not available here. Call Get to
// retrieve the monitor's LastRun details. Similarly, a run that exceeds 300
// seconds returns HTTP 202 with a body ogen has no case for, which surfaces
// as an error from this method; pass Async: true to avoid triggering it.
type RunResult struct{}

// Run runs a monitor and returns its run results.
func (s *Service) Run(ctx context.Context, monitorID string, in *RunInput) (*RunResult, error) {
	if in == nil {
		in = &RunInput{}
	}
	params := api.RunMonitorParams{MonitorId: monitorID}
	if in.Async {
		params.Async = api.NewOptBool(true)
	}

	res, err := s.api.RunMonitor(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.RunMonitorOK:
		_ = r
		return &RunResult{}, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetAuditLogEventActionsClientErrorResponse:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- RunnerInstances --------------------------------------------------------------

// RunnerInstance is a single execution instance of a runner polling Postman
// for upcoming monitor runs.
type RunnerInstance struct {
	ID           string
	Hostname     string
	UniqueID     string
	CLIVersion   string
	OSType       string
	RunnerID     string
	LastPingedAt string
}

// RunnerInstancesInput holds pagination options for RunnerInstances.
type RunnerInstancesInput struct {
	// Limit is the maximum number of rows to return, up to 25.
	Limit int
	// Cursor is the pagination cursor (use RunnerInstancesResult.NextCursor to page).
	Cursor string
}

// RunnerInstancesResult is the paginated set of runner instances.
type RunnerInstancesResult struct {
	Instances  []RunnerInstance
	NextCursor string
}

// RunnerInstances gets all instances of the runner polling Postman for
// upcoming monitor runs. Instances are runner executions that share the same
// runner ID and key. You can find a runner's ID in the Postman UI under
// Team > Team Settings > Runners.
func (s *Service) RunnerInstances(ctx context.Context, runnerID string, in *RunnerInstancesInput) (*RunnerInstancesResult, error) {
	if in == nil {
		in = &RunnerInstancesInput{}
	}
	params := api.GetRunnerInstancesParams{RunnerId: runnerID}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Cursor != "" {
		params.Cursor = api.NewOptString(in.Cursor)
	}

	res, err := s.api.GetRunnerInstances(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetRunnerInstances:
		out := &RunnerInstancesResult{}
		for _, d := range r.Data {
			out.Instances = append(out.Instances, RunnerInstance{
				ID:           d.ID.Or(""),
				Hostname:     d.Hostname.Or(""),
				UniqueID:     d.UniqueId.Or(""),
				CLIVersion:   d.CliVersion.Or(""),
				OSType:       d.OsType.Or(""),
				RunnerID:     d.RunnerId.Or(""),
				LastPingedAt: d.LastPingedAt.Or(""),
			})
		}
		if meta, ok := r.Meta.Get(); ok {
			out.NextCursor = meta.NextCursor.Or("")
		}
		return out, nil
	case *api.GetRunnerInstancesBadRequest:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.GetRunnerInstancesNotFound:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- RunnerMetrics --------------------------------------------------------------

// RunnerMetrics summarizes the Postman server-side metrics for a runner
// instance, such as monitor run queues and the last polling date.
type RunnerMetrics struct {
	LastPingAt                string
	OldestQueuedRunAgeSeconds int
	QueueDepth                int
}

// RunnerMetrics gets the Postman server-side metrics for a runner instance.
// You can find a runner's ID in the Postman UI under
// Team > Team Settings > Runners.
func (s *Service) RunnerMetrics(ctx context.Context, runnerID string) (*RunnerMetrics, error) {
	params := api.GetRunnerMetricsParams{RunnerId: runnerID}

	res, err := s.api.GetRunnerMetrics(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetRunnerMetrics:
		return &RunnerMetrics{
			LastPingAt:                r.LastPingAt.Or(""),
			OldestQueuedRunAgeSeconds: r.OldestQueuedRunAgeSeconds.Or(0),
			QueueDepth:                r.QueueDepth.Or(0),
		}, nil
	case *api.Common401Error:
		return nil, postmanerr.Empty(http.StatusUnauthorized)
	case *api.Common403Error:
		return nil, postmanerr.Empty(http.StatusForbidden)
	case *api.ErrorTypeTitleDetailCreatedAt:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusNotFound)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- helpers --------------------------------------------------------------

// rawToInt best-effort parses a raw JSON integer field into an int. Postman's
// generated schema represents a handful of numeric monitor fields as raw JSON
// (see the "Known approximations" note in scripts/gen-openapi/README.md);
// this recovers the common case where the value is a plain JSON integer.
func rawToInt(r jx.Raw) int {
	if len(r) == 0 {
		return 0
	}
	v, err := strconv.Atoi(string(r))
	if err != nil {
		return 0
	}
	return v
}

// intToRaw is the inverse of rawToInt. Zero is treated as "unset" and encodes
// to an omitted field.
func intToRaw(v int) jx.Raw {
	if v == 0 {
		return nil
	}
	return jx.Raw(strconv.Itoa(v))
}

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
