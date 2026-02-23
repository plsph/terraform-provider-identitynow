package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &AccessProfileResource{}
var _ resource.ResourceWithImportState = &AccessProfileResource{}

func NewAccessProfileResource() resource.Resource {
	return &AccessProfileResource{}
}

type AccessProfileResource struct {
	client *Config
}

type AccessProfileResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Name                types.String `tfsdk:"name"`
	Description         types.String `tfsdk:"description"`
	Owner               types.List   `tfsdk:"owner"`
	Source              types.List   `tfsdk:"source"`
	Entitlements        types.List   `tfsdk:"entitlements"`
	AccessRequestConfig types.List   `tfsdk:"access_request_config"`
	Enabled             types.Bool   `tfsdk:"enabled"`
	Requestable         types.Bool   `tfsdk:"requestable"`
}

type EntitlementRefModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

type AccessRequestConfigModel struct {
	CommentsRequired       types.Bool `tfsdk:"comments_required"`
	DenialCommentsRequired types.Bool `tfsdk:"denial_comments_required"`
	ReauthorizationRequired types.Bool `tfsdk:"reauthorization_required"`
	RequireEndDate         types.Bool `tfsdk:"require_end_date"`
	ApprovalSchemes        types.List `tfsdk:"approval_schemes"`
	MaxPermittedAccessDuration types.List `tfsdk:"max_permitted_access_duration"`
}

type ApprovalSchemeModel struct {
	ApproverType types.String `tfsdk:"approver_type"`
	ApproverID   types.String `tfsdk:"approver_id"`
}

type MaxPermittedAccessDurationModel struct {
	Value    types.Int64  `tfsdk:"value"`
	TimeUnit types.String `tfsdk:"time_unit"`
}

func (r *AccessProfileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_profile"
}

