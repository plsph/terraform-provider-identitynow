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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func init() {
	registerResource(NewLauncherResource)
	registerDataSource(NewLauncherDataSource)
}

const (
	// launcherDefaultType is the only launcher type.
	launcherDefaultType = "INTERACTIVE_PROCESS"
	// launcherDefaultReferenceType is the only launcher reference type.
	launcherDefaultReferenceType = "WORKFLOW"
)

// Launcher is a launcher as returned by the /v2026/launchers API. Config is a JSON document encoded
// as a string.
type Launcher struct {
	ID          string             `json:"id,omitempty"`
	Created     string             `json:"created,omitempty"`
	Modified    string             `json:"modified,omitempty"`
	Owner       *LauncherOwner     `json:"owner,omitempty"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Type        string             `json:"type"`
	Disabled    bool               `json:"disabled"`
	Reference   *LauncherReference `json:"reference,omitempty"`
	Config      string             `json:"config"`
}

// LauncherOwner is the owner of a launcher.
type LauncherOwner struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

// LauncherReference is the object a launcher starts, a workflow.
type LauncherReference struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

func (c *Client) GetLauncher(ctx context.Context, id string) (*Launcher, error) {
	var launcher Launcher
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/launchers/%s", id), nil, &launcher); err != nil {
		return nil, err
	}
	return &launcher, nil
}

func (c *Client) CreateLauncher(ctx context.Context, launcher *Launcher) (*Launcher, error) {
	var created Launcher
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/launchers", launcher, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateLauncher(ctx context.Context, id string, launcher *Launcher) (*Launcher, error) {
	var updated Launcher
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/launchers/%s", id), launcher, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteLauncher(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/launchers/%s", id), nil, nil)
}

var _ resource.Resource = &LauncherResource{}
var _ resource.ResourceWithImportState = &LauncherResource{}

func NewLauncherResource() resource.Resource {
	return &LauncherResource{}
}

type LauncherResource struct {
	client *Config
}

type LauncherModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
	Disabled    types.Bool   `tfsdk:"disabled"`
	Reference   types.List   `tfsdk:"reference"`
	ConfigJSON  types.String `tfsdk:"config_json"`
	Owner       types.List   `tfsdk:"owner"`
	Created     types.String `tfsdk:"created"`
	Modified    types.String `tfsdk:"modified"`
}

type LauncherReferenceModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
}

var launcherReferenceObjectType = types.ObjectType{AttrTypes: map[string]attr.Type{
	"id":   types.StringType,
	"type": types.StringType,
}}

func (r *LauncherResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_launcher"
}

func (r *LauncherResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a launcher, which lets users start an interactive process such as a workflow.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Launcher ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the launcher, at most 255 characters.",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Description of the launcher, at most 2000 characters.",
				Required:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Launcher type. Defaults to `INTERACTIVE_PROCESS`, the only supported type.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString(launcherDefaultType),
			},
			"disabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the launcher is disabled. Defaults to `false`.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"config_json": schema.StringAttribute{
				MarkdownDescription: "Launcher configuration as a JSON object of at most 4 KB, e.g. `jsonencode({ workflowId = \"...\" })`. Compared semantically.",
				Required:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"owner": schema.ListNestedAttribute{
				MarkdownDescription: "Owner of the launcher, the identity that created it.",
				Computed:            true,
				PlanModifiers:       []planmodifier.List{listplanmodifier.UseStateForUnknown()},
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":   schema.StringAttribute{MarkdownDescription: "Owner ID.", Computed: true},
					"type": schema.StringAttribute{MarkdownDescription: "Owner type.", Computed: true},
				}},
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
			"reference": schema.ListNestedBlock{
				MarkdownDescription: "Object the launcher starts. At most one block.",
				Validators:          []validator.List{listSizeBetween(0, 1)},
				NestedObject: schema.NestedBlockObject{Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						MarkdownDescription: "ID of the referenced object, e.g. a workflow ID.",
						Required:            true,
					},
					"type": schema.StringAttribute{
						MarkdownDescription: "Type of the referenced object. Defaults to `WORKFLOW`, the only supported type.",
						Optional:            true,
						Computed:            true,
					},
				}},
			},
		},
	}
}

