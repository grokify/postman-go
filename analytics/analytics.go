// Package analytics provides a high-level client for Postman's Analytics
// API.
//
// It returns metrics and insights for API usage, success, and
// workspace/team trends in Postman, and a catalog describing which metrics
// are available for which resources.
//
// Example:
//
//	client, _ := postman.NewClient(postman.WithAPIKey(key))
//	data, _ := client.Analytics().Data(ctx, &analytics.DataInput{
//		Resource: analytics.ResourceWorkspace,
//		Metrics:  analytics.MetricActiveWorkspaces,
//	})
package analytics

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/grokify/postman-go/internal/api"
	"github.com/grokify/postman-go/postmanerr"
)

// Resource identifies the kind of Postman entity analytics data or metadata
// is scoped to.
type Resource string

// Resource values.
const (
	// ResourceUser covers individual user activity and engagement.
	ResourceUser Resource = "user"
	// ResourceTeam covers team-level analytics, license consumption, and
	// organizational trends.
	ResourceTeam Resource = "team"
	// ResourceWorkspace covers workspace-level activity, elements, and
	// collaboration patterns.
	ResourceWorkspace Resource = "workspace"
	// ResourceAI covers Agent Mode usage across workspaces.
	ResourceAI Resource = "ai"
	// ResourceAPIDevelopment covers API development activity metrics.
	ResourceAPIDevelopment Resource = "api_development"
	// ResourceAPITesting covers API testing metrics.
	ResourceAPITesting Resource = "api_testing"
	// ResourceAPIProduction covers API production metrics.
	ResourceAPIProduction Resource = "api_production"
	// ResourceAPIDistribution covers API distribution metrics.
	ResourceAPIDistribution Resource = "api_distribution"
	// ResourceAPIManagement covers API management metrics.
	ResourceAPIManagement Resource = "api_management"
)

// Metric identifies a specific analytics metric. The metric must match the
// Resource it is queried with; see Metadata for the current catalog of
// valid resource/metric pairs.
type Metric string

// Metric values.
const (
	MetricActiveUsers                   Metric = "active_users"
	MetricWorkspaceActiveUsers          Metric = "workspace_active_users"
	MetricElementsInWorkspace           Metric = "elements_in_workspace"
	MetricActiveWorkspaces              Metric = "active_workspaces"
	MetricAPICalls                      Metric = "api_calls"
	MetricActiveCollections             Metric = "active_collections"
	MetricResponseStatus                Metric = "response_status"
	MetricPendingInvites                Metric = "pending_invites"
	MetricNeedsAttention                Metric = "needs_attention"
	MetricSuccessRate                   Metric = "success_rate"
	MetricUserRequests                  Metric = "user_requests"
	MetricUserAPIJourney                Metric = "user_api_journey"
	MetricWorkspaceDistribution         Metric = "workspace_distribution"
	MetricInternalWorkspaceDistribution Metric = "internal_workspace_distribution"
	MetricLicenseConsumption            Metric = "license_consumption"
	MetricMembers                       Metric = "members"
	MetricLastAutoflexCycle             Metric = "last_autoflex_cycle"
	MetricPartnerEngagementFunnel       Metric = "partner_engagement_funnel"
	MetricCollectionErrorAggregate      Metric = "collection_error_aggregate"
	MetricAgentModeUsers                Metric = "agent_mode_users"
	MetricNewVsReturningUsers           Metric = "new_vs_returning_users"
	MetricAgentModeSessions             Metric = "agent_mode_sessions"
	MetricMessagesSent                  Metric = "messages_sent"
	MetricCreditUsage                   Metric = "credit_usage"
	MetricCreditUsageByModel            Metric = "credit_usage_by_model"
	MetricUsageLeaderboard              Metric = "usage_leaderboard"
	MetricPeakActivity                  Metric = "peak_activity"
	MetricActivityDistribution          Metric = "activity_distribution"
	MetricTopAgentModelsByUsage         Metric = "top_agent_models_by_usage"
	MetricEntityActivity                Metric = "entity_activity"
	MetricTopEntities                   Metric = "top_entities"
	MetricRuns                          Metric = "runs"
	MetricFunctionalTestRuns            Metric = "functional_test_runs"
	MetricPerformanceTestRuns           Metric = "performance_test_runs"
	MetricMonitorRuns                   Metric = "monitor_runs"
	MetricFlowExecutions                Metric = "flow_executions"
	MetricPvtNetwork                    Metric = "pvt_network"
	MetricPartner                       Metric = "partner"
	MetricPublic                        Metric = "public"
	MetricWorkspaceActivity             Metric = "workspace_activity"
	MetricMembersOvertime               Metric = "members_overtime"
	MetricMemberInvites                 Metric = "member_invites"
	MetricInvitesSent                   Metric = "invites_sent"
	MetricInvitesAccepted               Metric = "invites_accepted"
)

