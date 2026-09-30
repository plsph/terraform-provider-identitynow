package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewOauthClientResource)
	registerDataSource(NewOauthClientDataSource)
}

// OauthClient is an API client as returned by the /v2026/oauth-clients API. Secret is only
// returned by the create call.
type OauthClient struct {
	ID                          string   `json:"id,omitempty"`
	Secret                      string   `json:"secret,omitempty"`
	BusinessName                *string  `json:"businessName,omitempty"`
	HomepageURL                 *string  `json:"homepageUrl,omitempty"`
	Name                        string   `json:"name"`
	Description                 string   `json:"description"`
	AccessTokenValiditySeconds  *int64   `json:"accessTokenValiditySeconds,omitempty"`
	RefreshTokenValiditySeconds *int64   `json:"refreshTokenValiditySeconds,omitempty"`
	RedirectURIs                []string `json:"redirectUris,omitempty"`
	GrantTypes                  []string `json:"grantTypes"`
	AccessType                  string   `json:"accessType,omitempty"`
	Type                        string   `json:"type,omitempty"`
	Internal                    *bool    `json:"internal,omitempty"`
	Enabled                     *bool    `json:"enabled,omitempty"`
	StrongAuthSupported         *bool    `json:"strongAuthSupported,omitempty"`
	ClaimsSupported             *bool    `json:"claimsSupported,omitempty"`
	Scope                       []string `json:"scope,omitempty"`
	Created                     string   `json:"created,omitempty"`
	Modified                    string   `json:"modified,omitempty"`
	LastUsed                    *string  `json:"lastUsed,omitempty"`
}

func (c *Client) GetOauthClient(ctx context.Context, id string) (*OauthClient, error) {
	var client OauthClient
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/oauth-clients/%s", id), nil, &client); err != nil {
		return nil, err
	}
	return &client, nil
}

func (c *Client) CreateOauthClient(ctx context.Context, client *OauthClient) (*OauthClient, error) {
	var created OauthClient
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/oauth-clients", client, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchOauthClient(ctx context.Context, id string, ops []jsonPatchOp) (*OauthClient, error) {
	var updated OauthClient
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/oauth-clients/%s", id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteOauthClient(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/oauth-clients/%s", id), nil, nil)
}

var _ resource.Resource = &OauthClientResource{}
var _ resource.ResourceWithImportState = &OauthClientResource{}

func NewOauthClientResource() resource.Resource {
	return &OauthClientResource{}
}

type OauthClientResource struct {
	client *Config
}

type OauthClientModel struct {
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	Description                 types.String `tfsdk:"description"`
	BusinessName                types.String `tfsdk:"business_name"`
	HomepageURL                 types.String `tfsdk:"homepage_url"`
	AccessTokenValiditySeconds  types.Int64  `tfsdk:"access_token_validity_seconds"`
	RefreshTokenValiditySeconds types.Int64  `tfsdk:"refresh_token_validity_seconds"`
	RedirectURIs                types.Set    `tfsdk:"redirect_uris"`
	GrantTypes                  types.Set    `tfsdk:"grant_types"`
	AccessType                  types.String `tfsdk:"access_type"`
	Type                        types.String `tfsdk:"type"`
	Internal                    types.Bool   `tfsdk:"internal"`
	Enabled                     types.Bool   `tfsdk:"enabled"`
	StrongAuthSupported         types.Bool   `tfsdk:"strong_auth_supported"`
	ClaimsSupported             types.Bool   `tfsdk:"claims_supported"`
	Scope                       types.Set    `tfsdk:"scope"`
	Secret                      types.String `tfsdk:"secret"`
	Created                     types.String `tfsdk:"created"`
	Modified                    types.String `tfsdk:"modified"`
}

func (r *OauthClientResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_client"
}

