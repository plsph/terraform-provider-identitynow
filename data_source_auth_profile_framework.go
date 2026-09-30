package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerDataSource(NewAuthProfileDataSource)
}

// AuthProfileSummary is an entry of the experimental /v2026/auth-profiles list.
type AuthProfileSummary struct {
	ID     string `json:"id"`
	Tenant string `json:"tenant"`
}

// AuthProfile is an auth profile as returned by the experimental /v2026/auth-profiles/{id} API.
type AuthProfile struct {
	Name               string  `json:"name"`
	OffNetwork         bool    `json:"offNetwork"`
	UntrustedGeography bool    `json:"untrustedGeography"`
	ApplicationID      *string `json:"applicationId"`
	ApplicationName    *string `json:"applicationName"`
	Type               string  `json:"type"`
	StrongAuthLogin    bool    `json:"strongAuthLogin"`
}

func (c *Client) ListAuthProfiles(ctx context.Context) ([]AuthProfileSummary, error) {
	var profiles []AuthProfileSummary
	if err := c.doJSON(ctx, http.MethodGet, "/v2026/auth-profiles", nil, &profiles, withExperimental()); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (c *Client) GetAuthProfile(ctx context.Context, id string) (*AuthProfile, error) {
	var profile AuthProfile
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/auth-profiles/%s", id), nil, &profile, withExperimental()); err != nil {
		return nil, err
	}
	return &profile, nil
}

// GetAuthProfileByName finds an auth profile by name. The list only contains IDs, so every
// profile is read until the names are known; it errors when several profiles have the name.
func (c *Client) GetAuthProfileByName(ctx context.Context, name string) (string, *AuthProfile, error) {
	summaries, err := c.ListAuthProfiles(ctx)
	if err != nil {
		return "", nil, err
	}
	var matchID string
	var match *AuthProfile
	for _, summary := range summaries {
		profile, err := c.GetAuthProfile(ctx, summary.ID)
		if err != nil {
			if isNotFound(err) {
				continue
			}
			return "", nil, err
		}
		if profile.Name != name {
			continue
		}
		if match != nil {
			return "", nil, fmt.Errorf("multiple auth profiles are named %q", name)
		}
		matchID, match = summary.ID, profile
	}
	if match == nil {
		return "", nil, &NotFoundError{fmt.Sprintf("auth profile with name %q not found", name)}
	}
	return matchID, match, nil
}

var _ datasource.DataSource = &AuthProfileDataSource{}
var _ datasource.DataSourceWithValidateConfig = &AuthProfileDataSource{}

func NewAuthProfileDataSource() datasource.DataSource {
	return &AuthProfileDataSource{}
}

type AuthProfileDataSource struct {
	client *Config
}

type AuthProfileModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Type               types.String `tfsdk:"type"`
	OffNetwork         types.Bool   `tfsdk:"off_network"`
	UntrustedGeography types.Bool   `tfsdk:"untrusted_geography"`
	ApplicationID      types.String `tfsdk:"application_id"`
	ApplicationName    types.String `tfsdk:"application_name"`
	StrongAuthLogin    types.Bool   `tfsdk:"strong_auth_login"`
}

func (d *AuthProfileDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_auth_profile"
}

func (d *AuthProfileDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up an authentication profile by ID or name. Uses an experimental API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Auth profile ID. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Auth profile name. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"type":                schema.StringAttribute{MarkdownDescription: "Auth profile type: `BLOCK`, `MFA`, `NON_PTA` or `PTA`", Computed: true},
			"off_network":         schema.BoolAttribute{MarkdownDescription: "Whether access from off network is blocked", Computed: true},
			"untrusted_geography": schema.BoolAttribute{MarkdownDescription: "Whether access from untrusted geographies is blocked", Computed: true},
			"application_id":      schema.StringAttribute{MarkdownDescription: "Application ID", Computed: true},
			"application_name":    schema.StringAttribute{MarkdownDescription: "Application name", Computed: true},
			"strong_auth_login":   schema.BoolAttribute{MarkdownDescription: "Whether strong authentication is enabled", Computed: true},
		},
	}
}

func (d *AuthProfileDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *AuthProfileDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *Config, got: %T", req.ProviderData))
		return
	}
	d.client = client
}

func setAuthProfileState(data *AuthProfileModel, id string, profile *AuthProfile) {
	data.ID = types.StringValue(id)
	data.Name = types.StringValue(profile.Name)
	data.Type = types.StringValue(profile.Type)
	data.OffNetwork = types.BoolValue(profile.OffNetwork)
	data.UntrustedGeography = types.BoolValue(profile.UntrustedGeography)
	data.ApplicationID = types.StringPointerValue(profile.ApplicationID)
	data.ApplicationName = types.StringPointerValue(profile.ApplicationName)
	data.StrongAuthLogin = types.BoolValue(profile.StrongAuthLogin)
}

func (d *AuthProfileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AuthProfileModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	id := data.ID.ValueString()
	var profile *AuthProfile
	if !data.ID.IsNull() {
		profile, err = client.GetAuthProfile(ctx, id)
	} else {
		id, profile, err = client.GetAuthProfileByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Auth profile not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read auth profile: %s", err))
		return
	}
	setAuthProfileState(&data, id, profile)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
