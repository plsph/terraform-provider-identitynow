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

var _ resource.Resource = &GovernanceGroupMembersResource{}
var _ resource.ResourceWithImportState = &GovernanceGroupMembersResource{}

func NewGovernanceGroupMembersResource() resource.Resource {
	return &GovernanceGroupMembersResource{}
}

type GovernanceGroupMembersResource struct {
	client *Config
}

type GovernanceGroupMembersResourceModel struct {
	ID                types.String `tfsdk:"id"`
	GovernanceGroupID types.String `tfsdk:"governance_group_id"`
	Members           types.List   `tfsdk:"members"`
}

type GovernanceGroupMemberModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func (r *GovernanceGroupMembersResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_governance_group_members"
}

func (r *GovernanceGroupMembersResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Governance Group Members resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Governance Group Members ID (same as governance_group_id)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"governance_group_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Governance Group ID",
			},
		},
		Blocks: map[string]schema.Block{
			"members": schema.ListNestedBlock{
				MarkdownDescription: "List of members",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Member ID",
						},
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Member name",
						},
						"type": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Member type",
						},
					},
				},
			},
		},
	}
}

func (r *GovernanceGroupMembersResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GovernanceGroupMembersResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GovernanceGroupMembersResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var members []GovernanceGroupMemberModel
	resp.Diagnostics.Append(data.Members.ElementsAs(ctx, &members, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ggMembers := &GovernanceGroupMembers{
		GovernanceGroupId: data.GovernanceGroupID.ValueString(),
	}
	for _, m := range members {
		ggMembers.GovernanceGroupMembersMembers = append(ggMembers.GovernanceGroupMembersMembers, &GovernanceGroupMembersMembers{
			ID:   m.ID.ValueString(),
			Name: m.Name.ValueString(),
			Type: m.Type.ValueString(),
		})
	}

	tflog.Info(ctx, "Creating Governance Group Members", map[string]interface{}{"governance_group_id": ggMembers.GovernanceGroupId})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	newGGMembers, err := client.CreateGovernanceGroupMembers(ctx, ggMembers, ggMembers.GovernanceGroupId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create governance group members: %s", err))
		return
	}

	data.ID = types.StringValue(newGGMembers.GovernanceGroupId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GovernanceGroupMembersResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GovernanceGroupMembersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Governance Group Members", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	ggMembers, err := client.GetGovernanceGroupMembers(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read governance group members: %s", err))
		return
	}

	data.GovernanceGroupID = types.StringValue(ggMembers.GovernanceGroupId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GovernanceGroupMembersResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GovernanceGroupMembersResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating Governance Group Members", map[string]interface{}{"id": data.ID.ValueString()})

	var members []GovernanceGroupMemberModel
	resp.Diagnostics.Append(data.Members.ElementsAs(ctx, &members, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ggMembers := &GovernanceGroupMembers{
		GovernanceGroupId: data.GovernanceGroupID.ValueString(),
	}
	for _, m := range members {
		ggMembers.GovernanceGroupMembersMembers = append(ggMembers.GovernanceGroupMembersMembers, &GovernanceGroupMembersMembers{
			ID:   m.ID.ValueString(),
			Name: m.Name.ValueString(),
			Type: m.Type.ValueString(),
		})
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	// Get current members for the update call
	currentMembers, err := client.GetGovernanceGroupMembers(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get current governance group members: %s", err))
		return
	}

	_, err = client.UpdateGovernanceGroupMembers(ctx, ggMembers, currentMembers, ggMembers.GovernanceGroupId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update governance group members: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *GovernanceGroupMembersResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GovernanceGroupMembersResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting Governance Group Members", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	ggMembers, err := client.GetGovernanceGroupMembers(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get governance group members: %s", err))
		return
	}

	err = client.DeleteGovernanceGroupMembers(ctx, ggMembers)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete governance group members: %s", err))
		return
	}
}

func (r *GovernanceGroupMembersResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