func (r *AccessProfileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Access Profile resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required: true,
			},
			"description": schema.StringAttribute{
				Required: true,
			},
			"enabled": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
			"requestable": schema.BoolAttribute{
				Optional: true,
				Computed: true,
			},
		},
		Blocks: map[string]schema.Block{
			"owner": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Required: true},
						"type": schema.StringAttribute{Required: true},
						"name": schema.StringAttribute{Required: true},
					},
				},
			},
			"source": schema.ListNestedBlock{
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Required: true},
						"type": schema.StringAttribute{Required: true},
						"name": schema.StringAttribute{Required: true},
					},
				},
			},
			"entitlements": schema.ListNestedBlock{
				MarkdownDescription: "Entitlements assigned to this access profile",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Entitlement ID",
						},
						"name": schema.StringAttribute{
							Required:            true,
							MarkdownDescription: "Entitlement name",
						},
						"type": schema.StringAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Entitlement type",
						},
					},
				},
			},
			"access_request_config": schema.ListNestedBlock{
				MarkdownDescription: "Access request configuration",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"comments_required": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "If comment is required",
						},
						"denial_comments_required": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "If denial comment is required",
						},
						"reauthorization_required": schema.BoolAttribute{
							Optional:            true,
							MarkdownDescription: "Indicates whether reauthorization is required",
						},
						"require_end_date": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							MarkdownDescription: "Indicates whether the requester must provide access end date",
						},
					},
					Blocks: map[string]schema.Block{
						"approval_schemes": schema.ListNestedBlock{
							MarkdownDescription: "Approval schemes",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"approver_type": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "Type of approver",
									},
									"approver_id": schema.StringAttribute{
										Optional:            true,
										MarkdownDescription: "Id of approver",
									},
								},
							},
						},
						"max_permitted_access_duration": schema.ListNestedBlock{
							MarkdownDescription: "Max permitted access duration",
							NestedObject: schema.NestedBlockObject{
								Attributes: map[string]schema.Attribute{
									"value": schema.Int64Attribute{
										Required:            true,
										MarkdownDescription: "The numeric value representing the amount of time",
									},
									"time_unit": schema.StringAttribute{
										Required:            true,
										MarkdownDescription: "The unit of time",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (r *AccessProfileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AccessProfileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AccessProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ap := &AccessProfile{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	var owners []OwnerModel
	resp.Diagnostics.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(owners) > 0 {
		ap.AccessProfileOwner = &ObjectInfo{
			ID:   owners[0].ID.ValueString(),
			Type: owners[0].Type.ValueString(),
			Name: owners[0].Name.ValueString(),
		}
	}

	var sources []OwnerModel
	resp.Diagnostics.Append(data.Source.ElementsAs(ctx, &sources, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(sources) > 0 {
		ap.AccessProfileSource = &ObjectInfo{
			ID:   sources[0].ID.ValueString(),
			Type: sources[0].Type.ValueString(),
			Name: sources[0].Name.ValueString(),
		}
	}

	if !data.Enabled.IsNull() {
		enabled := data.Enabled.ValueBool()
		ap.Enabled = &enabled
	}

	if !data.Requestable.IsNull() {
		requestable := data.Requestable.ValueBool()
		ap.Requestable = &requestable
	}

	// Entitlements
	if !data.Entitlements.IsNull() && len(data.Entitlements.Elements()) > 0 {
		var entModels []EntitlementRefModel
		resp.Diagnostics.Append(data.Entitlements.ElementsAs(ctx, &entModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		for _, em := range entModels {
			ent := &ObjectInfo{
				ID:   em.ID.ValueString(),
				Name: em.Name.ValueString(),
			}
			if !em.Type.IsNull() && em.Type.ValueString() != "" {
				ent.Type = em.Type.ValueString()
			} else {
				ent.Type = "ENTITLEMENT"
			}
			ap.Entitlements = append(ap.Entitlements, ent)
		}
	}

	// Access Request Config
	if !data.AccessRequestConfig.IsNull() && len(data.AccessRequestConfig.Elements()) > 0 {
		var arcModels []AccessRequestConfigModel
		resp.Diagnostics.Append(data.AccessRequestConfig.ElementsAs(ctx, &arcModels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(arcModels) > 0 {
			arc := arcModels[0]
			config := &AccessRequestConfigList{}
			if !arc.CommentsRequired.IsNull() {
				config.CommentsRequired = arc.CommentsRequired.ValueBool()
			}
			if !arc.DenialCommentsRequired.IsNull() {
				config.DenialCommentsRequired = arc.DenialCommentsRequired.ValueBool()
			}
			if !arc.ReauthorizationRequired.IsNull() {
				config.ReauthorizationRequired = arc.ReauthorizationRequired.ValueBool()
			}
			if !arc.RequireEndDate.IsNull() {
				config.RequireEndDate = arc.RequireEndDate.ValueBool()
			}
			if !arc.ApprovalSchemes.IsNull() && len(arc.ApprovalSchemes.Elements()) > 0 {
				var schemes []ApprovalSchemeModel
				resp.Diagnostics.Append(arc.ApprovalSchemes.ElementsAs(ctx, &schemes, false)...)
				if resp.Diagnostics.HasError() {
					return
				}
				for _, s := range schemes {
					config.ApprovalSchemes = append(config.ApprovalSchemes, &ApprovalSchemes{
						ApproverType: s.ApproverType.ValueString(),
						ApproverId:   s.ApproverID.ValueString(),
					})
				}
			}
			if !arc.MaxPermittedAccessDuration.IsNull() && len(arc.MaxPermittedAccessDuration.Elements()) > 0 {
				var durModels []MaxPermittedAccessDurationModel
				resp.Diagnostics.Append(arc.MaxPermittedAccessDuration.ElementsAs(ctx, &durModels, false)...)
				if resp.Diagnostics.HasError() {
					return
				}
				if len(durModels) > 0 {
					config.MaxPermittedAccessDuration = &MaxPermittedAccessDuration{
						Value:    int(durModels[0].Value.ValueInt64()),
						TimeUnit: durModels[0].TimeUnit.ValueString(),
					}
				}
			}
			ap.AccessRequestConfig = config
		}
	}

	tflog.Info(ctx, "Creating Access Profile", map[string]interface{}{"name": ap.Name})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	newAP, err := client.CreateAccessProfile(ctx, ap)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(newAP.ID)
	if newAP.Enabled != nil {
		data.Enabled = types.BoolValue(*newAP.Enabled)
	}
	if newAP.Requestable != nil {
		data.Requestable = types.BoolValue(*newAP.Requestable)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessProfileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AccessProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	ap, err := client.GetAccessProfile(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.Name = types.StringValue(ap.Name)
	data.Description = types.StringValue(ap.Description)
	if ap.Enabled != nil {
		data.Enabled = types.BoolValue(*ap.Enabled)
	}
	if ap.Requestable != nil {
		data.Requestable = types.BoolValue(*ap.Requestable)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessProfileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AccessProfileResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// Build update patches
	updatePatches := []*UpdateAccessProfile{
		{Op: "replace", Path: "/description", Value: data.Description.ValueString()},
	}

	_, err = client.UpdateAccessProfile(ctx, updatePatches, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *AccessProfileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AccessProfileResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	ap, err := client.GetAccessProfile(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	err = client.DeleteAccessProfile(ctx, ap)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
}

func (r *AccessProfileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
