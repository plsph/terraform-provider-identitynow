package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure IdentityNowProvider implements provider.Provider
var _ provider.Provider = &IdentityNowProvider{}

// IdentityNowProvider defines the provider implementation
type IdentityNowProvider struct {
	version string
}

// IdentityNowProviderModel describes the provider data model
type IdentityNowProviderModel struct {
	ApiUrl                 types.String `tfsdk:"api_url"`
	ClientId               types.String `tfsdk:"client_id"`
	ClientSecret           types.String `tfsdk:"client_secret"`
	Credentials            types.List   `tfsdk:"credentials"`
	MaxClientPoolSize      types.Int64  `tfsdk:"max_client_pool_size"`
	DefaultClientPoolSize  types.Int64  `tfsdk:"default_client_pool_size"`
	ClientRequestRateLimit types.Int64  `tfsdk:"client_request_rate_limit"`
}

// CredentialModel describes a single credential
type CredentialModel struct {
	ClientId     types.String `tfsdk:"client_id"`
	ClientSecret types.String `tfsdk:"client_secret"`
}

// New returns a new provider instance
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &IdentityNowProvider{
			version: version,
		}
	}
}

// Metadata returns the provider type name
func (p *IdentityNowProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "identitynow"
	resp.Version = p.version
}

// Schema defines the provider schema
func (p *IdentityNowProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Terraform provider for SailPoint IdentityNow",
		Attributes: map[string]schema.Attribute{
			"api_url": schema.StringAttribute{
				Description: "The URL to the IdentityNow API, e.g. https://<tenant>.api.identitynow.com. Can also be set with the IDENTITYNOW_URL environment variable.",
				Optional:    true,
			},
			"client_id": schema.StringAttribute{
				Description: "API client used to authenticate with the IdentityNow API",
				Optional:    true,
				Sensitive:   true,
			},
			"client_secret": schema.StringAttribute{
				Description: "API client secret used to authenticate with the IdentityNow API",
				Optional:    true,
				Sensitive:   true,
			},
			"credentials": schema.ListNestedAttribute{
				Description: "API client id and secret sets used to authenticate with the IdentityNow API",
				Optional:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"client_id": schema.StringAttribute{
							Description: "Client ID",
							Required:    true,
						},
						"client_secret": schema.StringAttribute{
							Description: "Client Secret",
							Required:    true,
							Sensitive:   true,
						},
					},
				},
			},
			"max_client_pool_size": schema.Int64Attribute{
				Description: "Max client pool size for communication with the IdentityNow API. Can also be set with the IDENTITYNOW_MAX_POOL_SIZE environment variable. Defaults to 1.",
				Optional:    true,
				Validators:  []validator.Int64{int64AtLeastValidator{min: 1}},
			},
			"default_client_pool_size": schema.Int64Attribute{
				Description: "Default client pool size for communication with the IdentityNow API. Can also be set with the IDENTITYNOW_DEF_POOL_SIZE environment variable. Defaults to 1.",
				Optional:    true,
				Validators:  []validator.Int64{int64AtLeastValidator{min: 1}},
			},
			"client_request_rate_limit": schema.Int64Attribute{
				Description: "Client request rate limit in requests per second for communication with the IdentityNow API. Can also be set with the IDENTITYNOW_CLI_RQ_RATE environment variable. Defaults to 10.",
				Optional:    true,
				Validators:  []validator.Int64{int64AtLeastValidator{min: 1}},
			},
		},
	}
}

