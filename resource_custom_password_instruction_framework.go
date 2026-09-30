package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewCustomPasswordInstructionResource)
	registerDataSource(NewCustomPasswordInstructionDataSource)
}

// customPasswordInstructionDefaultLocale is the locale used when no locale is given.
const customPasswordInstructionDefaultLocale = "default"

// customPasswordInstructionMaxLength is the maximum length of the page content.
const customPasswordInstructionMaxLength = 1000

// customPasswordInstructionPageIDs are the pages that support custom instructions.
var customPasswordInstructionPageIDs = []string{
	"change-password:enter-password",
	"change-password:finish",
	"flow-selection:select",
	"forget-username:user-email",
	"mfa:enter-code",
	"mfa:enter-kba",
	"mfa:select",
	"reset-password:enter-password",
	"reset-password:enter-username",
	"reset-password:finish",
	"unlock-account:enter-username",
	"unlock-account:finish",
}

// CustomPasswordInstruction is custom password instructions as returned by the experimental
// /v2026/custom-password-instructions API.
type CustomPasswordInstruction struct {
	PageID      string `json:"pageId"`
	PageContent string `json:"pageContent"`
	Locale      string `json:"locale,omitempty"`
}

// customPasswordInstructionPath returns the path of the instructions of a page and locale.
func customPasswordInstructionPath(pageID, locale string) string {
	requestPath := apiPath("/v2026/custom-password-instructions/%s", pageID)
	if locale != "" {
		requestPath += "?" + url.Values{"locale": {locale}}.Encode()
	}
	return requestPath
}

func (c *Client) GetCustomPasswordInstruction(ctx context.Context, pageID, locale string) (*CustomPasswordInstruction, error) {
	var instruction CustomPasswordInstruction
	if err := c.doJSON(ctx, http.MethodGet, customPasswordInstructionPath(pageID, locale), nil, &instruction, withExperimental()); err != nil {
		return nil, err
	}
	return &instruction, nil
}

func (c *Client) CreateCustomPasswordInstruction(ctx context.Context, instruction *CustomPasswordInstruction) (*CustomPasswordInstruction, error) {
	var created CustomPasswordInstruction
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/custom-password-instructions", instruction, &created, withExperimental()); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) DeleteCustomPasswordInstruction(ctx context.Context, pageID, locale string) error {
	return c.doJSON(ctx, http.MethodDelete, customPasswordInstructionPath(pageID, locale), nil, nil, withExperimental())
}

// customPasswordInstructionPageIDValidator checks that the page ID is one of the supported pages.
type customPasswordInstructionPageIDValidator struct{}

func (v customPasswordInstructionPageIDValidator) Description(ctx context.Context) string {
	return "value must be one of: " + strings.Join(customPasswordInstructionPageIDs, ", ")
}

func (v customPasswordInstructionPageIDValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v customPasswordInstructionPageIDValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	for _, pageID := range customPasswordInstructionPageIDs {
		if req.ConfigValue.ValueString() == pageID {
			return
		}
	}
	resp.Diagnostics.AddAttributeError(req.Path, "Invalid page ID", fmt.Sprintf("%s, got %q.", v.Description(ctx), req.ConfigValue.ValueString()))
}

// customPasswordInstructionLengthValidator checks the maximum length of the page content.
type customPasswordInstructionLengthValidator struct{}

func (v customPasswordInstructionLengthValidator) Description(ctx context.Context) string {
	return fmt.Sprintf("value must be at most %d characters long", customPasswordInstructionMaxLength)
}

func (v customPasswordInstructionLengthValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v customPasswordInstructionLengthValidator) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if length := len([]rune(req.ConfigValue.ValueString())); length > customPasswordInstructionMaxLength {
		resp.Diagnostics.AddAttributeError(req.Path, "Page content too long", fmt.Sprintf("%s, got %d.", v.Description(ctx), length))
	}
}

var _ resource.Resource = &CustomPasswordInstructionResource{}
var _ resource.ResourceWithImportState = &CustomPasswordInstructionResource{}

func NewCustomPasswordInstructionResource() resource.Resource {
	return &CustomPasswordInstructionResource{}
}

type CustomPasswordInstructionResource struct {
	client *Config
}

type CustomPasswordInstructionModel struct {
	ID          types.String `tfsdk:"id"`
	PageID      types.String `tfsdk:"page_id"`
	Locale      types.String `tfsdk:"locale"`
	PageContent types.String `tfsdk:"page_content"`
}

func (r *CustomPasswordInstructionResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_password_instruction"
}

