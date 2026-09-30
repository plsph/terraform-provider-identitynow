package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"reflect"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewNotificationTemplateResource)
	registerDataSource(NewNotificationTemplateDataSource)
}

// NotificationTemplate is a custom notification template as returned by the /v2026/notification-templates API.
type NotificationTemplate struct {
	ID            string                 `json:"id,omitempty"`
	Key           string                 `json:"key"`
	Name          string                 `json:"name,omitempty"`
	Medium        string                 `json:"medium"`
	Locale        string                 `json:"locale"`
	Subject       string                 `json:"subject,omitempty"`
	Header        *string                `json:"header,omitempty"`
	Body          string                 `json:"body,omitempty"`
	Footer        *string                `json:"footer,omitempty"`
	From          string                 `json:"from,omitempty"`
	ReplyTo       string                 `json:"replyTo,omitempty"`
	Description   string                 `json:"description,omitempty"`
	Created       string                 `json:"created,omitempty"`
	Modified      string                 `json:"modified,omitempty"`
	SlackTemplate map[string]interface{} `json:"slackTemplate,omitempty"`
	TeamsTemplate map[string]interface{} `json:"teamsTemplate,omitempty"`
}

// NotificationTemplateBulkDeleteItem identifies a template to delete.
type NotificationTemplateBulkDeleteItem struct {
	Key    string `json:"key"`
	Medium string `json:"medium,omitempty"`
	Locale string `json:"locale,omitempty"`
}

func (c *Client) GetNotificationTemplate(ctx context.Context, id string) (*NotificationTemplate, error) {
	var template NotificationTemplate
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/notification-templates/%s", id), nil, &template); err != nil {
		return nil, err
	}
	return &template, nil
}

// GetNotificationTemplateByKey returns the custom template for a key, medium and locale.
func (c *Client) GetNotificationTemplateByKey(ctx context.Context, key, medium, locale string) (*NotificationTemplate, error) {
	filter := eqFilter("key", key) + " and " + eqFilter("medium", medium) + " and " + eqFilter("locale", locale)
	items, err := listAllPages[NotificationTemplate](ctx, c, "/v2026/notification-templates", url.Values{"filters": {filter}})
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Key == key && items[i].Medium == medium && items[i].Locale == locale {
			return &items[i], nil
		}
	}
	return nil, &NotFoundError{fmt.Sprintf("notification template %q (%s, %s) not found", key, medium, locale)}
}

// GetNotificationTemplateDefault returns the default template for a key, medium and locale from
// /v2026/notification-template-defaults.
func (c *Client) GetNotificationTemplateDefault(ctx context.Context, key, medium, locale string) (*NotificationTemplate, error) {
	filter := eqFilter("key", key) + " and " + eqFilter("medium", medium) + " and " + eqFilter("locale", locale)
	items, err := listAllPages[NotificationTemplate](ctx, c, "/v2026/notification-template-defaults", url.Values{"filters": {filter}})
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].Key == key && items[i].Medium == medium && items[i].Locale == locale {
			return &items[i], nil
		}
	}
	return nil, &NotFoundError{fmt.Sprintf("default notification template %q (%s, %s) not found", key, medium, locale)}
}

// SaveNotificationTemplate creates the custom template for the key, medium and locale of template,
// or updates it when it already exists.
func (c *Client) SaveNotificationTemplate(ctx context.Context, template *NotificationTemplate) (*NotificationTemplate, error) {
	var saved NotificationTemplate
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/notification-templates", template, &saved); err != nil {
		return nil, err
	}
	return &saved, nil
}

func (c *Client) DeleteNotificationTemplate(ctx context.Context, key, medium, locale string) error {
	items := []NotificationTemplateBulkDeleteItem{{Key: key, Medium: medium, Locale: locale}}
	return c.doJSON(ctx, http.MethodPost, "/v2026/notification-templates/bulk-delete", items, nil)
}

var _ resource.Resource = &NotificationTemplateResource{}
var _ resource.ResourceWithImportState = &NotificationTemplateResource{}