func (r *OauthClientResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages an OAuth (API) client. The client secret is only returned when the client is created and is stored in state as a sensitive value.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "OAuth client ID, used as the client ID in OAuth flows.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name of the API client.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the API client.",
				Required:            true,
			},
			"business_name": schema.StringAttribute{
				MarkdownDescription: "Name of the business the API client belongs to.",
				Optional:            true,
			},
			"homepage_url": schema.StringAttribute{
				MarkdownDescription: "Homepage URL associated with the owner of the API client.",
				Optional:            true,
			},
			"access_token_validity_seconds": schema.Int64Attribute{
				MarkdownDescription: "Number of seconds an access token generated for this API client is valid for.",
				Required:            true,
				Validators:          []validator.Int64{int64AtLeastValidator{min: 1}},
			},
			"refresh_token_validity_seconds": schema.Int64Attribute{
				MarkdownDescription: "Number of seconds a refresh token generated for this API client is valid for. Defaults to the value chosen by IdentityNow.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.Int64{int64AtLeastValidator{min: 1}},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			},
			"redirect_uris": schema.SetAttribute{
				MarkdownDescription: "Approved redirect URIs for the `AUTHORIZATION_CODE` grant type.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"grant_types": schema.SetAttribute{
				MarkdownDescription: "OAuth 2.0 grant types the client can be used with: `CLIENT_CREDENTIALS`, `AUTHORIZATION_CODE` or `REFRESH_TOKEN`.",
				Required:            true,
				ElementType:         types.StringType,
			},
			"access_type": schema.StringAttribute{
				MarkdownDescription: "Access type of the API client, `ONLINE` or `OFFLINE`.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Client type, `CONFIDENTIAL` or `PUBLIC`. Cannot be updated, changing it forces a new client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown(), stringplanmodifier.RequiresReplace()},
			},
			"internal": schema.BoolAttribute{
				MarkdownDescription: "Whether the API client can be used for requests internal to IdentityNow. Cannot be updated, changing it forces a new client.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown(), boolplanmodifier.RequiresReplace()},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the API client is enabled. Defaults to `true`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(true),
			},
			"strong_auth_supported": schema.BoolAttribute{
				MarkdownDescription: "Whether the API client supports strong authentication.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"claims_supported": schema.BoolAttribute{
				MarkdownDescription: "Whether the API client supports the serialization of SAML claims with the `AUTHORIZATION_CODE` flow.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"scope": schema.SetAttribute{
				MarkdownDescription: "Scopes of the API client. Defaults to `sp:scopes:all`, all rights of the owner. Cannot be updated, changing it forces a new client.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown(), setplanmodifier.RequiresReplace()},
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Client secret. It is only returned when the client is created, so it is null for imported clients.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Creation date.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"modified": schema.StringAttribute{
				MarkdownDescription: "Last modification date.",
				Computed:            true,
			},
		},
	}
}

func (r *OauthClientResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	client, ok := req.ProviderData.(*Config)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *Config, got: %T", req.ProviderData))
		return
	}
	r.client = client
}

// oauthClientStringSetValue returns the elements of a known string set, or nil for null and unknown sets.
func oauthClientStringSetValue(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}
	values := []string{}
	diags.Append(set.ElementsAs(ctx, &values, false)...)
	return values
}

// oauthClientStringSetState returns the state value of a string set read from the API. For optional
// attributes an empty API value keeps a null prior value.
func oauthClientStringSetState(ctx context.Context, prior types.Set, values []string, optional bool, diags *diag.Diagnostics) types.Set {
	if len(values) == 0 && optional && prior.IsNull() {
		return types.SetNull(types.StringType)
	}
	if values == nil {
		values = []string{}
	}
	set, d := types.SetValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return set
}

// oauthClientFromModel builds the create request. Unknown optional and computed values are omitted,
// so the API applies its defaults.
func oauthClientFromModel(ctx context.Context, data OauthClientModel, diags *diag.Diagnostics) *OauthClient {
	return &OauthClient{
		Name:                        data.Name.ValueString(),
		Description:                 data.Description.ValueString(),
		BusinessName:                stringPointer(data.BusinessName),
		HomepageURL:                 stringPointer(data.HomepageURL),
		AccessTokenValiditySeconds:  int64Pointer(data.AccessTokenValiditySeconds),
		RefreshTokenValiditySeconds: int64Pointer(data.RefreshTokenValiditySeconds),
		RedirectURIs:                oauthClientStringSetValue(ctx, data.RedirectURIs, diags),
		GrantTypes:                  oauthClientStringSetValue(ctx, data.GrantTypes, diags),
		AccessType:                  data.AccessType.ValueString(),
		Type:                        data.Type.ValueString(),
		Internal:                    boolPointer(data.Internal),
		Enabled:                     boolPointer(data.Enabled),
		StrongAuthSupported:         boolPointer(data.StrongAuthSupported),
		ClaimsSupported:             boolPointer(data.ClaimsSupported),
		Scope:                       oauthClientStringSetValue(ctx, data.Scope, diags),
	}
}