// Configure configures the provider with the given configuration
func (p *IdentityNowProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	tflog.Info(ctx, "Configuring IdentityNow provider")

	var data IdentityNowProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	for name, value := range map[string]attr.Value{
		"api_url":                   data.ApiUrl,
		"client_id":                 data.ClientId,
		"client_secret":             data.ClientSecret,
		"credentials":               data.Credentials,
		"max_client_pool_size":      data.MaxClientPoolSize,
		"default_client_pool_size":  data.DefaultClientPoolSize,
		"client_request_rate_limit": data.ClientRequestRateLimit,
	} {
		if value.IsUnknown() {
			resp.Diagnostics.AddAttributeError(
				path.Root(name),
				"Unknown provider configuration value",
				fmt.Sprintf("The provider cannot be configured because %s is unknown. Set it to a value known at plan time or use its environment variable.", name),
			)
		}
	}
	if resp.Diagnostics.HasError() {
		return
	}

	// Set default values from environment variables
	apiURL := stringFromEnv(data.ApiUrl, "IDENTITYNOW_URL")
	clientID := stringFromEnv(data.ClientId, "IDENTITYNOW_CLIENT_ID")
	clientSecret := stringFromEnv(data.ClientSecret, "IDENTITYNOW_CLIENT_SECRET")
	maxPoolSize := int64FromEnv(data.MaxClientPoolSize, "IDENTITYNOW_MAX_POOL_SIZE", "max_client_pool_size", 1, &resp.Diagnostics)
	defaultPoolSize := int64FromEnv(data.DefaultClientPoolSize, "IDENTITYNOW_DEF_POOL_SIZE", "default_client_pool_size", 1, &resp.Diagnostics)
	rateLimit := int64FromEnv(data.ClientRequestRateLimit, "IDENTITYNOW_CLI_RQ_RATE", "client_request_rate_limit", 10, &resp.Diagnostics)

	// Validate required fields
	if apiURL == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Missing IdentityNow API URL",
			"The provider cannot create the IdentityNow API client as there is a missing or empty value for the IdentityNow API URL. "+
				"Set the api_url value in the configuration or use the IDENTITYNOW_URL environment variable. ",
		)
	} else if !strings.Contains(apiURL, "://") {
		apiURL = "https://" + apiURL
	}

	// Parse credentials
	credentials := []ClientCredential{}
	if !data.Credentials.IsNull() && len(data.Credentials.Elements()) > 0 {
		var credsList []CredentialModel
		resp.Diagnostics.Append(data.Credentials.ElementsAs(ctx, &credsList, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		for _, cred := range credsList {
			credentials = append(credentials, ClientCredential{
				ClientId:     cred.ClientId.ValueString(),
				ClientSecret: cred.ClientSecret.ValueString(),
			})
		}
	} else if clientID != "" && clientSecret != "" {
		credentials = []ClientCredential{{
			ClientId:     clientID,
			ClientSecret: clientSecret,
		}}
	} else {
		resp.Diagnostics.AddError(
			"Missing IdentityNow API credentials",
			"Set client_id and client_secret, or credentials, in the provider configuration, "+
				"or use the IDENTITYNOW_CLIENT_ID and IDENTITYNOW_CLIENT_SECRET environment variables.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Provider configuration", map[string]interface{}{
		"api_url":                   apiURL,
		"credentials_pool_size":     len(credentials),
		"max_client_pool_size":      maxPoolSize,
		"default_client_pool_size":  defaultPoolSize,
		"client_request_rate_limit": rateLimit,
	})

	config := &Config{
		URL:                    apiURL,
		Credentials:            credentials,
		MaxClientPoolSize:      int(maxPoolSize),
		DefaultClientPoolSize:  int(defaultPoolSize),
		ClientRequestRateLimit: int(rateLimit),
	}

	resp.DataSourceData = config
	resp.ResourceData = config

	tflog.Info(ctx, "Successfully configured IdentityNow provider")
}

// stringFromEnv returns the configured value, or the environment variable when the value is not set.
func stringFromEnv(value types.String, envVar string) string {
	if !value.IsNull() && value.ValueString() != "" {
		return value.ValueString()
	}
	return os.Getenv(envVar)
}

// int64FromEnv returns the configured value, or the environment variable, or the default when neither is set.
// Values must be at least 1.
func int64FromEnv(value types.Int64, envVar string, attribute string, defaultValue int64, diags *diag.Diagnostics) int64 {
	if !value.IsNull() {
		return value.ValueInt64()
	}
	raw := os.Getenv(envVar)
	if raw == "" {
		return defaultValue
	}
	parsed, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
	if err != nil || parsed < 1 {
		diags.AddAttributeError(
			path.Root(attribute),
			"Invalid environment variable value",
			fmt.Sprintf("%s must be an integer of at least 1, got %q.", envVar, raw),
		)
		return defaultValue
	}
	return parsed
}

// Resources returns the list of resources for this provider
func (p *IdentityNowProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSourceResource,
		NewAccessProfileResource,
		NewRoleResource,
		NewGovernanceGroupResource,
		NewSourceAppResource,
		NewAccessProfileAttachmentResource,
		NewGovernanceGroupMembersResource,
		NewAccountSchemaResource,
		NewPasswordPolicyResource,
		NewScheduleAccountAggregationResource,
		NewTaggedObjectResource,
		NewDimensionResource,
		NewWorkflowResource,
		NewSegmentResource,
		NewFormDefinitionResource,
	}
}

// DataSources returns the list of data sources for this provider
func (p *IdentityNowProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRoleDataSource,
		NewSourceDataSource,
		NewAccessProfileDataSource,
		NewIdentityDataSource,
		NewGovernanceGroupDataSource,
		NewSourceAppDataSource,
		NewSourceEntitlementDataSource,
		NewDimensionDataSource,
		NewWorkflowDataSource,
		NewSegmentDataSource,
		NewFormDefinitionDataSource,
	}
}