func NewNotificationTemplateResource() resource.Resource {
	return &NotificationTemplateResource{}
}

type NotificationTemplateResource struct {
	client *Config
}

type NotificationTemplateModel struct {
	ID                types.String `tfsdk:"id"`
	Key               types.String `tfsdk:"key"`
	Medium            types.String `tfsdk:"medium"`
	Locale            types.String `tfsdk:"locale"`
	Name              types.String `tfsdk:"name"`
	Subject           types.String `tfsdk:"subject"`
	Header            types.String `tfsdk:"header"`
	Body              types.String `tfsdk:"body"`
	Footer            types.String `tfsdk:"footer"`
	From              types.String `tfsdk:"from"`
	ReplyTo           types.String `tfsdk:"reply_to"`
	Description       types.String `tfsdk:"description"`
	SlackTemplateJSON types.String `tfsdk:"slack_template_json"`
	TeamsTemplateJSON types.String `tfsdk:"teams_template_json"`
	Created           types.String `tfsdk:"created"`
	Modified          types.String `tfsdk:"modified"`
}

func (r *NotificationTemplateResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_template"
}

// notificationTemplateOptionalComputed returns an optional string attribute that keeps the API
// value when it is not configured.
func notificationTemplateOptionalComputed(description string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: description + " When not set, a new template uses the value of the default template, and later the value stored in IdentityNow is kept.",
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
	}
}

func (r *NotificationTemplateResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a custom notification template. IdentityNow only allows customizing existing templates: the resource creates the custom template for a `key`, `medium` and `locale`, starting from the default template for the attributes that are not configured, and destroying it restores the default template. Creating fails when a custom template already exists for the key, medium and locale; import it instead.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Template ID",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"key": schema.StringAttribute{
				MarkdownDescription: "Key of the template to customize, e.g. `cloud_manual_work_item_summary`. Changing it forces a new template.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"medium": schema.StringAttribute{
				MarkdownDescription: "Message medium: `EMAIL`, `SLACK` or `TEAMS`. Changing it forces a new template.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{triggerSubscriptionOneOfValidator{values: []string{"EMAIL", "SLACK", "TEAMS"}}},
			},
			"locale": schema.StringAttribute{
				MarkdownDescription: "Locale of the message text as a BCP 47 language tag, e.g. `en`. Changing it forces a new template.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name":        notificationTemplateOptionalComputed("Template name."),
			"subject":     notificationTemplateOptionalComputed("Subject line. Supports Velocity template variables."),
			"body":        notificationTemplateOptionalComputed("Message body. Supports Velocity template variables."),
			"from":        notificationTemplateOptionalComputed("\"From:\" address, e.g. a verified sender address."),
			"reply_to":    notificationTemplateOptionalComputed("\"Reply To\" address."),
			"description": notificationTemplateOptionalComputed("Template description."),
			"header": schema.StringAttribute{
				MarkdownDescription: "Deprecated header of the template, read only. The header is now part of `body`; the API rejects non-null values.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"footer": schema.StringAttribute{
				MarkdownDescription: "Deprecated footer of the template, read only. The footer is now part of `body`; the API rejects non-null values.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"slack_template_json": schema.StringAttribute{
				MarkdownDescription: "Slack template as a JSON object, e.g. `text`, `blocks` and `attachments`. Null values and empty values are ignored when comparing with the API value.",
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"teams_template_json": schema.StringAttribute{
				MarkdownDescription: "Microsoft Teams template as a JSON object, e.g. `title`, `text` and `messageJSON`. Null values and empty values are ignored when comparing with the API value.",
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"created": schema.StringAttribute{
				MarkdownDescription: "Creation date",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"modified": schema.StringAttribute{
				MarkdownDescription: "Last modification date",
				Computed:            true,
			},
		},
	}
}

func (r *NotificationTemplateResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// notificationTemplateJSONFromModel decodes a JSON object attribute, nil when unset.
func notificationTemplateJSONFromModel(value types.String, attribute string, diags *diag.Diagnostics) map[string]interface{} {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(value.ValueString()), &decoded); err != nil {
		diags.AddAttributeError(path.Root(attribute), "Invalid JSON", err.Error())
		return nil
	}
	return decoded
}

// notificationTemplateFromModel builds the full request body from the plan. Unknown optional
// values are omitted, so the API keeps or defaults them. Header and footer are never sent.
func notificationTemplateFromModel(data NotificationTemplateModel, diags *diag.Diagnostics) *NotificationTemplate {
	return &NotificationTemplate{
		ID:            data.ID.ValueString(),
		Key:           data.Key.ValueString(),
		Medium:        data.Medium.ValueString(),
		Locale:        data.Locale.ValueString(),
		Name:          data.Name.ValueString(),
		Subject:       data.Subject.ValueString(),
		Body:          data.Body.ValueString(),
		From:          data.From.ValueString(),
		ReplyTo:       data.ReplyTo.ValueString(),
		Description:   data.Description.ValueString(),
		SlackTemplate: notificationTemplateJSONFromModel(data.SlackTemplateJSON, "slack_template_json", diags),
		TeamsTemplate: notificationTemplateJSONFromModel(data.TeamsTemplateJSON, "teams_template_json", diags),
	}
}

// notificationTemplateWithoutEmpty removes null, false, empty string and empty collection values
// from JSON objects, so defaults filled in by the API do not show as a difference.
func notificationTemplateWithoutEmpty(value interface{}) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		cleaned := map[string]interface{}{}
		for key, item := range v {
			item = notificationTemplateWithoutEmpty(item)
			switch typed := item.(type) {
			case nil:
				continue
			case bool:
				if !typed {
					continue
				}
			case string:
				if typed == "" {
					continue
				}
			case map[string]interface{}:
				if len(typed) == 0 {
					continue
				}
			case []interface{}:
				if len(typed) == 0 {
					continue
				}
			}
			cleaned[key] = item
		}
		return cleaned
	case []interface{}:
		cleaned := make([]interface{}, len(v))
		for i, item := range v {
			cleaned[i] = notificationTemplateWithoutEmpty(item)
		}
		return cleaned
	}
	return value
}