func (r *LauncherResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// launcherFromModel builds the create and replace request from the plan. Every writable field is
// modeled, so the full replacement semantics of PUT do not clear unmanaged configuration.
func launcherFromModel(ctx context.Context, data *LauncherModel, diags *diag.Diagnostics) *Launcher {
	data.Reference = listWithDefaultString(ctx, data.Reference, "type", launcherDefaultReferenceType, diags)
	launcher := &Launcher{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
		Type:        data.Type.ValueString(),
		Disabled:    data.Disabled.ValueBool(),
		Config:      data.ConfigJSON.ValueString(),
	}
	if !data.Reference.IsNull() && !data.Reference.IsUnknown() {
		var references []LauncherReferenceModel
		diags.Append(data.Reference.ElementsAs(ctx, &references, false)...)
		if len(references) > 0 {
			launcher.Reference = &LauncherReference{ID: references[0].ID.ValueString(), Type: references[0].Type.ValueString()}
		}
	}
	return launcher
}

// launcherConfigState keeps a prior config that is semantically equal to the API config.
func launcherConfigState(prior types.String, config string) types.String {
	if !prior.IsNull() && !prior.IsUnknown() && jsonSemanticallyEqual([]byte(prior.ValueString()), []byte(config)) {
		return prior
	}
	return types.StringValue(config)
}

func launcherOwnerState(ctx context.Context, owner *LauncherOwner, diags *diag.Diagnostics) types.List {
	ownerType := types.ObjectType{AttrTypes: map[string]attr.Type{"id": types.StringType, "type": types.StringType}}
	if owner == nil {
		return types.ListNull(ownerType)
	}
	list, d := types.ListValueFrom(ctx, ownerType, []LauncherReferenceModel{{ID: types.StringValue(owner.ID), Type: types.StringValue(owner.Type)}})
	diags.Append(d...)
	return list
}

func launcherReferenceState(ctx context.Context, reference *LauncherReference, diags *diag.Diagnostics) types.List {
	if reference == nil || reference.ID == "" {
		return types.ListNull(launcherReferenceObjectType)
	}
	referenceType := reference.Type
	if referenceType == "" {
		referenceType = launcherDefaultReferenceType
	}
	list, d := types.ListValueFrom(ctx, launcherReferenceObjectType, []LauncherReferenceModel{{ID: types.StringValue(reference.ID), Type: types.StringValue(referenceType)}})
	diags.Append(d...)
	return list
}

// launcherResolveComputed sets the computed attributes after create or update.
func launcherResolveComputed(ctx context.Context, data *LauncherModel, launcher *Launcher, diags *diag.Diagnostics) {
	if data.ID.IsUnknown() {
		data.ID = types.StringValue(launcher.ID)
	}
	if data.Owner.IsUnknown() {
		data.Owner = launcherOwnerState(ctx, launcher.Owner, diags)
	}
	if data.Created.IsUnknown() {
		data.Created = types.StringValue(launcher.Created)
	}
	data.Modified = types.StringValue(launcher.Modified)
}

// setLauncherState refreshes the model from the API.
func setLauncherState(ctx context.Context, data *LauncherModel, launcher *Launcher, diags *diag.Diagnostics) {
	data.ID = types.StringValue(launcher.ID)
	data.Name = types.StringValue(launcher.Name)
	data.Description = types.StringValue(launcher.Description)
	data.Type = types.StringValue(launcher.Type)
	data.Disabled = types.BoolValue(launcher.Disabled)
	data.Reference = launcherReferenceState(ctx, launcher.Reference, diags)
	data.ConfigJSON = launcherConfigState(data.ConfigJSON, launcher.Config)
	data.Owner = launcherOwnerState(ctx, launcher.Owner, diags)
	data.Created = types.StringValue(launcher.Created)
	data.Modified = types.StringValue(launcher.Modified)
}

func (r *LauncherResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data LauncherModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	launcher := launcherFromModel(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateLauncher(ctx, launcher)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create launcher: %s", err))
		return
	}
	launcherResolveComputed(ctx, &data, created, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LauncherResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data LauncherModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	launcher, err := client.GetLauncher(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read launcher: %s", err))
		return
	}
	setLauncherState(ctx, &data, launcher, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LauncherResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data LauncherModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	launcher := launcherFromModel(ctx, &data, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.UpdateLauncher(ctx, data.ID.ValueString(), launcher)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update launcher: %s", err))
		return
	}
	launcherResolveComputed(ctx, &data, updated, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *LauncherResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data LauncherModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteLauncher(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete launcher: %s", err))
	}
}

func (r *LauncherResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

var _ datasource.DataSource = &LauncherDataSource{}

func NewLauncherDataSource() datasource.DataSource {
	return &LauncherDataSource{}
}

type LauncherDataSource struct {
	client *Config
}

func (d *LauncherDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_launcher"
}

func (d *LauncherDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	idType := func(idDescription, typeDescription string) dsschema.NestedAttributeObject {
		return dsschema.NestedAttributeObject{Attributes: map[string]dsschema.Attribute{
			"id":   dsschema.StringAttribute{MarkdownDescription: idDescription, Computed: true},
			"type": dsschema.StringAttribute{MarkdownDescription: typeDescription, Computed: true},
		}}
	}
	resp.Schema = dsschema.Schema{
		MarkdownDescription: "Looks up a launcher by ID.",
		Attributes: map[string]dsschema.Attribute{
			"id":          dsschema.StringAttribute{MarkdownDescription: "Launcher ID.", Required: true},
			"name":        dsschema.StringAttribute{MarkdownDescription: "Name of the launcher.", Computed: true},
			"description": dsschema.StringAttribute{MarkdownDescription: "Description of the launcher.", Computed: true},
			"type":        dsschema.StringAttribute{MarkdownDescription: "Launcher type.", Computed: true},
			"disabled":    dsschema.BoolAttribute{MarkdownDescription: "Whether the launcher is disabled.", Computed: true},
			"reference": dsschema.ListNestedAttribute{
				MarkdownDescription: "Object the launcher starts.",
				Computed:            true,
				NestedObject:        idType("ID of the referenced object.", "Type of the referenced object."),
			},
			"config_json": dsschema.StringAttribute{MarkdownDescription: "Launcher configuration as a JSON object.", Computed: true},
			"owner": dsschema.ListNestedAttribute{
				MarkdownDescription: "Owner of the launcher.",
				Computed:            true,
				NestedObject:        idType("Owner ID.", "Owner type."),
			},
			"created":  dsschema.StringAttribute{MarkdownDescription: "Creation date.", Computed: true},
			"modified": dsschema.StringAttribute{MarkdownDescription: "Last modification date.", Computed: true},
		},
	}
}

func (d *LauncherDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *LauncherDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data LauncherModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	launcher, err := client.GetLauncher(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("id"), "Launcher not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read launcher: %s", err))
		return
	}
	setLauncherState(ctx, &data, launcher, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