// oauthClientResolveUnknowns keeps the planned values and resolves the unknown computed values from
// the API response after create or update.
func oauthClientResolveUnknowns(ctx context.Context, data *OauthClientModel, client *OauthClient, diags *diag.Diagnostics) {
	if data.ID.IsUnknown() {
		data.ID = types.StringValue(client.ID)
	}
	if data.RefreshTokenValiditySeconds.IsUnknown() {
		data.RefreshTokenValiditySeconds = optionalInt64State(types.Int64Null(), client.RefreshTokenValiditySeconds)
	}
	data.Type = computedStringFromAPI(data.Type, client.Type)
	data.Internal = computedBoolFromAPI(data.Internal, client.Internal)
	data.Enabled = computedBoolFromAPI(data.Enabled, client.Enabled)
	data.StrongAuthSupported = computedBoolFromAPI(data.StrongAuthSupported, client.StrongAuthSupported)
	data.ClaimsSupported = computedBoolFromAPI(data.ClaimsSupported, client.ClaimsSupported)
	if data.Scope.IsUnknown() {
		data.Scope = oauthClientStringSetState(ctx, types.SetNull(types.StringType), client.Scope, false, diags)
	}
	if data.Created.IsUnknown() {
		data.Created = types.StringValue(client.Created)
	}
	data.Modified = types.StringValue(client.Modified)
	if data.Secret.IsUnknown() {
		// The secret is only returned by the create call.
		data.Secret = stringValueOrNull(client.Secret)
	}
}

// setOauthClientState refreshes the model from the API, keeping null for unset optional attributes.
// The secret is not returned by the API and keeps its state value.
func setOauthClientState(ctx context.Context, data *OauthClientModel, client *OauthClient, diags *diag.Diagnostics) {
	data.ID = types.StringValue(client.ID)
	data.Name = types.StringValue(client.Name)
	data.Description = types.StringValue(client.Description)
	data.BusinessName = optionalStringState(data.BusinessName, oauthClientStringValue(client.BusinessName))
	data.HomepageURL = optionalStringState(data.HomepageURL, oauthClientStringValue(client.HomepageURL))
	data.AccessTokenValiditySeconds = optionalInt64State(data.AccessTokenValiditySeconds, client.AccessTokenValiditySeconds)
	data.RefreshTokenValiditySeconds = optionalInt64State(data.RefreshTokenValiditySeconds, client.RefreshTokenValiditySeconds)
	data.RedirectURIs = oauthClientStringSetState(ctx, data.RedirectURIs, client.RedirectURIs, true, diags)
	data.GrantTypes = oauthClientStringSetState(ctx, data.GrantTypes, client.GrantTypes, false, diags)
	data.AccessType = types.StringValue(client.AccessType)
	data.Type = types.StringValue(client.Type)
	data.Internal = types.BoolValue(client.Internal != nil && *client.Internal)
	data.Enabled = types.BoolValue(client.Enabled != nil && *client.Enabled)
	data.StrongAuthSupported = types.BoolValue(client.StrongAuthSupported != nil && *client.StrongAuthSupported)
	data.ClaimsSupported = types.BoolValue(client.ClaimsSupported != nil && *client.ClaimsSupported)
	data.Scope = oauthClientStringSetState(ctx, data.Scope, client.Scope, false, diags)
	data.Created = types.StringValue(client.Created)
	data.Modified = types.StringValue(client.Modified)
}

func oauthClientStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