// notificationTemplateJSONState returns the state value of a Slack or Teams template JSON attribute.
// The prior value is kept when it is equal to the API value after removing empty values.
func notificationTemplateJSONState(prior types.String, value map[string]interface{}) types.String {
	cleaned := notificationTemplateWithoutEmpty(map[string]interface{}(value))
	if !prior.IsNull() && !prior.IsUnknown() {
		var priorValue interface{}
		if err := json.Unmarshal([]byte(prior.ValueString()), &priorValue); err == nil && reflect.DeepEqual(notificationTemplateWithoutEmpty(priorValue), cleaned) {
			return prior
		}
	}
	return jsonStringState(types.StringNull(), cleaned)
}

// notificationTemplateNullableString maps a nullable API string to a state value.
func notificationTemplateNullableString(value *string) types.String {
	if value == nil {
		return types.StringNull()
	}
	return types.StringValue(*value)
}

// setNotificationTemplateState refreshes the model from the API.
func setNotificationTemplateState(data *NotificationTemplateModel, template *NotificationTemplate) {
	data.ID = types.StringValue(template.ID)
	data.Key = types.StringValue(template.Key)
	data.Medium = types.StringValue(template.Medium)
	data.Locale = types.StringValue(template.Locale)
	data.Name = types.StringValue(template.Name)
	data.Subject = types.StringValue(template.Subject)
	data.Header = notificationTemplateNullableString(template.Header)
	data.Body = types.StringValue(template.Body)
	data.Footer = notificationTemplateNullableString(template.Footer)
	data.From = types.StringValue(template.From)
	data.ReplyTo = types.StringValue(template.ReplyTo)
	data.Description = types.StringValue(template.Description)
	data.SlackTemplateJSON = notificationTemplateJSONState(data.SlackTemplateJSON, template.SlackTemplate)
	data.TeamsTemplateJSON = notificationTemplateJSONState(data.TeamsTemplateJSON, template.TeamsTemplate)
	data.Created = types.StringValue(template.Created)
	data.Modified = types.StringValue(template.Modified)
}

