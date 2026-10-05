package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewPersonalAccessTokenResource)
	registerDataSource(NewPersonalAccessTokenDataSource)
}

// PersonalAccessToken is a personal access token as returned by the /v2026/personal-access-tokens
// API. Secret is only returned by the create call.
type PersonalAccessToken struct {
	ID                         string                    `json:"id,omitempty"`
	Secret                     string                    `json:"secret,omitempty"`
	Name                       string                    `json:"name"`
	Scope                      []string                  `json:"scope,omitempty"`
	Owner                      *PersonalAccessTokenOwner `json:"owner,omitempty"`
	Created                    string                    `json:"created,omitempty"`
	LastUsed                   *string                   `json:"lastUsed,omitempty"`
	Managed                    *bool                     `json:"managed,omitempty"`
	AccessTokenValiditySeconds *int64                    `json:"accessTokenValiditySeconds,omitempty"`
	ExpirationDate             *string                   `json:"expirationDate,omitempty"`
	UserAwareTokenNeverExpires *bool                     `json:"userAwareTokenNeverExpires,omitempty"`
}

// PersonalAccessTokenOwner is the identity that owns a personal access token.
type PersonalAccessTokenOwner struct {
	Type string `json:"type,omitempty"`
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// personalAccessTokenNull is a JSON Patch value that is sent as an explicit null.
var personalAccessTokenNull = json.RawMessage("null")

// ListPersonalAccessTokens lists the tokens of ownerID. "me" lists the tokens of the caller, an empty
// owner lists all tokens of the tenant, which requires the idn:all-personal-access-tokens:read right.
// The endpoint is not paginated.
func (c *Client) ListPersonalAccessTokens(ctx context.Context, ownerID string) ([]PersonalAccessToken, error) {
	requestPath := "/v2026/personal-access-tokens"
	if ownerID != "" {
		requestPath += "?" + url.Values{"owner-id": {ownerID}}.Encode()
	}
	var tokens []PersonalAccessToken
	if err := c.doJSON(ctx, http.MethodGet, requestPath, nil, &tokens); err != nil {
		return nil, err
	}
	return tokens, nil
}

// GetPersonalAccessToken finds a token by ID. There is no endpoint to get a single token, so the
// tokens of the caller are searched first and then all tokens of the tenant. Listing all tokens
// needs an extra right, so a 403 from that list counts as not found.
func (c *Client) GetPersonalAccessToken(ctx context.Context, id string) (*PersonalAccessToken, error) {
	for _, owner := range []string{"me", ""} {
		tokens, err := c.ListPersonalAccessTokens(ctx, owner)
		var apiErr *APIError
		if owner == "" && errors.As(err, &apiErr) && apiErr.StatusCode == http.StatusForbidden {
			break
		}
		if err != nil {
			return nil, err
		}
		for i := range tokens {
			if tokens[i].ID == id {
				return &tokens[i], nil
			}
		}
	}
	return nil, &NotFoundError{fmt.Sprintf("personal access token %q not found", id)}
}

// GetPersonalAccessTokenByName finds a token of ownerID ("me" for the caller) by exact name.
func (c *Client) GetPersonalAccessTokenByName(ctx context.Context, ownerID, name string) (*PersonalAccessToken, error) {
	tokens, err := c.ListPersonalAccessTokens(ctx, ownerID)
	if err != nil {
		return nil, err
	}
	var match *PersonalAccessToken
	for i := range tokens {
		if tokens[i].Name != name {
			continue
		}
		if match != nil {
			return nil, fmt.Errorf("multiple personal access tokens are named %q, set owner_id", name)
		}
		match = &tokens[i]
	}
	if match == nil {
		return nil, &NotFoundError{fmt.Sprintf("personal access token with name %q not found", name)}
	}
	return match, nil
}

func (c *Client) CreatePersonalAccessToken(ctx context.Context, token *PersonalAccessToken) (*PersonalAccessToken, error) {
	var created PersonalAccessToken
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/personal-access-tokens", token, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchPersonalAccessToken(ctx context.Context, id string, ops []jsonPatchOp) (*PersonalAccessToken, error) {
	var updated PersonalAccessToken
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/personal-access-tokens/%s", id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeletePersonalAccessToken(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/personal-access-tokens/%s", id), nil, nil)
}