// oauthClientPatchOps returns JSON Patch operations for the patchable attributes that changed.
// type, internal and scope cannot be patched and force replacement instead.
func oauthClientPatchOps(ctx context.Context, plan, state OauthClientModel, diags *diag.Diagnostics) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	b.replaceOrRemoveIfChanged(plan.BusinessName, state.BusinessName, "/businessName", plan.BusinessName.ValueString())
	b.replaceOrRemoveIfChanged(plan.HomepageURL, state.HomepageURL, "/homepageUrl", plan.HomepageURL.ValueString())
	b.replaceIfChanged(plan.AccessTokenValiditySeconds, state.AccessTokenValiditySeconds, "/accessTokenValiditySeconds", plan.AccessTokenValiditySeconds.ValueInt64())
	if !plan.RefreshTokenValiditySeconds.IsNull() {
		b.replaceIfChanged(plan.RefreshTokenValiditySeconds, state.RefreshTokenValiditySeconds, "/refreshTokenValiditySeconds", plan.RefreshTokenValiditySeconds.ValueInt64())
	}
	redirectURIs := oauthClientStringSetValue(ctx, plan.RedirectURIs, diags)
	if redirectURIs == nil {
		redirectURIs = []string{}
	}
	b.replaceIfChanged(plan.RedirectURIs, state.RedirectURIs, "/redirectUris", redirectURIs)
	b.replaceIfChanged(plan.GrantTypes, state.GrantTypes, "/grantTypes", oauthClientStringSetValue(ctx, plan.GrantTypes, diags))
	b.replaceIfChanged(plan.AccessType, state.AccessType, "/accessType", plan.AccessType.ValueString())
	b.replaceIfChanged(plan.Enabled, state.Enabled, "/enabled", plan.Enabled.ValueBool())
	b.replaceIfChanged(plan.StrongAuthSupported, state.StrongAuthSupported, "/strongAuthSupported", plan.StrongAuthSupported.ValueBool())
	b.replaceIfChanged(plan.ClaimsSupported, state.ClaimsSupported, "/claimsSupported", plan.ClaimsSupported.ValueBool())
	return b.ops
}