func (r *CustomPasswordInstructionResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages custom instructions shown on a page of the password reset, change password, unlock account and forgotten username flows. The API has no update, so every change forces new instructions. There is one set of instructions per page and locale: creating the resource takes over existing instructions of the page and locale, and `create_before_destroy` must not be used. Changes of the content made outside Terraform are not detected. Uses an experimental API.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Identifier in the form `<page_id>/<locale>`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"page_id": schema.StringAttribute{
				MarkdownDescription: "Page the instructions are shown on, e.g. `reset-password:enter-password`. Changing it forces new instructions.",
				Required:            true,
				Validators:          []validator.String{customPasswordInstructionPageIDValidator{}},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"locale": schema.StringAttribute{
				MarkdownDescription: "BCP 47 language tag of the instructions. Defaults to `default`. Changing it forces new instructions.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(customPasswordInstructionDefaultLocale),
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"page_content": schema.StringAttribute{
				MarkdownDescription: "Instructions in basic HTML, at most 1000 characters. IdentityNow sanitizes the content; the configured content is kept in state, so changes made outside Terraform are not detected. Changing it forces new instructions.",
				Required:            true,
				Validators:          []validator.String{customPasswordInstructionLengthValidator{}},
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
		},
	}
}

func (r *CustomPasswordInstructionResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// customPasswordInstructionID returns the resource ID of a page and locale.
func customPasswordInstructionID(pageID, locale string) string {
	return pageID + "/" + locale
}

// setCustomPasswordInstructionState refreshes the model from the API. The locale is part of the
// request and keeps its state value.
func setCustomPasswordInstructionState(data *CustomPasswordInstructionModel, instruction *CustomPasswordInstruction) {
	if instruction.PageID != "" {
		data.PageID = types.StringValue(instruction.PageID)
	}
	data.PageContent = types.StringValue(instruction.PageContent)
	data.ID = types.StringValue(customPasswordInstructionID(data.PageID.ValueString(), data.Locale.ValueString()))
}

// setCustomPasswordInstructionResourceState refreshes the resource model from the API. IdentityNow
// sanitizes the page content, so the returned content can differ from the configured content even
// though nothing changed. Since page_content forces replacement, taking the API value would replace
// the instructions on every apply; the prior content is kept instead and the API value is only used
// when there is no prior value (import). Changes of the content made outside Terraform are not detected.
func setCustomPasswordInstructionResourceState(data *CustomPasswordInstructionModel, instruction *CustomPasswordInstruction) {
	prior := data.PageContent
	setCustomPasswordInstructionState(data, instruction)
	if !prior.IsNull() && !prior.IsUnknown() && prior.ValueString() != "" {
		data.PageContent = prior
	}
}

func (r *CustomPasswordInstructionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CustomPasswordInstructionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if _, err := client.CreateCustomPasswordInstruction(ctx, &CustomPasswordInstruction{
		PageID:      data.PageID.ValueString(),
		PageContent: data.PageContent.ValueString(),
		Locale:      data.Locale.ValueString(),
	}); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create custom password instructions: %s", err))
		return
	}
	data.ID = types.StringValue(customPasswordInstructionID(data.PageID.ValueString(), data.Locale.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomPasswordInstructionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CustomPasswordInstructionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	instruction, err := client.GetCustomPasswordInstruction(ctx, data.PageID.ValueString(), data.Locale.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read custom password instructions: %s", err))
		return
	}
	setCustomPasswordInstructionResourceState(&data, instruction)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update is never called with changes, since every configurable attribute forces replacement.
func (r *CustomPasswordInstructionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CustomPasswordInstructionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *CustomPasswordInstructionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CustomPasswordInstructionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteCustomPasswordInstruction(ctx, data.PageID.ValueString(), data.Locale.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete custom password instructions: %s", err))
	}
}

// ImportState accepts `<page_id>` for the default locale or `<page_id>/<locale>`.
func (r *CustomPasswordInstructionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	pageID, locale, found := strings.Cut(req.ID, "/")
	if !found {
		locale = customPasswordInstructionDefaultLocale
	}
	if pageID == "" || locale == "" {
		resp.Diagnostics.AddError("Invalid import ID", fmt.Sprintf("Expected <page_id> or <page_id>/<locale>, got %q.", req.ID))
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), customPasswordInstructionID(pageID, locale))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("page_id"), pageID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("locale"), locale)...)
}

var _ datasource.DataSource = &CustomPasswordInstructionDataSource{}

func NewCustomPasswordInstructionDataSource() datasource.DataSource {
	return &CustomPasswordInstructionDataSource{}
}

type CustomPasswordInstructionDataSource struct {
	client *Config
}

func (d *CustomPasswordInstructionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_custom_password_instruction"
}

func (d *CustomPasswordInstructionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Reads the custom password instructions of a page. Uses an experimental API.",
		Attributes: map[string]dsschema.Attribute{
			"id": dsschema.StringAttribute{
				MarkdownDescription: "Identifier in the form `<page_id>/<locale>`.",
				Computed:            true,
			},
			"page_id": dsschema.StringAttribute{
				MarkdownDescription: "Page the instructions are shown on, e.g. `reset-password:enter-password`.",
				Required:            true,
				Validators:          []validator.String{customPasswordInstructionPageIDValidator{}},
			},
			"locale": dsschema.StringAttribute{
				MarkdownDescription: "BCP 47 language tag of the instructions. Defaults to `default`.",
				Optional:            true,
				Computed:            true,
			},
			"page_content": dsschema.StringAttribute{
				MarkdownDescription: "Instructions in basic HTML.",
				Computed:            true,
			},
		},
	}
}

func (d *CustomPasswordInstructionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *CustomPasswordInstructionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CustomPasswordInstructionModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if data.Locale.IsNull() {
		data.Locale = types.StringValue(customPasswordInstructionDefaultLocale)
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	instruction, err := client.GetCustomPasswordInstruction(ctx, data.PageID.ValueString(), data.Locale.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("page_id"), "Custom password instructions not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read custom password instructions: %s", err))
		return
	}
	setCustomPasswordInstructionState(&data, instruction)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
