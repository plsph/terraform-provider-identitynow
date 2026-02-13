package main

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	providerDefaultEmptyString = "nil"
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
				Description: "The URL to the IdentityNow API",
				Required:    true,
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
				Description: "Max client pool size for communication with the IdentityNow API",
				Optional:    true,
			},
			"default_client_pool_size": schema.Int64Attribute{
				Description: "Default client pool size for communication with the IdentityNow API",
				Optional:    true,
			},
			"client_request_rate_limit": schema.Int64Attribute{
				Description: "Client request rate limit for communication with the IdentityNow API",
				Optional:    true,
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

	// Set default values from environment variables
	if data.ApiUrl.IsNull() {
		apiUrl := os.Getenv("IDENTITYNOW_URL")
		if apiUrl == "" {
			apiUrl = providerDefaultEmptyString
		}
		data.ApiUrl = types.StringValue(apiUrl)
	}

	if data.ClientId.IsNull() {
		clientId := os.Getenv("IDENTITYNOW_CLIENT_ID")
		if clientId == "" {
			clientId = providerDefaultEmptyString
		}
		data.ClientId = types.StringValue(clientId)
	}

	if data.ClientSecret.IsNull() {
		clientSecret := os.Getenv("IDENTITYNOW_CLIENT_SECRET")
		if clientSecret == "" {
			clientSecret = providerDefaultEmptyString
		}
		data.ClientSecret = types.StringValue(clientSecret)
	}

	if data.MaxClientPoolSize.IsNull() {
		maxPoolSize := os.Getenv("IDENTITYNOW_MAX_POOL_SIZE")
		if maxPoolSize == "" {
			data.MaxClientPoolSize = types.Int64Value(1)
		}
	}

	if data.DefaultClientPoolSize.IsNull() {
		defPoolSize := os.Getenv("IDENTITYNOW_DEF_POOL_SIZE")
		if defPoolSize == "" {
			data.DefaultClientPoolSize = types.Int64Value(1)
		}
	}

	if data.ClientRequestRateLimit.IsNull() {
		rateLimit := os.Getenv("IDENTITYNOW_CLI_RQ_RATE")
		if rateLimit == "" {
			data.ClientRequestRateLimit = types.Int64Value(10)
		}
	}

	// Validate required fields
	if data.ApiUrl.ValueString() == providerDefaultEmptyString {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_url"),
			"Missing IdentityNow API URL",
			"The provider cannot create the IdentityNow API client as there is a missing or empty value for the IdentityNow API URL. "+
				"Set the api_url value in the configuration or use the IDENTITYNOW_URL environment variable. ",
		)
	}

	if resp.Diagnostics.HasError() {
		return
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
	} else {
		credentials = []ClientCredential{{
			ClientId:     data.ClientId.ValueString(),
			ClientSecret: data.ClientSecret.ValueString(),
		}}
	}

	tflog.Debug(ctx, "Provider configuration", map[string]interface{}{
		"api_url":                    data.ApiUrl.ValueString(),
		"credentials_pool_size":      len(credentials),
		"max_client_pool_size":       data.MaxClientPoolSize.ValueInt64(),
		"default_client_pool_size":   data.DefaultClientPoolSize.ValueInt64(),
		"client_request_rate_limit":  data.ClientRequestRateLimit.ValueInt64(),
	})

	config := &Config{
		URL:                    data.ApiUrl.ValueString(),
		ClientId:               data.ClientId.ValueString(),
		ClientSecret:           data.ClientSecret.ValueString(),
		Credentials:            credentials,
		MaxClientPoolSize:      int(data.MaxClientPoolSize.ValueInt64()),
		DefaultClientPoolSize:  int(data.DefaultClientPoolSize.ValueInt64()),
		ClientRequestRateLimit: int(data.ClientRequestRateLimit.ValueInt64()),
	}

	resp.DataSourceData = config
	resp.ResourceData = config

	tflog.Info(ctx, "Successfully configured IdentityNow provider")
}

// Resources returns the list of resources for this provider
func (p *IdentityNowProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewSourceResource,
		NewAccessProfileResource,
		NewRoleResource,
	}
}

// DataSources returns the list of data sources for this provider
func (p *IdentityNowProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewRoleDataSource,
	}
}