// resolveNotificationTemplateComputed keeps the planned values and resolves unknown values from the
// API response after Create or Update.
func resolveNotificationTemplateComputed(data *NotificationTemplateModel, saved *NotificationTemplate) {
	if data.ID.IsUnknown() {
		data.ID = types.StringValue(saved.ID)
	}
	data.Name = computedStringFromAPI(data.Name, saved.Name)
	data.Subject = computedStringFromAPI(data.Subject, saved.Subject)
	data.Body = computedStringFromAPI(data.Body, saved.Body)
	data.From = computedStringFromAPI(data.From, saved.From)
	data.ReplyTo = computedStringFromAPI(data.ReplyTo, saved.ReplyTo)
	data.Description = computedStringFromAPI(data.Description, saved.Description)
	if data.Header.IsUnknown() {
		data.Header = notificationTemplateNullableString(saved.Header)
	}
	if data.Footer.IsUnknown() {
		data.Footer = notificationTemplateNullableString(saved.Footer)
	}
	if data.Created.IsUnknown() {
		data.Created = types.StringValue(saved.Created)
	}
	data.Modified = types.StringValue(saved.Modified)
}

// notificationTemplateFillFromDefault sets the unknown (not configured) text attributes of a new
// template to the values of the default template, so the new custom template starts as a copy of
// the default instead of an empty template.
func notificationTemplateFillFromDefault(data *NotificationTemplateModel, defaults *NotificationTemplate) {
	fill := func(value *types.String, defaultValue string) {
		if value.IsUnknown() {
			*value = types.StringValue(defaultValue)
		}
	}
	fill(&data.Name, defaults.Name)
	fill(&data.Subject, defaults.Subject)
	fill(&data.Body, defaults.Body)
	fill(&data.From, defaults.From)
	fill(&data.ReplyTo, defaults.ReplyTo)
	fill(&data.Description, defaults.Description)
}

// notificationTemplatePrepareCreate checks that no custom template exists for the key, medium and
// locale (POST would silently overwrite it, and destroying the resource would delete it), and fills
// the attributes that are not configured from the default template.
func notificationTemplatePrepareCreate(ctx context.Context, client *Client, data *NotificationTemplateModel, diagnostics *diag.Diagnostics) {
	key, medium, locale := data.Key.ValueString(), data.Medium.ValueString(), data.Locale.ValueString()
	existing, err := client.GetNotificationTemplateByKey(ctx, key, medium, locale)
	if err == nil {
		diagnostics.AddError("Notification template already exists",
			fmt.Sprintf("A custom notification template for key %q, medium %s and locale %q already exists (ID %s). "+
				"Import it with `terraform import identitynow_notification_template.<name> %s` to manage it with Terraform.", key, medium, locale, existing.ID, existing.ID))
		return
	}
	if !isNotFound(err) {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to check for an existing notification template: %s", err))
		return
	}
	defaults, err := client.GetNotificationTemplateDefault(ctx, key, medium, locale)
	if err != nil {
		if isNotFound(err) {
			// Without a default the API decides; it rejects keys that do not exist.
			return
		}
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read the default notification template: %s", err))
		return
	}
	notificationTemplateFillFromDefault(data, defaults)
}

// save sends the planned template and returns the planned model with resolved computed values.
// On create it first checks for an existing custom template and starts from the default template.
func (r *NotificationTemplateResource) save(ctx context.Context, plan tfsdk.Plan, create bool, diagnostics *diag.Diagnostics) (*NotificationTemplateModel, bool) {
	var data NotificationTemplateModel
	diagnostics.Append(plan.Get(ctx, &data)...)
	if diagnostics.HasError() {
		return nil, false
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		diagnostics.AddError("Client Error", err.Error())
		return nil, false
	}
	if create {
		notificationTemplatePrepareCreate(ctx, client, &data, diagnostics)
		if diagnostics.HasError() {
			return nil, false
		}
	}
	template := notificationTemplateFromModel(data, diagnostics)
	if diagnostics.HasError() {
		return nil, false
	}
	saved, err := client.SaveNotificationTemplate(ctx, template)
	if err != nil {
		diagnostics.AddError("Client Error", fmt.Sprintf("Unable to save notification template: %s", err))
		return nil, false
	}
	resolveNotificationTemplateComputed(&data, saved)
	return &data, true
}

