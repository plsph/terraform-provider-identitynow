package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &SourceAppResource{}
var _ resource.ResourceWithImportState = &SourceAppResource{}

func NewSourceAppResource() resource.Resource {
	return &SourceAppResource{}
}

type SourceAppResource struct {
	client *Config
}

type SourceAppResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	Enabled          types.Bool   `tfsdk:"enabled"`
	MatchAllAccounts types.Bool   `tfsdk:"match_all_accounts"`
	Source           types.List   `tfsdk:"source"`
}

type SourceAppSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func (r *SourceAppResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source_app"
}

func (r *SourceAppResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Source App resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source App ID",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source App name",
			},
			"description": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source App description",
			},
			"enabled": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether the source app is enabled",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"match_all_accounts": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether to match all accounts",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"source": schema.ListNestedBlock{
				MarkdownDescription: "Account source for the source app",
				Validators:          []validator.List{listSizeBetween(0, 1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Source ID",
						},
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Source name",
						},
						"type": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Source type, defaults to SOURCE",
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
					},
				},
			},
		},
	}
}

func (r *SourceAppResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SourceAppResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SourceAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	sa := &SourceApp{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	sa.Enabled = boolPointer(data.Enabled)
	sa.MatchAllAccounts = boolPointer(data.MatchAllAccounts)
	data.Source = sourceAppSourceWithType(ctx, data.Source, &resp.Diagnostics)
	sa.SourceAppSource = sourceAppSourceValue(ctx, data.Source, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Creating Source App", map[string]interface{}{"name": sa.Name})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	newSA, err := client.CreateSourceApp(ctx, sa)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create source app: %s", err))
		return
	}

	data.ID = types.StringValue(newSA.ID)
	data.Enabled = computedBoolFromAPI(data.Enabled, newSA.Enabled)
	data.MatchAllAccounts = computedBoolFromAPI(data.MatchAllAccounts, newSA.MatchAllAccounts)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceAppResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SourceAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Source App", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	sa, err := client.GetSourceApp(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read source app: %s", err))
		return
	}

	data.Name = types.StringValue(sa.Name)
	data.Description = types.StringValue(sa.Description)
	if sa.Enabled != nil {
		data.Enabled = types.BoolValue(*sa.Enabled)
	}
	if sa.MatchAllAccounts != nil {
		data.MatchAllAccounts = types.BoolValue(*sa.MatchAllAccounts)
	}

	sourceObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"name": types.StringType,
		"type": types.StringType,
	}}
	if sa.SourceAppSource != nil {
		sourceType := sa.SourceAppSource.Type
		if sourceType == "" {
			sourceType = "SOURCE"
		}
		sourceModels := []SourceAppSourceModel{
			{
				ID:   types.StringValue(fmt.Sprintf("%v", sa.SourceAppSource.ID)),
				Name: types.StringValue(sa.SourceAppSource.Name),
				Type: types.StringValue(sourceType),
			},
		}
		sourceList, diags := types.ListValueFrom(ctx, sourceObjType, sourceModels)
		resp.Diagnostics.Append(diags...)
		data.Source = sourceList
	} else {
		data.Source, _ = types.ListValue(sourceObjType, []attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceAppResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SourceAppResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating Source App", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	updatePatches := []*UpdateSourceApp{
		{Op: "replace", Path: "/name", Value: data.Name.ValueString()},
		{Op: "replace", Path: "/description", Value: data.Description.ValueString()},
	}

	if enabled := boolPointer(data.Enabled); enabled != nil {
		updatePatches = append(updatePatches, &UpdateSourceApp{Op: "replace", Path: "/enabled", Value: *enabled})
	}

	if matchAll := boolPointer(data.MatchAllAccounts); matchAll != nil {
		updatePatches = append(updatePatches, &UpdateSourceApp{Op: "replace", Path: "/matchAllAccounts", Value: *matchAll})
	}

	data.Source = sourceAppSourceWithType(ctx, data.Source, &resp.Diagnostics)
	if source := sourceAppSourceValue(ctx, data.Source, &resp.Diagnostics); source != nil {
		updatePatches = append(updatePatches, &UpdateSourceApp{Op: "replace", Path: "/accountSource", Value: source})
	}
	if resp.Diagnostics.HasError() {
		return
	}

	updated, err := client.UpdateSourceApp(ctx, updatePatches, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update source app: %s", err))
		return
	}
	data.Enabled = computedBoolFromAPI(data.Enabled, updated.Enabled)
	data.MatchAllAccounts = computedBoolFromAPI(data.MatchAllAccounts, updated.MatchAllAccounts)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceAppResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SourceAppResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting Source App", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	sa, err := client.GetSourceApp(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get source app: %s", err))
		return
	}

	err = client.DeleteSourceApp(ctx, sa)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete source app: %s", err))
		return
	}
}

func (r *SourceAppResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// boolPointer returns a pointer to a known boolean value, or nil for null and unknown values.
func boolPointer(value types.Bool) *bool {
	if value.IsNull() || value.IsUnknown() {
		return nil
	}
	v := value.ValueBool()
	return &v
}

// sourceAppSourceWithType resolves an unknown source type to SOURCE, the only account source type.
func sourceAppSourceWithType(ctx context.Context, list types.List, diags *diag.Diagnostics) types.List {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return list
	}
	var sources []SourceAppSourceModel
	diags.Append(list.ElementsAs(ctx, &sources, false)...)
	for i := range sources {
		if sources[i].Type.IsUnknown() || sources[i].Type.IsNull() {
			sources[i].Type = types.StringValue("SOURCE")
		}
	}
	result, d := types.ListValueFrom(ctx, list.ElementType(ctx), sources)
	diags.Append(d...)
	return result
}

func sourceAppSourceValue(ctx context.Context, list types.List, diags *diag.Diagnostics) *ObjectInfo {
	if list.IsNull() || list.IsUnknown() || len(list.Elements()) == 0 {
		return nil
	}
	var sources []SourceAppSourceModel
	diags.Append(list.ElementsAs(ctx, &sources, false)...)
	if len(sources) == 0 {
		return nil
	}
	return &ObjectInfo{
		ID:   sources[0].ID.ValueString(),
		Name: sources[0].Name.ValueString(),
		Type: sources[0].Type.ValueString(),
	}
}