// View selects the shape of returned analytics data.
type View string

// View values.
const (
	// ViewDetailed returns extensive information.
	ViewDetailed View = "detailed"
	// ViewSummary returns aggregated information.
	ViewSummary View = "summary"
	// ViewTrend returns trend information over a duration.
	ViewTrend View = "trend"
)

// Duration filters analytics data to a relative time window.
type Duration string

// Duration values.
const (
	DurationLast7Days   Duration = "last_7_days"
	DurationLast30Days  Duration = "last_30_days"
	DurationLast180Days Duration = "last_180_days"
	DurationLastMonth   Duration = "last_month"
	DurationLast6Months Duration = "last_6_months"
	DurationLast1Year   Duration = "last_1_year"
)

// UserType filters analytics data by user type.
type UserType string

// UserType values.
const (
	UserTypeNew       UserType = "new"
	UserTypeReturning UserType = "returning"
)

// EntityType filters analytics data by Postman entity type.
type EntityType string

// EntityType values.
const (
	EntityTypeCollection       EntityType = "collection"
	EntityTypeSpecification    EntityType = "specification"
	EntityTypeMock             EntityType = "mock"
	EntityTypeFlow             EntityType = "flow"
	EntityTypeSDKCollection    EntityType = "sdk-collection"
	EntityTypeSDKSpecification EntityType = "sdk-specification"
)

// Service is the high-level Analytics client. Obtain one via
// postman.Client.Analytics.
type Service struct {
	api *api.Client
}

// New creates an Analytics service over the given generated API client.
// Most callers should use postman.Client.Analytics instead of calling this
// directly.
func New(apiClient *api.Client) *Service {
	return &Service{api: apiClient}
}

// --- Data -------------------------------------------------------------------

// DataInput holds the filters for Data.
//
// Resource and Metrics are required and must be a valid combination; call
// Metadata to discover valid resource/metric pairs.
type DataInput struct {
	Resource Resource
	Metrics  Metric

	// View is the shape of returned data; only some resource/metric pairs
	// support each value.
	View View
	// WorkspaceType is a comma-separated list of internal, public, and
	// partner workspace types to filter by.
	WorkspaceType string
	// UserID is a comma-separated list of user IDs to filter by. Only used
	// with the user_requests metric for the workspace resource.
	UserID string
	// Duration filters the response by the given duration.
	Duration Duration
	// RequestID is a comma-separated list of userId-requestId pairs to
	// filter by. Only used with the user_requests metric.
	RequestID string
	// ResponseStatus is a comma-separated list of HTTP status codes
	// (100-600) to filter by. Only used with the user_requests metric.
	ResponseStatus string
	// AttentionType is a comma-separated list of issue types to filter by.
	// Only used with the needs_attention metric.
	AttentionType string
	// Period filters results for a given period of time (YEAR-MONTH or
	// YEAR-MONTH-DAY).
	Period string
	// UserType filters results by user type for supported views.
	UserType UserType
	// EntityType filters results by Postman entity type.
	EntityType EntityType
	// Limit is the maximum number of rows to return.
	Limit int
	// Offset is the zero-based offset of the first item to return.
	Offset int
}

// DataResult is the analytics data returned by Data. Data is free-form JSON:
// its shape depends on the requested Resource/Metrics/View combination.
type DataResult struct {
	Data json.RawMessage
}