func (r *NotificationTemplateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	if data, ok := r.save(ctx, req.Plan, true, &resp.Diagnostics); ok {
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}
}

func (r *NotificationTemplateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data NotificationTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	template, err := client.GetNotificationTemplate(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read notification template: %s", err))
		return
	}
	setNotificationTemplateState(&data, template)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update sends the full template again. The API matches it by key, medium and locale.
func (r *NotificationTemplateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	if data, ok := r.save(ctx, req.Plan, false, &resp.Diagnostics); ok {
		resp.Diagnostics.Append(resp.State.Set(ctx, data)...)
	}
}

func (r *NotificationTemplateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data NotificationTemplateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteNotificationTemplate(ctx, data.Key.ValueString(), data.Medium.ValueString(), data.Locale.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete notification template: %s", err))
	}
}

func (r *NotificationTemplateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &NotificationTemplateDataSource{}
var _ datasource.DataSourceWithValidateConfig = &NotificationTemplateDataSource{}

func NewNotificationTemplateDataSource() datasource.DataSource {
	return &NotificationTemplateDataSource{}
}

type NotificationTemplateDataSource struct {
	client *Config
}

func (d *NotificationTemplateDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_notification_template"
}

func (d *NotificationTemplateDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	lookup := func(description string) dsschema.StringAttribute {
		return dsschema.StringAttribute{MarkdownDescription: description, Optional: true, Computed: true}
	}
	computed := func(description string) dsschema.StringAttribute {
		return dsschema.StringAttribute{MarkdownDescription: description, Computed: true}
	}
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a custom notification template by ID, or by key, medium and locale.",
		Attributes: map[string]dsschema.Attribute{
			"id":                  lookup("Template ID. Set either `id`, or `key`, `medium` and `locale`."),
			"key":                 lookup("Template key. Set either `id`, or `key`, `medium` and `locale`."),
			"medium":              lookup("Message medium: `EMAIL`, `SLACK` or `TEAMS`."),
			"locale":              lookup("Locale as a BCP 47 language tag, e.g. `en`."),
			"name":                computed("Template name"),
			"subject":             computed("Subject line"),
			"header":              computed("Deprecated header"),
			"body":                computed("Message body"),
			"footer":              computed("Deprecated footer"),
			"from":                computed("\"From:\" address"),
			"reply_to":            computed("\"Reply To\" address"),
			"description":         computed("Template description"),
			"slack_template_json": computed("Slack template as a JSON object"),
			"teams_template_json": computed("Microsoft Teams template as a JSON object"),
			"created":             computed("Creation date"),
			"modified":            computed("Last modification date"),
		},
	}
}

func (d *NotificationTemplateDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var data NotificationTemplateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	byKey := !data.Key.IsNull() || !data.Medium.IsNull() || !data.Locale.IsNull()
	allKey := !data.Key.IsNull() && !data.Medium.IsNull() && !data.Locale.IsNull()
	if data.ID.IsNull() == !byKey || (byKey && !allKey) {
		resp.Diagnostics.AddError("Invalid configuration", "Set either id, or all of key, medium and locale.")
	}
}

func (d *NotificationTemplateDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *NotificationTemplateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data NotificationTemplateModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var template *NotificationTemplate
	attribute := path.Root("id")
	if !data.ID.IsNull() {
		template, err = client.GetNotificationTemplate(ctx, data.ID.ValueString())
	} else {
		attribute = path.Root("key")
		template, err = client.GetNotificationTemplateByKey(ctx, data.Key.ValueString(), data.Medium.ValueString(), data.Locale.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(attribute, "Notification template not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read notification template: %s", err))
		return
	}
	setNotificationTemplateState(&data, template)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
