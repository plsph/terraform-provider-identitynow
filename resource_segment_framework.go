package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SegmentResource{}
var _ resource.ResourceWithImportState = &SegmentResource{}

func NewSegmentResource() resource.Resource {
	return &SegmentResource{}
}

type SegmentResource struct {
	client *Config
}

type SegmentResourceModel struct {
	ID                     types.String `tfsdk:"id"`
	Name                   types.String `tfsdk:"name"`
	Description            types.String `tfsdk:"description"`
	Owner                  types.List   `tfsdk:"owner"`
	VisibilityCriteriaJSON types.String `tfsdk:"visibility_criteria_json"`
	Active                 types.Bool   `tfsdk:"active"`
	Created                types.String `tfsdk:"created"`
	Modified               types.String `tfsdk:"modified"`
}

func (r *SegmentResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_segment"
}

func (r *SegmentResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SailPoint identity segment.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The segment business name.",
			},
			"description": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "An optional description for the segment.",
			},
			"visibility_criteria_json": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Visibility criteria as a JSON object. The object must follow the SailPoint Visibility Criteria schema.",
			},
			"active": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				MarkdownDescription: "Whether the segment is active. Inactive segments have no effect.",
			},
			"created": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The segment creation timestamp.",
			},
			"modified": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The segment modification timestamp.",
			},
		},
		Blocks: map[string]schema.Block{
			"owner": schema.ListNestedBlock{
				MarkdownDescription: "The segment owner.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Required: true},
						"type": schema.StringAttribute{Required: true},
						"name": schema.StringAttribute{Required: true},
					},
				},
			},
		},
	}
}

func (r *SegmentResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func segmentVisibilityValue(raw types.String, diagnostics *diag.Diagnostics) interface{} {
	if raw.IsNull() || raw.IsUnknown() || raw.ValueString() == "" {
		return nil
	}
	var value interface{}
	if err := json.Unmarshal([]byte(raw.ValueString()), &value); err != nil {
		diagnostics.AddAttributeError(path.Root("visibility_criteria_json"), "Invalid visibility criteria JSON", err.Error())
		return nil
	}
	return value
}

func segmentOwnerValue(ctx context.Context, owner types.List, diagnostics *diag.Diagnostics) *ObjectInfo {
	if owner.IsNull() || owner.IsUnknown() {
		return nil
	}
	var owners []OwnerModel
	diagnostics.Append(owner.ElementsAs(ctx, &owners, false)...)
	if len(owners) == 0 {
		return nil
	}
	return &ObjectInfo{ID: owners[0].ID.ValueString(), Type: owners[0].Type.ValueString(), Name: owners[0].Name.ValueString()}
}

func (r *SegmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	segment := &Segment{
		Name:               data.Name.ValueString(),
		Description:        data.Description.ValueString(),
		Owner:              segmentOwnerValue(ctx, data.Owner, &resp.Diagnostics),
		VisibilityCriteria: segmentVisibilityValue(data.VisibilityCriteriaJSON, &resp.Diagnostics),
		Active:             data.Active.ValueBool(),
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	newSegment, err := client.CreateSegment(ctx, segment)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create segment: %s", err))
		return
	}
	data.ID = types.StringValue(newSegment.ID)
	data.Created = types.StringValue(newSegment.Created)
	data.Modified = types.StringValue(newSegment.Modified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SegmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	segment, err := client.GetSegment(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.Name = types.StringValue(segment.Name)
	data.Description = types.StringValue(segment.Description)
	data.Active = types.BoolValue(segment.Active)
	data.Created = types.StringValue(segment.Created)
	data.Modified = types.StringValue(segment.Modified)
	data.Owner = segmentOwnerState(ctx, segment.Owner, &resp.Diagnostics)
	if segment.VisibilityCriteria != nil {
		criteria, err := json.Marshal(segment.VisibilityCriteria)
		if err != nil {
			resp.Diagnostics.AddError("State Error", fmt.Sprintf("Unable to encode visibility criteria: %s", err))
			return
		}
		data.VisibilityCriteriaJSON = types.StringValue(string(criteria))
	} else {
		data.VisibilityCriteriaJSON = types.StringNull()
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func segmentOwnerState(ctx context.Context, owner *ObjectInfo, diagnostics *diag.Diagnostics) types.List {
	ownerType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id": types.StringType, "type": types.StringType, "name": types.StringType,
	}}
	if owner == nil {
		return types.ListNull(ownerType)
	}
	owners, diags := types.ListValueFrom(ctx, ownerType, []OwnerModel{{
		ID: types.StringValue(fmt.Sprintf("%v", owner.ID)), Type: types.StringValue(owner.Type), Name: types.StringValue(owner.Name),
	}})
	diagnostics.Append(diags...)
	return owners
}

func (r *SegmentResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	patches := []*UpdateSegment{
		{Op: "replace", Path: "/name", Value: data.Name.ValueString()},
		{Op: "replace", Path: "/description", Value: data.Description.ValueString()},
		{Op: "replace", Path: "/active", Value: data.Active.ValueBool()},
	}
	if owner := segmentOwnerValue(ctx, data.Owner, &resp.Diagnostics); owner != nil {
		patches = append(patches, &UpdateSegment{Op: "replace", Path: "/owner", Value: owner})
	}
	if criteria := segmentVisibilityValue(data.VisibilityCriteriaJSON, &resp.Diagnostics); criteria != nil {
		patches = append(patches, &UpdateSegment{Op: "replace", Path: "/visibilityCriteria", Value: criteria})
	}
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	updated, err := client.UpdateSegment(ctx, data.ID.ValueString(), patches)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update segment: %s", err))
		return
	}
	data.Created = types.StringValue(updated.Created)
	data.Modified = types.StringValue(updated.Modified)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SegmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SegmentResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteSegment(ctx, data.ID.ValueString()); err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete segment: %s", err))
	}
}

func (r *SegmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