func (r *OauthClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data OauthClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request := oauthClientFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateOauthClient(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create OAuth client: %s", err))
		return
	}
	oauthClientResolveUnknowns(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OauthClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data OauthClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	oauthClient, err := client.GetOauthClient(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read OAuth client: %s", err))
		return
	}
	setOauthClientState(ctx, &data, oauthClient, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OauthClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state OauthClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops := oauthClientPatchOps(ctx, data, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	var updated *OauthClient
	if len(ops) == 0 {
		updated = &OauthClient{ID: state.ID.ValueString(), Modified: state.Modified.ValueString()}
	} else {
		client, err := r.client.IdentityNowClient(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		updated, err = client.PatchOauthClient(ctx, data.ID.ValueString(), ops)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update OAuth client: %s", err))
			return
		}
	}
	oauthClientResolveUnknowns(ctx, &data, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *OauthClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data OauthClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteOauthClient(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete OAuth client: %s", err))
	}
}

func (r *OauthClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &OauthClientDataSource{}

func NewOauthClientDataSource() datasource.DataSource {
	return &OauthClientDataSource{}
}

type OauthClientDataSource struct {
	client *Config
}

type OauthClientDataSourceModel struct {
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	Description                 types.String `tfsdk:"description"`
	BusinessName                types.String `tfsdk:"business_name"`
	HomepageURL                 types.String `tfsdk:"homepage_url"`
	AccessTokenValiditySeconds  types.Int64  `tfsdk:"access_token_validity_seconds"`
	RefreshTokenValiditySeconds types.Int64  `tfsdk:"refresh_token_validity_seconds"`
	RedirectURIs                types.Set    `tfsdk:"redirect_uris"`
	GrantTypes                  types.Set    `tfsdk:"grant_types"`
	AccessType                  types.String `tfsdk:"access_type"`
	Type                        types.String `tfsdk:"type"`
	Internal                    types.Bool   `tfsdk:"internal"`
	Enabled                     types.Bool   `tfsdk:"enabled"`
	StrongAuthSupported         types.Bool   `tfsdk:"strong_auth_supported"`
	ClaimsSupported             types.Bool   `tfsdk:"claims_supported"`
	Scope                       types.Set    `tfsdk:"scope"`
	Created                     types.String `tfsdk:"created"`
	Modified                    types.String `tfsdk:"modified"`
	LastUsed                    types.String `tfsdk:"last_used"`
}

func (d *OauthClientDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oauth_client"
}

func (d *OauthClientDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	computedString := func(description string) dsschema.StringAttribute {
		return dsschema.StringAttribute{MarkdownDescription: description, Computed: true}
	}
	computedBool := func(description string) dsschema.BoolAttribute {
		return dsschema.BoolAttribute{MarkdownDescription: description, Computed: true}
	}
	computedSet := func(description string) dsschema.SetAttribute {
		return dsschema.SetAttribute{MarkdownDescription: description, Computed: true, ElementType: types.StringType}
	}
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up an OAuth (API) client by ID. The client secret is not available.",
		Attributes: map[string]dsschema.Attribute{
			"id": dsschema.StringAttribute{
				MarkdownDescription: "OAuth client ID.",
				Required:            true,
			},
			"name":                           computedString("Human-readable name of the API client."),
			"description":                    computedString("Description of the API client."),
			"business_name":                  computedString("Name of the business the API client belongs to."),
			"homepage_url":                   computedString("Homepage URL associated with the owner of the API client."),
			"access_token_validity_seconds":  dsschema.Int64Attribute{MarkdownDescription: "Number of seconds an access token is valid for.", Computed: true},
			"refresh_token_validity_seconds": dsschema.Int64Attribute{MarkdownDescription: "Number of seconds a refresh token is valid for.", Computed: true},
			"redirect_uris":                  computedSet("Approved redirect URIs."),
			"grant_types":                    computedSet("OAuth 2.0 grant types the client can be used with."),
			"access_type":                    computedString("Access type, `ONLINE` or `OFFLINE`."),
			"type":                           computedString("Client type, `CONFIDENTIAL` or `PUBLIC`."),
			"internal":                       computedBool("Whether the API client can be used for requests internal to IdentityNow."),
			"enabled":                        computedBool("Whether the API client is enabled."),
			"strong_auth_supported":          computedBool("Whether the API client supports strong authentication."),
			"claims_supported":               computedBool("Whether the API client supports the serialization of SAML claims."),
			"scope":                          computedSet("Scopes of the API client."),
			"created":                        computedString("Creation date."),
			"modified":                       computedString("Last modification date."),
			"last_used":                      computedString("Date the client was last used to generate an access token, updated once a day."),
		},
	}
}

func (d *OauthClientDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *OauthClientDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config OauthClientDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	oauthClient, err := client.GetOauthClient(ctx, config.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("id"), "OAuth client not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read OAuth client: %s", err))
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, oauthClientDataSourceState(ctx, oauthClient, &resp.Diagnostics))...)
}

// oauthClientDataSourceState maps an API client onto the data source model.
func oauthClientDataSourceState(ctx context.Context, client *OauthClient, diags *diag.Diagnostics) *OauthClientDataSourceModel {
	var model OauthClientModel
	setOauthClientState(ctx, &model, client, diags)
	lastUsed := types.StringNull()
	if client.LastUsed != nil {
		lastUsed = types.StringValue(*client.LastUsed)
	}
	return &OauthClientDataSourceModel{
		ID:                          model.ID,
		Name:                        model.Name,
		Description:                 model.Description,
		BusinessName:                stringValueOrNull(oauthClientStringValue(client.BusinessName)),
		HomepageURL:                 stringValueOrNull(oauthClientStringValue(client.HomepageURL)),
		AccessTokenValiditySeconds:  model.AccessTokenValiditySeconds,
		RefreshTokenValiditySeconds: model.RefreshTokenValiditySeconds,
		RedirectURIs:                oauthClientStringSetState(ctx, types.SetNull(types.StringType), client.RedirectURIs, false, diags),
		GrantTypes:                  model.GrantTypes,
		AccessType:                  model.AccessType,
		Type:                        model.Type,
		Internal:                    model.Internal,
		Enabled:                     model.Enabled,
		StrongAuthSupported:         model.StrongAuthSupported,
		ClaimsSupported:             model.ClaimsSupported,
		Scope:                       model.Scope,
		Created:                     model.Created,
		Modified:                    model.Modified,
		LastUsed:                    lastUsed,
	}
}
