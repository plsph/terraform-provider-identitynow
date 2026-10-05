package provider

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewScheduledSearchResource)
	registerDataSource(NewScheduledSearchDataSource)
}

// ScheduledSearch is a scheduled search as returned by the /v2026/scheduled-searches API.
type ScheduledSearch struct {
	ID                  string                 `json:"id,omitempty"`
	Owner               *ScheduledSearchRef    `json:"owner,omitempty"`
	OwnerID             string                 `json:"ownerId,omitempty"`
	Name                *string                `json:"name,omitempty"`
	Description         *string                `json:"description,omitempty"`
	SavedSearchID       string                 `json:"savedSearchId"`
	Created             string                 `json:"created,omitempty"`
	Modified            string                 `json:"modified,omitempty"`
	Schedule            map[string]interface{} `json:"schedule"`
	Recipients          []ScheduledSearchRef   `json:"recipients"`
	Enabled             *bool                  `json:"enabled,omitempty"`
	EmailEmptyResults   *bool                  `json:"emailEmptyResults,omitempty"`
	DisplayQueryDetails *bool                  `json:"displayQueryDetails,omitempty"`
}

// ScheduledSearchRef is a typed reference to an identity, the owner or a recipient.
type ScheduledSearchRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func (c *Client) GetScheduledSearch(ctx context.Context, id string) (*ScheduledSearch, error) {
	var search ScheduledSearch
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/scheduled-searches/%s", id), nil, &search); err != nil {
		return nil, err
	}
	return &search, nil
}