var _ resource.Resource = &PersonalAccessTokenResource{}
var _ resource.ResourceWithImportState = &PersonalAccessTokenResource{}
var _ resource.ResourceWithValidateConfig = &PersonalAccessTokenResource{}

func NewPersonalAccessTokenResource() resource.Resource {
	return &PersonalAccessTokenResource{}
}

type PersonalAccessTokenResource struct {
	client *Config
}

type PersonalAccessTokenModel struct {
	ID                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	Scope                      types.Set    `tfsdk:"scope"`
	AccessTokenValiditySeconds types.Int64  `tfsdk:"access_token_validity_seconds"`
	ExpirationDate             types.String `tfsdk:"expiration_date"`
	UserAwareTokenNeverExpires types.Bool   `tfsdk:"user_aware_token_never_expires"`
	Owner                      types.List   `tfsdk:"owner"`
	Managed                    types.Bool   `tfsdk:"managed"`
	Secret                     types.String `tfsdk:"secret"`
	Created                    types.String `tfsdk:"created"`
}

func (r *PersonalAccessTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_personal_access_token"
}

// personalAccessTokenOwnerAttributes are the attributes of the computed owner reference.
func personalAccessTokenOwnerAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"id":   schema.StringAttribute{MarkdownDescription: "Owner identity ID.", Computed: true},
		"type": schema.StringAttribute{MarkdownDescription: "Owner type, `IDENTITY`.", Computed: true},
		"name": schema.StringAttribute{MarkdownDescription: "Owner display name.", Computed: true},
	}
}

func (r *PersonalAccessTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a personal access token of the identity the provider authenticates as. The secret is only returned when the token is created and is stored in state as a sensitive value.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Token ID, used as the client ID when requesting access tokens.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Token name, unique among the tokens of the owner.",
				Required:            true,
			},
			"scope": schema.SetAttribute{
				MarkdownDescription: "Scopes of the token. Defaults to `sp:scopes:all`, all rights of the owner. Scope changes apply to access tokens generated after the change.",
				Optional:            true,
				Computed:            true,
				ElementType:         types.StringType,
				PlanModifiers:       []planmodifier.Set{setplanmodifier.UseStateForUnknown()},
			},
			"access_token_validity_seconds": schema.Int64Attribute{
				MarkdownDescription: "Number of seconds an access token generated with this token is valid, between 15 and 43200 (the default). Cannot be updated, changing it forces a new token.",
				Optional:            true,
				Computed:            true,
				Validators:          []validator.Int64{int64AtLeastValidator{min: 15}},
				PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown(), int64planmodifier.RequiresReplace()},
			},
			"expiration_date": schema.StringAttribute{
				MarkdownDescription: "Date and time (RFC 3339) when the token expires, must be in the future. When not set the token never expires, which requires `user_aware_token_never_expires = true`.",
				Optional:            true,
			},
			"user_aware_token_never_expires": schema.BoolAttribute{
				MarkdownDescription: "Acknowledges the security implications of a token that never expires. Must be `true` when `expiration_date` is not set.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"owner": schema.ListNestedAttribute{
				MarkdownDescription: "Identity that owns the token.",
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject:        schema.NestedAttributeObject{Attributes: personalAccessTokenOwnerAttributes()},
			},
			"managed": schema.BoolAttribute{
				MarkdownDescription: "Whether the token is managed by the SailPoint platform, for example by workflows.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"secret": schema.StringAttribute{
				MarkdownDescription: "Token secret, used as the client secret. It is only returned when the token is created, so it is null for imported tokens.",
				Computed:            true,
				Sensitive:           true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Creation date.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *PersonalAccessTokenResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var expiration types.String
	var neverExpires types.Bool
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("expiration_date"), &expiration)...)
	resp.Diagnostics.Append(req.Config.GetAttribute(ctx, path.Root("user_aware_token_never_expires"), &neverExpires)...)
	if resp.Diagnostics.HasError() || expiration.IsUnknown() || neverExpires.IsUnknown() {
		return
	}
	if expiration.IsNull() && !neverExpires.ValueBool() {
		resp.Diagnostics.AddAttributeError(path.Root("user_aware_token_never_expires"), "Token without expiration",
			"A token without expiration_date never expires. Set expiration_date, or set user_aware_token_never_expires = true to acknowledge a token that never expires.")
	}
	if !expiration.IsNull() {
		if _, err := time.Parse(time.RFC3339, expiration.ValueString()); err != nil {
			resp.Diagnostics.AddAttributeError(path.Root("expiration_date"), "Invalid expiration date", fmt.Sprintf("expiration_date must be an RFC 3339 date and time: %s", err))
		}
	}
}

