package provider

import (
	"bytes"
	"context"
	"fmt"
	"mime/multipart"
	"net/http"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

func init() {
	registerResource(NewBrandingResource)
	registerDataSource(NewBrandingDataSource)
}

// Branding is a branding item as returned by the /v2026/brandings API.
type Branding struct {
	Name                      string  `json:"name"`
	ProductName               *string `json:"productName,omitempty"`
	ActionButtonColor         *string `json:"actionButtonColor,omitempty"`
	ActiveLinkColor           *string `json:"activeLinkColor,omitempty"`
	NavigationColor           *string `json:"navigationColor,omitempty"`
	EmailFromAddress          *string `json:"emailFromAddress,omitempty"`
	StandardLogoURL           *string `json:"standardLogoURL,omitempty"`
	LoginInformationalMessage *string `json:"loginInformationalMessage,omitempty"`
}

// brandingMultipartFields returns the form fields of a create or update request. Nil values are
// not sent. The logo file (fileStandard) is not managed.
func brandingMultipartFields(branding *Branding) map[string]string {
	fields := map[string]string{"name": branding.Name}
	for key, value := range map[string]*string{
		"productName":               branding.ProductName,
		"actionButtonColor":         branding.ActionButtonColor,
		"activeLinkColor":           branding.ActiveLinkColor,
		"navigationColor":           branding.NavigationColor,
		"emailFromAddress":          branding.EmailFromAddress,
		"loginInformationalMessage": branding.LoginInformationalMessage,
	} {
		if value != nil {
			fields[key] = *value
		}
	}
	return fields
}

// brandingMultipartBody encodes form fields as a multipart/form-data body, in sorted key order.
func brandingMultipartBody(fields map[string]string) ([]byte, string, error) {
	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, key := range keys {
		if err := writer.WriteField(key, fields[key]); err != nil {
			return nil, "", err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return body.Bytes(), writer.FormDataContentType(), nil
}

// brandingMultipartRequest sends branding as a multipart/form-data request and decodes the JSON response.
func (c *Client) brandingMultipartRequest(ctx context.Context, method, requestPath string, branding *Branding) (*Branding, error) {
	body, contentType, err := brandingMultipartBody(brandingMultipartFields(branding))
	if err != nil {
		return nil, fmt.Errorf("encoding request body for %s %s: %w", method, requestPath, err)
	}
	req, err := http.NewRequest(method, c.BaseURL+requestPath, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", contentType)
	tflog.Debug(ctx, "Sending API request", map[string]interface{}{"method": method, "url": req.URL.String()})
	var result Branding
	if err := c.sendRequest(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) GetBranding(ctx context.Context, name string) (*Branding, error) {
	var branding Branding
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/brandings/%s", name), nil, &branding); err != nil {
		return nil, err
	}
	return &branding, nil
}

func (c *Client) CreateBranding(ctx context.Context, branding *Branding) (*Branding, error) {
	return c.brandingMultipartRequest(ctx, http.MethodPost, "/v2026/brandings", branding)
}

func (c *Client) UpdateBranding(ctx context.Context, name string, branding *Branding) (*Branding, error) {
	return c.brandingMultipartRequest(ctx, http.MethodPut, apiPath("/v2026/brandings/%s", name), branding)
}

func (c *Client) DeleteBranding(ctx context.Context, name string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/brandings/%s", name), nil, nil)
}

var _ resource.Resource = &BrandingResource{}
var _ resource.ResourceWithImportState = &BrandingResource{}

func NewBrandingResource() resource.Resource {
	return &BrandingResource{}
}

type BrandingResource struct {
	client *Config
}

type BrandingModel struct {
	ID                        types.String `tfsdk:"id"`
	Name                      types.String `tfsdk:"name"`
	ProductName               types.String `tfsdk:"product_name"`
	ActionButtonColor         types.String `tfsdk:"action_button_color"`
	ActiveLinkColor           types.String `tfsdk:"active_link_color"`
	NavigationColor           types.String `tfsdk:"navigation_color"`
	EmailFromAddress          types.String `tfsdk:"email_from_address"`
	LoginInformationalMessage types.String `tfsdk:"login_informational_message"`
	StandardLogoURL           types.String `tfsdk:"standard_logo_url"`
}

func (r *BrandingResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branding"
}

func (r *BrandingResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// The API documents no defaults for these values, so they are plain optional attributes: an unset
	// attribute is null in state, and removing it clears the value.
	optional := func(description string) schema.StringAttribute {
		return schema.StringAttribute{
			MarkdownDescription: description + " Removing it clears the value.",
			Optional:            true,
		}
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a branding item (product name, colors, email sender and login message). The logo is not managed: upload it in the IdentityNow UI. Updates do not send a logo; if IdentityNow treats a missing logo as a removal, an update clears the uploaded logo.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Name of the branding item.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the branding item. Changing it forces a new branding item to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"product_name": schema.StringAttribute{
				MarkdownDescription: "Product name shown in the UI and in emails.",
				Required:            true,
			},
			"action_button_color":         optional("Hex color of action buttons, e.g. `0074D9`."),
			"active_link_color":           optional("Hex color of links."),
			"navigation_color":            optional("Hex color of the navigation bar."),
			"email_from_address":          optional("Sender address of emails."),
			"login_informational_message": optional("Informational message shown on the login page."),
			"standard_logo_url": schema.StringAttribute{
				MarkdownDescription: "URL of the standard logo. The logo is not managed by this resource, see the note above.",
				Computed:            true,
			},
		},
	}
}

func (r *BrandingResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// brandingFromModel builds the request. Unset optional values are not sent. With prior (update), a
// value that is removed from the configuration (null in the plan, set in the prior state) is sent
// as an empty string, so the update clears it instead of leaving it unchanged.
func brandingFromModel(data BrandingModel, prior *BrandingModel) *Branding {
	value := func(planned types.String, priorValue func(BrandingModel) types.String) *string {
		if planned.IsNull() && prior != nil {
			if p := priorValue(*prior); !p.IsNull() && !p.IsUnknown() && p.ValueString() != "" {
				empty := ""
				return &empty
			}
		}
		return stringPointer(planned)
	}
	return &Branding{
		Name:                      data.Name.ValueString(),
		ProductName:               stringPointer(data.ProductName),
		ActionButtonColor:         value(data.ActionButtonColor, func(m BrandingModel) types.String { return m.ActionButtonColor }),
		ActiveLinkColor:           value(data.ActiveLinkColor, func(m BrandingModel) types.String { return m.ActiveLinkColor }),
		NavigationColor:           value(data.NavigationColor, func(m BrandingModel) types.String { return m.NavigationColor }),
		EmailFromAddress:          value(data.EmailFromAddress, func(m BrandingModel) types.String { return m.EmailFromAddress }),
		LoginInformationalMessage: value(data.LoginInformationalMessage, func(m BrandingModel) types.String { return m.LoginInformationalMessage }),
	}
}

// brandingStringState returns the state value of a nullable API string.
func brandingStringState(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// brandingResolveComputed keeps the planned values and resolves unknown values from the API.
func brandingResolveComputed(data *BrandingModel, branding *Branding) {
	data.ID = data.Name
	resolve := func(planned types.String, value *string) types.String {
		if planned.IsUnknown() {
			return brandingStringState(value)
		}
		return planned
	}
	data.ActionButtonColor = resolve(data.ActionButtonColor, branding.ActionButtonColor)
	data.ActiveLinkColor = resolve(data.ActiveLinkColor, branding.ActiveLinkColor)
	data.NavigationColor = resolve(data.NavigationColor, branding.NavigationColor)
	data.EmailFromAddress = resolve(data.EmailFromAddress, branding.EmailFromAddress)
	data.LoginInformationalMessage = resolve(data.LoginInformationalMessage, branding.LoginInformationalMessage)
	data.StandardLogoURL = brandingStringState(branding.StandardLogoURL)
}

// setBrandingState refreshes the model from the API.
func setBrandingState(data *BrandingModel, branding *Branding) {
	name := branding.Name
	if name == "" {
		name = data.ID.ValueString()
	}
	data.Name = types.StringValue(name)
	data.ID = data.Name
	data.ProductName = optionalStringState(data.ProductName, oauthClientStringValue(branding.ProductName))
	data.ActionButtonColor = optionalStringState(data.ActionButtonColor, oauthClientStringValue(branding.ActionButtonColor))
	data.ActiveLinkColor = optionalStringState(data.ActiveLinkColor, oauthClientStringValue(branding.ActiveLinkColor))
	data.NavigationColor = optionalStringState(data.NavigationColor, oauthClientStringValue(branding.NavigationColor))
	data.EmailFromAddress = optionalStringState(data.EmailFromAddress, oauthClientStringValue(branding.EmailFromAddress))
	data.LoginInformationalMessage = optionalStringState(data.LoginInformationalMessage, oauthClientStringValue(branding.LoginInformationalMessage))
	data.StandardLogoURL = brandingStringState(branding.StandardLogoURL)
}

func (r *BrandingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data BrandingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateBranding(ctx, brandingFromModel(data, nil))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create branding item: %s", err))
		return
	}
	brandingResolveComputed(&data, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrandingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data BrandingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	branding, err := client.GetBranding(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read branding item: %s", err))
		return
	}
	setBrandingState(&data, branding)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrandingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, prior BrandingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &prior)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// The logo (fileStandard) is not sent, see the resource description.
	updated, err := client.UpdateBranding(ctx, data.ID.ValueString(), brandingFromModel(data, &prior))
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update branding item: %s", err))
		return
	}
	brandingResolveComputed(&data, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *BrandingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data BrandingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteBranding(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete branding item: %s", err))
	}
}

func (r *BrandingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &BrandingDataSource{}

func NewBrandingDataSource() datasource.DataSource {
	return &BrandingDataSource{}
}

type BrandingDataSource struct {
	client *Config
}

func (d *BrandingDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branding"
}

func (d *BrandingDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	computed := func(description string) dsschema.StringAttribute {
		return dsschema.StringAttribute{MarkdownDescription: description, Computed: true}
	}
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a branding item by name.",
		Attributes: map[string]dsschema.Attribute{
			"name": dsschema.StringAttribute{
				MarkdownDescription: "Name of the branding item, e.g. `default`.",
				Required:            true,
			},
			"id":                          computed("Name of the branding item."),
			"product_name":                computed("Product name."),
			"action_button_color":         computed("Hex color of action buttons."),
			"active_link_color":           computed("Hex color of links."),
			"navigation_color":            computed("Hex color of the navigation bar."),
			"email_from_address":          computed("Sender address of emails."),
			"login_informational_message": computed("Informational message shown on the login page."),
			"standard_logo_url":           computed("URL of the standard logo."),
		},
	}
}

func (d *BrandingDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *BrandingDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BrandingModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	branding, err := client.GetBranding(ctx, data.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Branding item not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read branding item: %s", err))
		return
	}
	data.ProductName = types.StringNull()
	setBrandingState(&data, branding)
	data.ProductName = brandingStringState(branding.ProductName)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
