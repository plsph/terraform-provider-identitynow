package provider

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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewSavedSearchResource)
	registerDataSource(NewSavedSearchDataSource)
}

// SavedSearch is a saved search as returned by the /v2026/saved-searches API.
type SavedSearch struct {
	ID          string                 `json:"id,omitempty"`
	Owner       *SavedSearchRef        `json:"owner,omitempty"`
	OwnerID     string                 `json:"ownerId,omitempty"`
	Public      *bool                  `json:"public,omitempty"`
	Name        string                 `json:"name"`
	Description *string                `json:"description,omitempty"`
	Created     string                 `json:"created,omitempty"`
	Modified    string                 `json:"modified,omitempty"`
	Indices     []string               `json:"indices"`
	Columns     map[string]interface{} `json:"columns,omitempty"`
	Query       string                 `json:"query"`
	Fields      []string               `json:"fields,omitempty"`
	OrderBy     map[string][]string    `json:"orderBy,omitempty"`
	Sort        []string               `json:"sort,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
}

// SavedSearchRef is a typed reference, e.g. the owner of a saved search.
type SavedSearchRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func (c *Client) GetSavedSearch(ctx context.Context, id string) (*SavedSearch, error) {
	var search SavedSearch
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/saved-searches/%s", id), nil, &search); err != nil {
		return nil, err
	}
	return &search, nil
}

func (c *Client) CreateSavedSearch(ctx context.Context, search *SavedSearch) (*SavedSearch, error) {
	var created SavedSearch
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/saved-searches", search, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdateSavedSearch replaces the managed fields with PUT, keeping the owner and other unmanaged fields.
func (c *Client) UpdateSavedSearch(ctx context.Context, id string, managed map[string]interface{}) (*SavedSearch, error) {
	var updated SavedSearch
	if err := c.putMerged(ctx, apiPath("/v2026/saved-searches/%s", id), managed, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteSavedSearch(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/saved-searches/%s", id), nil, nil)
}

var _ resource.Resource = &SavedSearchResource{}
var _ resource.ResourceWithImportState = &SavedSearchResource{}

func NewSavedSearchResource() resource.Resource {
	return &SavedSearchResource{}
}

type SavedSearchResource struct {
	client *Config
}

type SavedSearchModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Public      types.Bool   `tfsdk:"public"`
	Indices     types.List   `tfsdk:"indices"`
	Query       types.String `tfsdk:"query"`
	Fields      types.List   `tfsdk:"fields"`
	OrderBy     types.Map    `tfsdk:"order_by"`
	Sort        types.List   `tfsdk:"sort"`
	FiltersJSON types.String `tfsdk:"filters_json"`
	ColumnsJSON types.String `tfsdk:"columns_json"`
	OwnerID     types.String `tfsdk:"owner_id"`
	Created     types.String `tfsdk:"created"`
	Modified    types.String `tfsdk:"modified"`
}

var savedSearchOrderByType = types.ListType{ElemType: types.StringType}

func (r *SavedSearchResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saved_search"
}

func (r *SavedSearchResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a saved search, which can be run on a schedule with `identitynow_scheduled_search`. The saved search is owned by the identity the provider authenticates as.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Saved search ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Saved search name.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Saved search description.",
				Optional:            true,
			},
			"public": schema.BoolAttribute{
				MarkdownDescription: "Whether the saved search is visible to others than the owner. Saved searches cannot be made public at this time, so it is always `false`.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"indices": schema.ListAttribute{
				MarkdownDescription: "Search indices: `accessprofiles`, `accountactivities`, `entitlements`, `events`, `identities`, `roles` or `*`.",
				Required:            true,
				ElementType:         types.StringType,
				Validators:          []validator.List{listSizeBetween(1, 0)},
			},
			"query": schema.StringAttribute{
				MarkdownDescription: "Search query in Elasticsearch query string syntax, such as `@accounts(disabled:true)`.",
				Required:            true,
			},
			"fields": schema.ListAttribute{
				MarkdownDescription: "Fields searched in a multi-field query.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"order_by": schema.MapAttribute{
				MarkdownDescription: "Map of document type to the fields its results are sorted by, e.g. `{ identity = [\"lastName\", \"firstName\"] }`. It takes precedence over `sort`.",
				Optional:            true,
				ElementType:         savedSearchOrderByType,
			},
			"sort": schema.ListAttribute{
				MarkdownDescription: "Fields the results are sorted by.",
				Optional:            true,
				ElementType:         types.StringType,
			},
			"filters_json": schema.StringAttribute{
				MarkdownDescription: "Filters per field name as a JSON object. Each filter has a `type` (`EXISTS`, `RANGE` or `TERMS`), a `range` (`lower` and `upper` bounds with `value` and `inclusive`), `terms` and `exclude`. Use `jsonencode()`.",
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"columns_json": schema.StringAttribute{
				MarkdownDescription: "Columns returned per document type as a JSON object, each column with a `field` and a `header`, e.g. `{ identity = [{ field = \"displayName\", header = \"Display Name\" }] }`. Use `jsonencode()`.",
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"owner_id": schema.StringAttribute{
				MarkdownDescription: "ID of the identity that owns the saved search. The owner cannot be changed.",
				Computed:            true,
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

func (r *SavedSearchResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func savedSearchStrings(ctx context.Context, list types.List, diags *diag.Diagnostics) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	values := []string{}
	diags.Append(list.ElementsAs(ctx, &values, false)...)
	return values
}

// savedSearchStringsState maps an optional string list, keeping null when unset and the API returns none.
func savedSearchStringsState(ctx context.Context, values []string, prior types.List, diags *diag.Diagnostics) types.List {
	if len(values) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.ListNull(types.StringType)
	}
	if values == nil {
		values = []string{}
	}
	list, d := types.ListValueFrom(ctx, types.StringType, values)
	diags.Append(d...)
	return list
}

func savedSearchOrderByValue(ctx context.Context, value types.Map, diags *diag.Diagnostics) map[string][]string {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	orderBy := map[string][]string{}
	diags.Append(value.ElementsAs(ctx, &orderBy, false)...)
	return orderBy
}

func savedSearchOrderByState(ctx context.Context, orderBy map[string][]string, prior types.Map, diags *diag.Diagnostics) types.Map {
	if len(orderBy) == 0 && (prior.IsNull() || prior.IsUnknown()) {
		return types.MapNull(savedSearchOrderByType)
	}
	if orderBy == nil {
		orderBy = map[string][]string{}
	}
	value, d := types.MapValueFrom(ctx, savedSearchOrderByType, orderBy)
	diags.Append(d...)
	return value
}

// savedSearchFromModel converts the model to the create request body.
func savedSearchFromModel(ctx context.Context, data SavedSearchModel, diags *diag.Diagnostics) *SavedSearch {
	search := &SavedSearch{
		Name:        data.Name.ValueString(),
		Description: stringPointer(data.Description),
		Indices:     savedSearchStrings(ctx, data.Indices, diags),
		Query:       data.Query.ValueString(),
		Fields:      savedSearchStrings(ctx, data.Fields, diags),
		OrderBy:     savedSearchOrderByValue(ctx, data.OrderBy, diags),
		Sort:        savedSearchStrings(ctx, data.Sort, diags),
	}
	if filters, ok := campaignTemplateJSONValue(data.FiltersJSON, "filters_json", diags).(map[string]interface{}); ok {
		search.Filters = filters
	}
	if columns, ok := campaignTemplateJSONValue(data.ColumnsJSON, "columns_json", diags).(map[string]interface{}); ok {
		search.Columns = columns
	}
	return search
}

// savedSearchManagedFields returns the fields sent with PUT. Unset optional values are sent as
// null to clear them; the owner is kept as read.
func savedSearchManagedFields(ctx context.Context, data SavedSearchModel, diags *diag.Diagnostics) map[string]interface{} {
	search := savedSearchFromModel(ctx, data, diags)
	managed := map[string]interface{}{
		"name":        search.Name,
		"description": search.Description,
		"indices":     search.Indices,
		"query":       search.Query,
		"fields":      search.Fields,
		"orderBy":     search.OrderBy,
		"sort":        search.Sort,
		"filters":     search.Filters,
		"columns":     search.Columns,
	}
	return managed
}

// setSavedSearchComputed resolves the computed attributes after apply from the API response.
func setSavedSearchComputed(data *SavedSearchModel, search *SavedSearch) {
	data.ID = types.StringValue(search.ID)
	data.Public = types.BoolValue(search.Public != nil && *search.Public)
	ownerID := search.OwnerID
	if search.Owner != nil && search.Owner.ID != "" {
		ownerID = search.Owner.ID
	}
	data.OwnerID = types.StringValue(ownerID)
	data.Created = types.StringValue(search.Created)
	data.Modified = types.StringValue(search.Modified)
}

// setSavedSearchState maps an API saved search onto the model when reading.
func setSavedSearchState(ctx context.Context, data *SavedSearchModel, search *SavedSearch, diags *diag.Diagnostics) {
	setSavedSearchComputed(data, search)
	description := ""
	if search.Description != nil {
		description = *search.Description
	}
	data.Name = types.StringValue(search.Name)
	data.Description = optionalStringState(data.Description, description)
	indices := search.Indices
	if indices == nil {
		indices = []string{}
	}
	var d diag.Diagnostics
	data.Indices, d = types.ListValueFrom(ctx, types.StringType, indices)
	diags.Append(d...)
	data.Query = types.StringValue(search.Query)
	data.Fields = savedSearchStringsState(ctx, search.Fields, data.Fields, diags)
	data.OrderBy = savedSearchOrderByState(ctx, search.OrderBy, data.OrderBy, diags)
	data.Sort = savedSearchStringsState(ctx, search.Sort, data.Sort, diags)
	data.FiltersJSON = jsonSubsetState(data.FiltersJSON, search.Filters)
	data.ColumnsJSON = jsonSubsetState(data.ColumnsJSON, search.Columns)
}

func (r *SavedSearchResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SavedSearchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	search := savedSearchFromModel(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateSavedSearch(ctx, search)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create saved search: %s", err))
		return
	}
	setSavedSearchComputed(&data, created)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SavedSearchResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SavedSearchModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	search, err := client.GetSavedSearch(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read saved search: %s", err))
		return
	}
	setSavedSearchState(ctx, &data, search, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SavedSearchResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SavedSearchModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	managed := savedSearchManagedFields(ctx, data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.UpdateSavedSearch(ctx, data.ID.ValueString(), managed)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update saved search: %s", err))
		return
	}
	data.Modified = types.StringValue(updated.Modified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SavedSearchResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SavedSearchModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteSavedSearch(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete saved search: %s", err))
	}
}

func (r *SavedSearchResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &SavedSearchDataSource{}

func NewSavedSearchDataSource() datasource.DataSource {
	return &SavedSearchDataSource{}
}

type SavedSearchDataSource struct {
	client *Config
}

func (d *SavedSearchDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saved_search"
}

func (d *SavedSearchDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a saved search by ID.",
		Attributes: map[string]dsschema.Attribute{
			"id":           dsschema.StringAttribute{MarkdownDescription: "Saved search ID.", Required: true},
			"name":         dsschema.StringAttribute{MarkdownDescription: "Saved search name.", Computed: true},
			"description":  dsschema.StringAttribute{MarkdownDescription: "Saved search description.", Computed: true},
			"public":       dsschema.BoolAttribute{MarkdownDescription: "Whether the saved search is public.", Computed: true},
			"indices":      dsschema.ListAttribute{MarkdownDescription: "Search indices.", Computed: true, ElementType: types.StringType},
			"query":        dsschema.StringAttribute{MarkdownDescription: "Search query.", Computed: true},
			"fields":       dsschema.ListAttribute{MarkdownDescription: "Fields searched in a multi-field query.", Computed: true, ElementType: types.StringType},
			"order_by":     dsschema.MapAttribute{MarkdownDescription: "Map of document type to sort fields.", Computed: true, ElementType: savedSearchOrderByType},
			"sort":         dsschema.ListAttribute{MarkdownDescription: "Fields the results are sorted by.", Computed: true, ElementType: types.StringType},
			"filters_json": dsschema.StringAttribute{MarkdownDescription: "Filters per field name as a JSON object.", Computed: true},
			"columns_json": dsschema.StringAttribute{MarkdownDescription: "Columns per document type as a JSON object.", Computed: true},
			"owner_id":     dsschema.StringAttribute{MarkdownDescription: "ID of the identity that owns the saved search.", Computed: true},
			"created":      dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
			"modified":     dsschema.StringAttribute{MarkdownDescription: "Last modification date.", Computed: true},
		},
	}
}

func (d *SavedSearchDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SavedSearchDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SavedSearchModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	search, err := client.GetSavedSearch(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read saved search: %s", err))
		return
	}
	setSavedSearchState(ctx, &data, search, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
