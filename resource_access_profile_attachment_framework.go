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

var _ resource.Resource = &AccessProfileAttachmentResource{}
var _ resource.ResourceWithImportState = &AccessProfileAttachmentResource{}

func NewAccessProfileAttachmentResource() resource.Resource {
	return &AccessProfileAttachmentResource{}
}

type AccessProfileAttachmentResource struct {
	client *Config
}

type AccessProfileAttachmentResourceModel struct {
	ID             types.String `tfsdk:"id"`
	SourceAppID    types.String `tfsdk:"source_app_id"`
	AccessProfiles types.List   `tfsdk:"access_profiles"`
}

func (r *AccessProfileAttachmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_profile_attachment"
}

func (r *AccessProfileAttachmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Access Profile Attachment resource - attaches access profiles to a source app",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Access Profile Attachment ID (same as source_app_id)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_app_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source App ID",
			},
			"access_profiles": schema.ListAttribute{
				Required:            true,
				MarkdownDescription: "List of access profile IDs to attach",
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *AccessProfileAttachmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AccessProfileAttachmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccessProfileAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var accessProfiles []string
	resp.Diagnostics.Append(data.AccessProfiles.ElementsAs(ctx, &accessProfiles, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attachment := &AccessProfileAttachment{
		SourceAppId:    data.SourceAppID.ValueString(),
		AccessProfiles: accessProfiles,
	}

	tflog.Info(ctx, "Creating Access Profile Attachment", map[string]interface{}{"source_app_id": attachment.SourceAppId})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	newAttachment, err := client.UpdateAccessProfileAttachment(ctx, attachment, attachment.SourceAppId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create access profile attachment: %s", err))
		return
	}

	data.ID = types.StringValue(newAttachment.SourceAppId)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessProfileAttachmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccessProfileAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Access Profile Attachment", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	attachment, err := client.GetAccessProfileAttachment(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read access profile attachment: %s", err))
		return
	}

	data.SourceAppID = types.StringValue(attachment.SourceAppId)

	apList, diags := types.ListValueFrom(ctx, types.StringType, attachment.AccessProfiles)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.AccessProfiles = apList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessProfileAttachmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AccessProfileAttachmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating Access Profile Attachment", map[string]interface{}{"id": data.ID.ValueString()})

	var accessProfiles []string
	resp.Diagnostics.Append(data.AccessProfiles.ElementsAs(ctx, &accessProfiles, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	attachment := &AccessProfileAttachment{
		SourceAppId:    data.SourceAppID.ValueString(),
		AccessProfiles: accessProfiles,
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	_, err = client.UpdateAccessProfileAttachment(ctx, attachment, attachment.SourceAppId)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update access profile attachment: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessProfileAttachmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccessProfileAttachmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting Access Profile Attachment", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	attachment, err := client.GetAccessProfileAttachment(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get access profile attachment: %s", err))
		return
	}

	err = client.DeleteAccessProfileAttachment(ctx, attachment)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete access profile attachment: %s", err))
		return
	}
}

func (r *AccessProfileAttachmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
