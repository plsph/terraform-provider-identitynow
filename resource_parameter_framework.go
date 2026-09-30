package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewParameterResource)
	registerDataSource(NewParameterDataSource)
}

// Parameter is a stored parameter as returned by the /v2026/parameter-storage/parameters API. The
// API never returns the private fields.
type Parameter struct {
	ID                          string                 `json:"id,omitempty"`
	OwnerID                     string                 `json:"ownerId"`
	Name                        string                 `json:"name"`
	Type                        string                 `json:"type"`
	Description                 string                 `json:"description,omitempty"`
	PrimaryField                string                 `json:"primaryField,omitempty"`
	PublicFields                map[string]interface{} `json:"publicFields,omitempty"`
	LastModifiedAt              string                 `json:"lastModifiedAt,omitempty"`
	LastModifiedBy              string                 `json:"lastModifiedBy,omitempty"`
	PrivateFieldsLastModifiedAt string                 `json:"privateFieldsLastModifiedAt,omitempty"`
	PrivateFieldsLastModifiedBy string                 `json:"privateFieldsLastModifiedBy,omitempty"`
}

// ParameterRequest is the body of the create request.
type ParameterRequest struct {
	OwnerID       string                 `json:"ownerId"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Description   string                 `json:"description,omitempty"`
	PublicFields  map[string]interface{} `json:"publicFields,omitempty"`
	PrivateFields string                 `json:"privateFields,omitempty"`
}

func (c *Client) GetParameter(ctx context.Context, id string) (*Parameter, error) {
	var parameter Parameter
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/parameter-storage/parameters/%s", id), nil, &parameter); err != nil {
		return nil, err
	}
	return &parameter, nil
}

func (c *Client) GetParameterByName(ctx context.Context, name string) (*Parameter, error) {
	return findByName(ctx, c, "/v2026/parameter-storage/parameters", "parameter", name, func(p Parameter) string { return p.Name })
}

func (c *Client) CreateParameter(ctx context.Context, parameter *ParameterRequest) (*Parameter, error) {
	var created Parameter
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/parameter-storage/parameters", parameter, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) PatchParameter(ctx context.Context, id string, ops []jsonPatchOp) (*Parameter, error) {
	var updated Parameter
	if err := c.doJSON(ctx, http.MethodPatch, apiPath("/v2026/parameter-storage/parameters/%s", id), ops, &updated, withJSONPatch()); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteParameter(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/parameter-storage/parameters/%s", id), nil, nil)
}

var _ resource.Resource = &ParameterResource{}
var _ resource.ResourceWithImportState = &ParameterResource{}

func NewParameterResource() resource.Resource {
	return &ParameterResource{}
}

type ParameterResource struct {
	client *Config
}

type ParameterModel struct {
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	Description                 types.String `tfsdk:"description"`
	Type                        types.String `tfsdk:"type"`
	OwnerID                     types.String `tfsdk:"owner_id"`
	PublicFieldsJSON            types.String `tfsdk:"public_fields_json"`
	PrivateFields               types.String `tfsdk:"private_fields"`
	PrimaryField                types.String `tfsdk:"primary_field"`
	LastModifiedAt              types.String `tfsdk:"last_modified_at"`
	PrivateFieldsLastModifiedAt types.String `tfsdk:"private_fields_last_modified_at"`
}

func (r *ParameterResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parameter"
}

func (r *ParameterResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a parameter in parameter storage, for example credentials used by connectors. The private fields are write-only in the API: they are stored in state as a sensitive value and changes made outside Terraform are only detected through `private_fields_last_modified_at`.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Parameter ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Human-readable name of the parameter.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the parameter.",
				Optional:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Parameter type, see the parameter storage specification. Cannot be changed, changing it forces a new parameter.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"owner_id": schema.StringAttribute{
				MarkdownDescription: "Identity ID of the parameter owner.",
				Required:            true,
			},
			"public_fields_json": schema.StringAttribute{
				MarkdownDescription: "Public fields of the parameter as a JSON object, as defined by the type specification. Compared semantically.",
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"private_fields": schema.StringAttribute{
				MarkdownDescription: "Private (secret) fields of the parameter as a JWE AES256 encrypted blob containing a JSON object. The provider does not encrypt the value. The API never returns it; removing the attribute leaves the stored private fields unchanged.",
				Optional:            true,
				Sensitive:           true,
			},
			"primary_field": schema.StringAttribute{
				MarkdownDescription: "Name of the primary field in the public fields.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"last_modified_at": schema.StringAttribute{
				MarkdownDescription: "Date any field of the parameter was last changed.",
				Computed:            true,
			},
			"private_fields_last_modified_at": schema.StringAttribute{
				MarkdownDescription: "Date the private fields were last changed.",
				Computed:            true,
			},
		},
	}
}

func (r *ParameterResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// parameterPublicFields decodes public_fields_json, or returns nil when it is not set.
func parameterPublicFields(value types.String) (map[string]interface{}, error) {
	if value.IsNull() || value.IsUnknown() || value.ValueString() == "" {
		return nil, nil
	}
	fields := map[string]interface{}{}
	if err := json.Unmarshal([]byte(value.ValueString()), &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

// parameterResolveComputed sets the computed attributes from the API after create or update.
func parameterResolveComputed(data *ParameterModel, parameter *Parameter) {
	if data.ID.IsUnknown() {
		data.ID = types.StringValue(parameter.ID)
	}
	if data.PrimaryField.IsUnknown() {
		data.PrimaryField = types.StringValue(parameter.PrimaryField)
	}
	data.LastModifiedAt = types.StringValue(parameter.LastModifiedAt)
	data.PrivateFieldsLastModifiedAt = types.StringValue(parameter.PrivateFieldsLastModifiedAt)
}

// parameterTimestampLayouts are the timestamp formats the API may use (RFC 3339 variants).
var parameterTimestampLayouts = []string{
	time.RFC3339Nano,
	"2006-01-02T15:04:05.999999999Z0700",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
}

// parameterParseTimestamp parses a timestamp in one of parameterTimestampLayouts. Timestamps
// without a zone are read as UTC.
func parameterParseTimestamp(value string) (time.Time, bool) {
	for _, layout := range parameterTimestampLayouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parameterTimestampsEqual reports whether two timestamps denote the same instant, so a different
// precision or zone notation of the same timestamp is not treated as a change. Values that cannot be
// parsed are compared as strings.
func parameterTimestampsEqual(a, b string) bool {
	if a == b {
		return true
	}
	at, aok := parameterParseTimestamp(a)
	bt, bok := parameterParseTimestamp(b)
	return aok && bok && at.Equal(bt)
}

// setParameterState refreshes the model from the API. The private fields are not returned, so the
// state value is kept unless they were changed outside Terraform, which is detected by a changed
// private_fields_last_modified_at: the value is then cleared so the next plan sets it again.
func setParameterState(data *ParameterModel, parameter *Parameter) {
	priorPrivateModified := data.PrivateFieldsLastModifiedAt
	data.ID = types.StringValue(parameter.ID)
	data.Name = types.StringValue(parameter.Name)
	data.Description = optionalStringState(data.Description, parameter.Description)
	data.Type = types.StringValue(parameter.Type)
	data.OwnerID = types.StringValue(parameter.OwnerID)
	data.PublicFieldsJSON = jsonStringState(data.PublicFieldsJSON, parameter.PublicFields)
	data.PrimaryField = types.StringValue(parameter.PrimaryField)
	data.LastModifiedAt = types.StringValue(parameter.LastModifiedAt)
	data.PrivateFieldsLastModifiedAt = types.StringValue(parameter.PrivateFieldsLastModifiedAt)
	if !priorPrivateModified.IsNull() && !priorPrivateModified.IsUnknown() {
		if parameterTimestampsEqual(priorPrivateModified.ValueString(), parameter.PrivateFieldsLastModifiedAt) {
			data.PrivateFieldsLastModifiedAt = priorPrivateModified
		} else {
			data.PrivateFields = types.StringNull()
		}
	}
}

// parameterReadBack reads a parameter again after create or update, so the state holds the
// timestamps in the form GET returns them, which Read compares with. When the read fails, the
// write response is used and a warning is reported.
func parameterReadBack(ctx context.Context, client *Client, written *Parameter, diags *diag.Diagnostics) *Parameter {
	if written == nil || written.ID == "" {
		return written
	}
	parameter, err := client.GetParameter(ctx, written.ID)
	if err != nil {
		diags.AddWarning("Unable to read parameter", fmt.Sprintf("The parameter was written, but reading it back failed; the timestamps of the write response are used: %s", err))
		return written
	}
	return parameter
}

// parameterPatchOps returns JSON Patch operations for the changed attributes. The type cannot be
// changed and forces replacement.
func parameterPatchOps(plan, state ParameterModel) ([]jsonPatchOp, error) {
	var b patchBuilder
	b.replaceIfChanged(plan.Name, state.Name, "/name", plan.Name.ValueString())
	b.replaceOrRemoveIfChanged(plan.Description, state.Description, "/description", plan.Description.ValueString())
	b.replaceIfChanged(plan.OwnerID, state.OwnerID, "/ownerId", plan.OwnerID.ValueString())
	if !plan.PublicFieldsJSON.Equal(state.PublicFieldsJSON) {
		fields, err := parameterPublicFields(plan.PublicFieldsJSON)
		if err != nil {
			return nil, err
		}
		if fields == nil {
			fields = map[string]interface{}{}
		}
		b.ops = append(b.ops, jsonPatchOp{Op: "replace", Path: "/publicFields", Value: fields})
	}
	if !plan.PrivateFields.IsNull() {
		b.replaceIfChanged(plan.PrivateFields, state.PrivateFields, "/privateFields", plan.PrivateFields.ValueString())
	}
	return b.ops, nil
}

func (r *ParameterResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ParameterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	publicFields, err := parameterPublicFields(data.PublicFieldsJSON)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("public_fields_json"), "Invalid JSON", err.Error())
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateParameter(ctx, &ParameterRequest{
		OwnerID:       data.OwnerID.ValueString(),
		Name:          data.Name.ValueString(),
		Type:          data.Type.ValueString(),
		Description:   data.Description.ValueString(),
		PublicFields:  publicFields,
		PrivateFields: data.PrivateFields.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create parameter: %s", err))
		return
	}
	parameterResolveComputed(&data, parameterReadBack(ctx, client, created, &resp.Diagnostics))
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ParameterResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ParameterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	parameter, err := client.GetParameter(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read parameter: %s", err))
		return
	}
	setParameterState(&data, parameter)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ParameterResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data, state ParameterModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	ops, err := parameterPatchOps(data, state)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("public_fields_json"), "Invalid JSON", err.Error())
		return
	}
	updated := &Parameter{
		ID:                          state.ID.ValueString(),
		PrimaryField:                state.PrimaryField.ValueString(),
		LastModifiedAt:              state.LastModifiedAt.ValueString(),
		PrivateFieldsLastModifiedAt: state.PrivateFieldsLastModifiedAt.ValueString(),
	}
	if len(ops) > 0 {
		client, err := r.client.IdentityNowClient(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", err.Error())
			return
		}
		updated, err = client.PatchParameter(ctx, data.ID.ValueString(), ops)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update parameter: %s", err))
			return
		}
		if updated.ID == "" {
			updated.ID = state.ID.ValueString()
		}
		updated = parameterReadBack(ctx, client, updated, &resp.Diagnostics)
	}
	parameterResolveComputed(&data, updated)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ParameterResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ParameterModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteParameter(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete parameter, parameters that are still referenced cannot be deleted: %s", err))
	}
}

func (r *ParameterResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &ParameterDataSource{}
var _ datasource.DataSourceWithValidateConfig = &ParameterDataSource{}

func NewParameterDataSource() datasource.DataSource {
	return &ParameterDataSource{}
}

type ParameterDataSource struct {
	client *Config
}

type ParameterDataSourceModel struct {
	ID                          types.String `tfsdk:"id"`
	Name                        types.String `tfsdk:"name"`
	Description                 types.String `tfsdk:"description"`
	Type                        types.String `tfsdk:"type"`
	OwnerID                     types.String `tfsdk:"owner_id"`
	PublicFieldsJSON            types.String `tfsdk:"public_fields_json"`
	PrimaryField                types.String `tfsdk:"primary_field"`
	LastModifiedAt              types.String `tfsdk:"last_modified_at"`
	LastModifiedBy              types.String `tfsdk:"last_modified_by"`
	PrivateFieldsLastModifiedAt types.String `tfsdk:"private_fields_last_modified_at"`
	PrivateFieldsLastModifiedBy types.String `tfsdk:"private_fields_last_modified_by"`
}

func (d *ParameterDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_parameter"
}

func (d *ParameterDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	computed := func(description string) dsschema.StringAttribute {
		return dsschema.StringAttribute{MarkdownDescription: description, Computed: true}
	}
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a parameter in parameter storage by ID or name. Only public fields are returned.",
		Attributes: map[string]dsschema.Attribute{
			"id": dsschema.StringAttribute{
				MarkdownDescription: "Parameter ID. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"name": dsschema.StringAttribute{
				MarkdownDescription: "Parameter name. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"description":                     computed("Description of the parameter."),
			"type":                            computed("Parameter type."),
			"owner_id":                        computed("Identity ID of the parameter owner."),
			"public_fields_json":              computed("Public fields of the parameter as a JSON object."),
			"primary_field":                   computed("Name of the primary field in the public fields."),
			"last_modified_at":                computed("Date any field of the parameter was last changed."),
			"last_modified_by":                computed("ID of the user who last changed the parameter."),
			"private_fields_last_modified_at": computed("Date the private fields were last changed."),
			"private_fields_last_modified_by": computed("ID of the user who last changed the private fields."),
		},
	}
}

func (d *ParameterDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *ParameterDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *ParameterDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ParameterDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var parameter *Parameter
	if !data.ID.IsNull() {
		parameter, err = client.GetParameter(ctx, data.ID.ValueString())
	} else {
		parameter, err = client.GetParameterByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Parameter not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read parameter: %s", err))
		return
	}
	data.ID = types.StringValue(parameter.ID)
	data.Name = types.StringValue(parameter.Name)
	data.Description = stringValueOrNull(parameter.Description)
	data.Type = types.StringValue(parameter.Type)
	data.OwnerID = types.StringValue(parameter.OwnerID)
	data.PublicFieldsJSON = jsonStringState(types.StringNull(), parameter.PublicFields)
	data.PrimaryField = stringValueOrNull(parameter.PrimaryField)
	data.LastModifiedAt = stringValueOrNull(parameter.LastModifiedAt)
	data.LastModifiedBy = stringValueOrNull(parameter.LastModifiedBy)
	data.PrivateFieldsLastModifiedAt = stringValueOrNull(parameter.PrivateFieldsLastModifiedAt)
	data.PrivateFieldsLastModifiedBy = stringValueOrNull(parameter.PrivateFieldsLastModifiedBy)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