func (c *Client) CreateScheduledSearch(ctx context.Context, search *ScheduledSearch) (*ScheduledSearch, error) {
	var created ScheduledSearch
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/scheduled-searches", search, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdateScheduledSearch replaces the managed fields with PUT, keeping the owner and other unmanaged fields.
func (c *Client) UpdateScheduledSearch(ctx context.Context, id string, managed map[string]interface{}) (*ScheduledSearch, error) {
	var updated ScheduledSearch
	if err := c.putMerged(ctx, apiPath("/v2026/scheduled-searches/%s", id), managed, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteScheduledSearch(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/scheduled-searches/%s", id), nil, nil)
}

var _ resource.Resource = &ScheduledSearchResource{}
var _ resource.ResourceWithImportState = &ScheduledSearchResource{}

func NewScheduledSearchResource() resource.Resource {
	return &ScheduledSearchResource{}
}

type ScheduledSearchResource struct {
	client *Config
}

type ScheduledSearchModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	SavedSearchID       types.String `tfsdk:"saved_search_id"`
	ScheduleJSON        types.String `tfsdk:"schedule_json"`
	Recipient           types.List   `tfsdk:"recipient"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	EmailEmptyResults   types.Bool   `tfsdk:"email_empty_results"`
	DisplayQueryDetails types.Bool   `tfsdk:"display_query_details"`
	Owner               types.List   `tfsdk:"owner"`
	Created             types.String `tfsdk:"created"`
	Modified            types.String `tfsdk:"modified"`
}

type ScheduledSearchRefModel struct {
	Type types.String `tfsdk:"type"`
	ID   types.String `tfsdk:"id"`
}

var scheduledSearchRefObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"type": types.StringType,
	"id":   types.StringType,
}}

func (r *ScheduledSearchResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_search"
}

func (r *ScheduledSearchResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a scheduled search, which runs a saved search on a schedule and emails the results. The scheduled search is owned by the identity the provider authenticates as.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Scheduled search ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Scheduled search name.",
				Optional:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Scheduled search description.",
				Optional:            true,
			},
			"saved_search_id": schema.StringAttribute{
				MarkdownDescription: "ID of the saved search that is run.",
				Required:            true,
			},
			"schedule_json": schema.StringAttribute{
				MarkdownDescription: "Schedule as a JSON object with `type` (`DAILY`, `WEEKLY`, `MONTHLY`, `CALENDAR` or `ANNUALLY`), the `months`, `days` and `hours` selectors (each with `type` `LIST` or `RANGE`, `values` and an optional `interval`), `expiration` and `timeZoneId`. `hours` is required. Use `jsonencode()`. The value is compared on the configured fields only, so formatting, key order and fields IdentityNow adds do not produce a diff.",
				Required:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the scheduled search is enabled. When not set, the value chosen by IdentityNow is kept.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"email_empty_results": schema.BoolAttribute{
				MarkdownDescription: "Whether an email is sent when the search returns no results. When not set, the value chosen by IdentityNow is kept.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"display_query_details": schema.BoolAttribute{
				MarkdownDescription: "Whether the email includes the query and a preview of the results, which can contain personal data. When not set, the value chosen by IdentityNow is kept.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"owner": schema.ListNestedAttribute{
				MarkdownDescription: "Owner of the scheduled search, with `type` and `id`. Set by the API to the identity that created it.",
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{MarkdownDescription: "Owner type.", Computed: true},
						"id":   schema.StringAttribute{MarkdownDescription: "Owner identity ID.", Computed: true},
					},
				},
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
		Blocks: map[string]schema.Block{
			"recipient": schema.ListNestedBlock{
				MarkdownDescription: "Identity that receives the search results by email. At least one block is required.",
				Validators:          []validator.List{listSizeBetween(1, 0)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{MarkdownDescription: "Identity ID.", Required: true},
						"type": schema.StringAttribute{
							MarkdownDescription: "Recipient type, `IDENTITY` (default).",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString("IDENTITY"),
						},
					},
				},
			},
		},
	}
}

func (r *ScheduledSearchResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func scheduledSearchRecipientsValue(ctx context.Context, list types.List, diags *diag.Diagnostics) []ScheduledSearchRef {
	recipients := []ScheduledSearchRef{}
	if list.IsNull() || list.IsUnknown() {
		return recipients
	}
	var models []ScheduledSearchRefModel
	diags.Append(list.ElementsAs(ctx, &models, false)...)
	for _, model := range models {
		recipients = append(recipients, ScheduledSearchRef{Type: model.Type.ValueString(), ID: model.ID.ValueString()})
	}
	return recipients
}

// scheduledSearchRefsState maps references to a list in the prior order.
func scheduledSearchRefsState(ctx context.Context, refs []ScheduledSearchRef, prior types.List, diags *diag.Diagnostics) types.List {
	if len(refs) == 0 {
		return types.ListNull(scheduledSearchRefObjectType)
	}
	ordered := orderByPriorIDs(refs, scheduledSearchRecipientsValue(ctx, prior, diags), func(r ScheduledSearchRef) string { return r.ID })
	models := make([]ScheduledSearchRefModel, 0, len(ordered))
	for _, ref := range ordered {
		refType := ref.Type
		if refType == "" {
			refType = "IDENTITY"
		}
		models = append(models, ScheduledSearchRefModel{Type: types.StringValue(refType), ID: types.StringValue(ref.ID)})
	}
	list, d := types.ListValueFrom(ctx, scheduledSearchRefObjectType, models)
	diags.Append(d...)
	return list
}

// scheduledSearchFromModel converts the model to the create request body.
func scheduledSearchFromModel(ctx context.Context, data ScheduledSearchModel, diags *diag.Diagnostics) *ScheduledSearch {
	search := &ScheduledSearch{
		Name:                stringPointer(data.Name),
		Description:         stringPointer(data.Description),
		SavedSearchID:       data.SavedSearchID.ValueString(),
		Schedule:            map[string]interface{}{},
		Recipients:          scheduledSearchRecipientsValue(ctx, data.Recipient, diags),
		Enabled:             boolPointer(data.Enabled),
		EmailEmptyResults:   boolPointer(data.EmailEmptyResults),
		DisplayQueryDetails: boolPointer(data.DisplayQueryDetails),
	}
	if schedule, ok := campaignTemplateJSONValue(data.ScheduleJSON, "schedule_json", diags).(map[string]interface{}); ok {
		search.Schedule = schedule
	}
	return search
}

// scheduledSearchManagedFields returns the fields sent with PUT. Unset names and descriptions are
// sent as null to clear them; flags are only sent when known, so values chosen by the API are kept.
func scheduledSearchManagedFields(ctx context.Context, data ScheduledSearchModel, diags *diag.Diagnostics) map[string]interface{} {
	search := scheduledSearchFromModel(ctx, data, diags)
	managed := map[string]interface{}{
		"name":          search.Name,
		"description":   search.Description,
		"savedSearchId": search.SavedSearchID,
		"schedule":      search.Schedule,
		"recipients":    search.Recipients,
	}
	if search.Enabled != nil {
		managed["enabled"] = *search.Enabled
	}
	if search.EmailEmptyResults != nil {
		managed["emailEmptyResults"] = *search.EmailEmptyResults
	}
	if search.DisplayQueryDetails != nil {
		managed["displayQueryDetails"] = *search.DisplayQueryDetails
	}
	return managed
}

// setScheduledSearchComputed resolves the computed attributes after apply from the API response.
func setScheduledSearchComputed(ctx context.Context, data *ScheduledSearchModel, search *ScheduledSearch, diags *diag.Diagnostics) {
	data.ID = types.StringValue(search.ID)
	data.Created = types.StringValue(search.Created)
	data.Modified = types.StringValue(search.Modified)
	data.Enabled = computedBoolFromAPI(data.Enabled, search.Enabled)
	data.EmailEmptyResults = computedBoolFromAPI(data.EmailEmptyResults, search.EmailEmptyResults)
	data.DisplayQueryDetails = computedBoolFromAPI(data.DisplayQueryDetails, search.DisplayQueryDetails)
	owner := search.Owner
	if owner == nil && search.OwnerID != "" {
		owner = &ScheduledSearchRef{Type: "IDENTITY", ID: search.OwnerID}
	}
	var owners []ScheduledSearchRef
	if owner != nil {
		owners = []ScheduledSearchRef{*owner}
	}
	data.Owner = scheduledSearchRefsState(ctx, owners, types.ListNull(scheduledSearchRefObjectType), diags)
}

// setScheduledSearchState maps an API scheduled search onto the model when reading.
func setScheduledSearchState(ctx context.Context, data *ScheduledSearchModel, search *ScheduledSearch, diags *diag.Diagnostics) {
	deref := func(v *string) string {
		if v == nil {
			return ""
		}
		return *v
	}
	setScheduledSearchComputed(ctx, data, search, diags)
	data.Name = optionalStringState(data.Name, deref(search.Name))
	data.Description = optionalStringState(data.Description, deref(search.Description))
	data.SavedSearchID = types.StringValue(search.SavedSearchID)
	data.ScheduleJSON = jsonSubsetState(data.ScheduleJSON, search.Schedule)
	data.Recipient = scheduledSearchRefsState(ctx, search.Recipients, data.Recipient, diags)
	data.Enabled = types.BoolValue(search.Enabled != nil && *search.Enabled)
	data.EmailEmptyResults = types.BoolValue(search.EmailEmptyResults != nil && *search.EmailEmptyResults)
	data.DisplayQueryDetails = types.BoolValue(search.DisplayQueryDetails != nil && *search.DisplayQueryDetails)
}

func (r *ScheduledSearchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ScheduledSearchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	search := scheduledSearchFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateScheduledSearch(ctx, search)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create scheduled search: %s", err))
		return
	}
	setScheduledSearchComputed(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduledSearchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ScheduledSearchModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	search, err := client.GetScheduledSearch(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read scheduled search: %s", err))
		return
	}
	setScheduledSearchState(ctx, &data, search, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduledSearchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ScheduledSearchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	managed := scheduledSearchManagedFields(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.UpdateScheduledSearch(ctx, data.ID.ValueString(), managed)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update scheduled search: %s", err))
		return
	}
	data.Modified = types.StringValue(updated.Modified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ScheduledSearchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ScheduledSearchModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteScheduledSearch(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete scheduled search: %s", err))
	}
}

func (r *ScheduledSearchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &ScheduledSearchDataSource{}

func NewScheduledSearchDataSource() datasource.DataSource {
	return &ScheduledSearchDataSource{}
}

type ScheduledSearchDataSource struct {
	client *Config
}

func (d *ScheduledSearchDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_scheduled_search"
}

func scheduledSearchRefDataSourceAttribute(description string) dsschema.ListNestedAttribute {
	return dsschema.ListNestedAttribute{
		MarkdownDescription: description,
		Computed:            true,
		NestedObject: dsschema.NestedAttributeObject{
			Attributes: map[string]dsschema.Attribute{
				"type": dsschema.StringAttribute{MarkdownDescription: "Identity type.", Computed: true},
				"id":   dsschema.StringAttribute{MarkdownDescription: "Identity ID.", Computed: true},
			},
		},
	}
}

func (d *ScheduledSearchDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a scheduled search by ID.",
		Attributes: map[string]dsschema.Attribute{
			"id":                    dsschema.StringAttribute{MarkdownDescription: "Scheduled search ID.", Required: true},
			"name":                  dsschema.StringAttribute{MarkdownDescription: "Scheduled search name.", Computed: true},
			"description":           dsschema.StringAttribute{MarkdownDescription: "Scheduled search description.", Computed: true},
			"saved_search_id":       dsschema.StringAttribute{MarkdownDescription: "ID of the saved search that is run.", Computed: true},
			"schedule_json":         dsschema.StringAttribute{MarkdownDescription: "Schedule as a JSON object.", Computed: true},
			"recipient":             scheduledSearchRefDataSourceAttribute("Identities that receive the search results, with `type` and `id`."),
			"enabled":               dsschema.BoolAttribute{MarkdownDescription: "Whether the scheduled search is enabled.", Computed: true},
			"email_empty_results":   dsschema.BoolAttribute{MarkdownDescription: "Whether an email is sent when the search returns no results.", Computed: true},
			"display_query_details": dsschema.BoolAttribute{MarkdownDescription: "Whether the email includes the query and a preview of the results.", Computed: true},
			"owner":                 scheduledSearchRefDataSourceAttribute("Owner of the scheduled search, with `type` and `id`."),
			"created":               dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
			"modified":              dsschema.StringAttribute{MarkdownDescription: "Last modification date.", Computed: true},
		},
	}
}

func (d *ScheduledSearchDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ScheduledSearchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ScheduledSearchModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	search, err := client.GetScheduledSearch(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read scheduled search: %s", err))
		return
	}
	setScheduledSearchState(ctx, &data, search, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