// Data returns analytics data for the given resource, metrics, and filters,
// across team, internal, public, and Partner Workspaces.
func (s *Service) Data(ctx context.Context, in *DataInput) (*DataResult, error) {
	if in == nil {
		in = &DataInput{}
	}
	params := api.GetAnalyticsDataParams{
		Resource: api.AnalyticsResource(in.Resource),
		Metrics:  api.AnalyticsMetrics(in.Metrics),
	}
	if in.View != "" {
		params.View = api.NewOptAnalyticsView(api.AnalyticsView(in.View))
	}
	if in.WorkspaceType != "" {
		params.WorkspaceType = api.NewOptString(in.WorkspaceType)
	}
	if in.UserID != "" {
		params.UserId = api.NewOptString(in.UserID)
	}
	if in.Duration != "" {
		params.Duration = api.NewOptAnalyticsDuration(api.AnalyticsDuration(in.Duration))
	}
	if in.RequestID != "" {
		params.RequestId = api.NewOptString(in.RequestID)
	}
	if in.ResponseStatus != "" {
		params.ResponseStatus = api.NewOptString(in.ResponseStatus)
	}
	if in.AttentionType != "" {
		params.AttentionType = api.NewOptString(in.AttentionType)
	}
	if in.Period != "" {
		params.Period = api.NewOptString(in.Period)
	}
	if in.UserType != "" {
		params.UserType = api.NewOptAnalyticsUserType(api.AnalyticsUserType(in.UserType))
	}
	if in.EntityType != "" {
		params.EntityType = api.NewOptAnalyticsEntityType(api.AnalyticsEntityType(in.EntityType))
	}
	if in.Limit > 0 {
		params.Limit = api.NewOptInt(in.Limit)
	}
	if in.Offset > 0 {
		params.Offset = api.NewOptInt(in.Offset)
	}

	res, err := s.api.GetAnalyticsData(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetAnalyticsData:
		return &DataResult{Data: json.RawMessage(r.Data)}, nil
	case *api.ErrorTypeTitleDetailStatus:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusBadRequest)
	case *api.GetAnalyticsDataUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.GetAnalyticsDataForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- Metadata -----------------------------------------------------------

// MetadataInput holds the filters for Metadata.
type MetadataInput struct {
	// Include is a comma-separated list of additional information to
	// include in the response. Accepts "parameters" and "response".
	Include string
	// Resources is a comma-separated list of resource types to filter the
	// metrics by. Accepts "user", "workspace", "team", and "ai".
	Resources string
	// Metrics is a comma-separated list of metrics to filter the response
	// by. If empty, the response returns metadata for all available
	// metrics.
	Metrics Metric
}

// MetadataResult is the analytics resource/metric catalog returned by
// Metadata.
type MetadataResult struct {
	Description string
	// Resources is free-form JSON: each entry's shape depends on which
	// Include values were requested.
	Resources []json.RawMessage
}

// Metadata returns the catalog of analytics resources and their
// corresponding metrics for use with Data.
func (s *Service) Metadata(ctx context.Context, in *MetadataInput) (*MetadataResult, error) {
	if in == nil {
		in = &MetadataInput{}
	}
	params := api.GetAnalyticsMetadataParams{}
	if in.Include != "" {
		params.Include = api.NewOptString(in.Include)
	}
	if in.Resources != "" {
		params.Resources = api.NewOptString(in.Resources)
	}
	if in.Metrics != "" {
		params.Metrics = api.NewOptAnalyticsMetrics(api.AnalyticsMetrics(in.Metrics))
	}

	res, err := s.api.GetAnalyticsMetadata(ctx, params)
	if err != nil {
		return nil, err
	}

	switch r := res.(type) {
	case *api.GetAnalyticsMetadata:
		out := &MetadataResult{}
		if data, ok := r.Data.Get(); ok {
			out.Description = data.Description.Or("")
			for _, dr := range data.Resources {
				out.Resources = append(out.Resources, json.RawMessage(dr))
			}
		}
		return out, nil
	case *api.GetAnalyticsMetadataUnauthorized:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusUnauthorized)
	case *api.GetAnalyticsMetadataForbidden:
		return nil, postmanerr.FromProblemDetails([]byte(*r), http.StatusForbidden)
	case *api.Common500Error:
		return nil, postmanerr.Empty(http.StatusInternalServerError)
	default:
		return nil, errUnexpectedResponse(res)
	}
}

// --- error helpers ----------------------------------------------------------

func errUnexpectedResponse(_ any) error {
	return &postmanerr.APIError{Title: "unexpected response type from API"}
}
