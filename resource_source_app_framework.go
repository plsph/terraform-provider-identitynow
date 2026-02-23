package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
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
				MarkdownDescription: "Whether the source app is enabled",
			},
			"match_all_accounts": schema.BoolAttribute{
				Optional:            true,
				MarkdownDescription: "Whether to match all accounts",
			},
		},
		Blocks: map[string]schema.Block{
			"source": schema.ListNestedBlock{
				MarkdownDescription: "Account source for the source app",
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
							MarkdownDescription: "Source type",
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

	if !data.Enabled.IsNull() {
		v := data.Enabled.ValueBool()
		sa.Enabled = &v
	}

	if !data.MatchAllAccounts.IsNull() {
		v := data.MatchAllAccounts.ValueBool()
		sa.MatchAllAccounts = &v
	}

	var sources []SourceAppSourceModel
	resp.Diagnostics.Append(data.Source.ElementsAs(ctx, &sources, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(sources) > 0 {
		sa.SourceAppSource = &ObjectInfo{
			ID:   sources[0].ID.ValueString(),
			Name: sources[0].Name.ValueString(),
			Type: sources[0].Type.ValueString(),
		}
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
		if _, notFound := err.(*NotFoundError); notFound {
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

	if !data.Enabled.IsNull() {
		updatePatches = append(updatePatches, &UpdateSourceApp{Op: "replace", Path: "/enabled", Value: data.Enabled.ValueBool()})
	}

	if !data.MatchAllAccounts.IsNull() {
		updatePatches = append(updatePatches, &UpdateSourceApp{Op: "replace", Path: "/matchAllAccounts", Value: data.MatchAllAccounts.ValueBool()})
	}

	_, err = client.UpdateSourceApp(ctx, updatePatches, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update source app: %s", err))
		return
	}

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
		if _, notFound := err.(*NotFoundError); notFound {
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