func (r *PersonalAccessTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// personalAccessTokenOwnerState converts the token owner to a single-element list, or null.
func personalAccessTokenOwnerState(ctx context.Context, owner *PersonalAccessTokenOwner, diags *diag.Diagnostics) types.List {
	if owner == nil {
		return types.ListNull(objectInfoObjectType)
	}
	list, d := types.ListValueFrom(ctx, objectInfoObjectType, []OwnerModel{{
		ID: types.StringValue(owner.ID), Type: types.StringValue(owner.Type), Name: types.StringValue(owner.Name),
	}})
	diags.Append(d...)
	return list
}

// personalAccessTokenTimeState keeps a prior date and time that denotes the same instant as the API
// value, so a different precision or time zone does not cause a diff.
func personalAccessTokenTimeState(prior types.String, value *string) types.String {
	if value == nil || *value == "" {
		return types.StringNull()
	}
	if !prior.IsNull() && !prior.IsUnknown() {
		priorTime, errPrior := time.Parse(time.RFC3339, prior.ValueString())
		apiTime, errAPI := time.Parse(time.RFC3339, *value)
		if errPrior == nil && errAPI == nil && priorTime.Equal(apiTime) {
			return prior
		}
	}
	return types.StringValue(*value)
}

// personalAccessTokenFromModel builds the create request.
func personalAccessTokenFromModel(ctx context.Context, data PersonalAccessTokenModel, diags *diag.Diagnostics) *PersonalAccessToken {
	return &PersonalAccessToken{
		Name:                       data.Name.ValueString(),
		Scope:                      oauthClientStringSetValue(ctx, data.Scope, diags),
		AccessTokenValiditySeconds: int64Pointer(data.AccessTokenValiditySeconds),
		ExpirationDate:             stringPointer(data.ExpirationDate),
		UserAwareTokenNeverExpires: boolPointer(data.UserAwareTokenNeverExpires),
	}
}

// personalAccessTokenResolveUnknowns keeps the planned values and resolves unknown computed values
// from the API response after create or update.
func personalAccessTokenResolveUnknowns(ctx context.Context, data *PersonalAccessTokenModel, token *PersonalAccessToken, diags *diag.Diagnostics) {
	if data.ID.IsUnknown() {
		data.ID = types.StringValue(token.ID)
	}
	if data.Scope.IsUnknown() {
		data.Scope = oauthClientStringSetState(ctx, types.SetNull(types.StringType), token.Scope, false, diags)
	}
	if data.AccessTokenValiditySeconds.IsUnknown() {
		data.AccessTokenValiditySeconds = optionalInt64State(types.Int64Null(), token.AccessTokenValiditySeconds)
	}
	data.UserAwareTokenNeverExpires = computedBoolFromAPI(data.UserAwareTokenNeverExpires, token.UserAwareTokenNeverExpires)
	if data.Owner.IsUnknown() {
		data.Owner = personalAccessTokenOwnerState(ctx, token.Owner, diags)
	}
	data.Managed = computedBoolFromAPI(data.Managed, token.Managed)
	if data.Secret.IsUnknown() {
		// The secret is only returned by the create call.
		data.Secret = stringValueOrNull(token.Secret)
	}
	if data.Created.IsUnknown() {
		data.Created = types.StringValue(token.Created)
	}
}

// setPersonalAccessTokenState refreshes the model from the API. The secret keeps its state value.
func setPersonalAccessTokenState(ctx context.Context, data *PersonalAccessTokenModel, token *PersonalAccessToken, diags *diag.Diagnostics) {
	data.ID = types.StringValue(token.ID)
	data.Name = types.StringValue(token.Name)
	data.Scope = oauthClientStringSetState(ctx, data.Scope, token.Scope, false, diags)
	data.AccessTokenValiditySeconds = optionalInt64State(data.AccessTokenValiditySeconds, token.AccessTokenValiditySeconds)
	data.ExpirationDate = personalAccessTokenTimeState(data.ExpirationDate, token.ExpirationDate)
	data.UserAwareTokenNeverExpires = types.BoolValue(token.UserAwareTokenNeverExpires != nil && *token.UserAwareTokenNeverExpires)
	data.Owner = personalAccessTokenOwnerState(ctx, token.Owner, diags)
	data.Managed = types.BoolValue(token.Managed != nil && *token.Managed)
	data.Created = types.StringValue(token.Created)
}

// personalAccessTokenPatchOps returns JSON Patch operations for the changed patchable attributes:
// name, scope, expirationDate and userAwareTokenNeverExpires. Clearing the expiration date requires
// userAwareTokenNeverExpires in the same request, so it is always sent in that case.
func personalAccessTokenPatchOps(ctx context.Context, plan, state PersonalAccessTokenModel, diags *diag.Diagnostics) []jsonPatchOp {
	var b patchBuilder
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceIfChanged(plan.Scope, state.Scope, "/scope", oauthClientStringSetValue(ctx, plan.Scope, diags))
	expirationChanged := !plan.ExpirationDate.Equal(state.ExpirationDate)
	if expirationChanged {
		if plan.ExpirationDate.IsNull() {
			b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/expirationDate", Value: personalAccessTokenNull})
		} else {
			b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/expirationDate", Value: plan.ExpirationDate.ValueString()})
		}
	}
	if expirationChanged && plan.ExpirationDate.IsNull() && !plan.UserAwareTokenNeverExpires.IsUnknown() {
		b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/userAwareTokenNeverExpires", Value: plan.UserAwareTokenNeverExpires.ValueBool()})
	} else {
		b.replaceIfChanged(plan.UserAwareTokenNeverExpires, state.UserAwareTokenNeverExpires, "/userAwareTokenNeverExpires", plan.UserAwareTokenNeverExpires.ValueBool())
	}
	return b.ops
}

func (r *PersonalAccessTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data PersonalAccessTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	request := personalAccessTokenFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreatePersonalAccessToken(ctx, request)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create personal access token: %s", err))
		return
	}
	personalAccessTokenResolveUnknowns(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PersonalAccessTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data PersonalAccessTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	token, err := client.GetPersonalAccessToken(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read personal access token: %s", err))
		return
	}
	setPersonalAccessTokenState(ctx, &data, token, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PersonalAccessTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state PersonalAccessTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops := personalAccessTokenPatchOps(ctx, data, state, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	updated := &PersonalAccessToken{ID: state.ID.ValueString()}
	if len(ops) > 0 {
		client, err := r.client.IdentityNowClient(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		updated, err = client.PatchPersonalAccessToken(ctx, data.ID.ValueString(), ops)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update personal access token: %s", err))
			return
		}
	}
	personalAccessTokenResolveUnknowns(ctx, &data, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *PersonalAccessTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data PersonalAccessTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeletePersonalAccessToken(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete personal access token: %s", err))
	}
}

func (r *PersonalAccessTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &PersonalAccessTokenDataSource{}

func NewPersonalAccessTokenDataSource() datasource.DataSource {
	return &PersonalAccessTokenDataSource{}
}

type PersonalAccessTokenDataSource struct {
	client *Config
}

type PersonalAccessTokenDataSourceModel struct {
	Name                       types.String `tfsdk:"name"`
	OwnerID                    types.String `tfsdk:"owner_id"`
	ID                         types.String `tfsdk:"id"`
	Scope                      types.Set    `tfsdk:"scope"`
	AccessTokenValiditySeconds types.Int64  `tfsdk:"access_token_validity_seconds"`
	ExpirationDate             types.String `tfsdk:"expiration_date"`
	UserAwareTokenNeverExpires types.Bool   `tfsdk:"user_aware_token_never_expires"`
	Owner                      types.List   `tfsdk:"owner"`
	Managed                    types.Bool   `tfsdk:"managed"`
	Created                    types.String `tfsdk:"created"`
	LastUsed                   types.String `tfsdk:"last_used"`
}

func (d *PersonalAccessTokenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_personal_access_token"
}

func (d *PersonalAccessTokenDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a personal access token by name. The token secret is not available.",
		Attributes: map[string]dsschema.Attribute{
			"name": dsschema.StringAttribute{
				MarkdownDescription: "Token name.",
				Required:            true,
			},
			"owner_id": dsschema.StringAttribute{
				MarkdownDescription: "Identity ID of the token owner. Defaults to the identity the provider authenticates as. Looking up tokens of other identities requires the `idn:all-personal-access-tokens:read` right.",
				Optional:            true,
			},
			"id":                             dsschema.StringAttribute{MarkdownDescription: "Token ID.", Computed: true},
			"scope":                          dsschema.SetAttribute{MarkdownDescription: "Scopes of the token.", Computed: true, ElementType: types.StringType},
			"access_token_validity_seconds":  dsschema.Int64Attribute{MarkdownDescription: "Number of seconds an access token generated with this token is valid.", Computed: true},
			"expiration_date":                dsschema.StringAttribute{MarkdownDescription: "Expiration date, null when the token never expires.", Computed: true},
			"user_aware_token_never_expires": dsschema.BoolAttribute{MarkdownDescription: "Whether a token that never expires was acknowledged.", Computed: true},
			"owner": dsschema.ListNestedAttribute{
				MarkdownDescription: "Identity that owns the token.",
				Computed:            true,
				NestedObject: dsschema.NestedAttributeObject{Attributes: map[string]dsschema.Attribute{
					"id":   dsschema.StringAttribute{MarkdownDescription: "Owner identity ID.", Computed: true},
					"type": dsschema.StringAttribute{MarkdownDescription: "Owner type, `IDENTITY`.", Computed: true},
					"name": dsschema.StringAttribute{MarkdownDescription: "Owner display name.", Computed: true},
				}},
			},
			"managed":   dsschema.BoolAttribute{MarkdownDescription: "Whether the token is managed by the SailPoint platform.", Computed: true},
			"created":   dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
			"last_used": dsschema.StringAttribute{MarkdownDescription: "Date the token was last used to generate an access token, updated once a day.", Computed: true},
		},
	}
}

func (d *PersonalAccessTokenDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *PersonalAccessTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data PersonalAccessTokenDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	owner := "me"
	if !data.OwnerID.IsNull() {
		owner = data.OwnerID.ValueString()
	}
	token, err := client.GetPersonalAccessTokenByName(ctx, owner, data.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Personal access token not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read personal access token: %s", err))
		return
	}
	personalAccessTokenDataSourceState(ctx, &data, token, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// personalAccessTokenDataSourceState maps an API token onto the data source model.
func personalAccessTokenDataSourceState(ctx context.Context, data *PersonalAccessTokenDataSourceModel, token *PersonalAccessToken, diags *diag.Diagnostics) {
	var model PersonalAccessTokenModel
	model.Scope = types.SetNull(types.StringType)
	setPersonalAccessTokenState(ctx, &model, token, diags)
	data.ID = model.ID
	data.Name = model.Name
	data.Scope = model.Scope
	data.AccessTokenValiditySeconds = model.AccessTokenValiditySeconds
	data.ExpirationDate = model.ExpirationDate
	data.UserAwareTokenNeverExpires = model.UserAwareTokenNeverExpires
	data.Owner = model.Owner
	data.Managed = model.Managed
	data.Created = model.Created
	data.LastUsed = types.StringNull()
	if token.LastUsed != nil {
		data.LastUsed = types.StringValue(*token.LastUsed)
	}
}
