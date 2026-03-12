package main

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &TaggedObjectResource{}
var _ resource.ResourceWithImportState = &TaggedObjectResource{}

func NewTaggedObjectResource() resource.Resource {
	return &TaggedObjectResource{}
}

type TaggedObjectResource struct {
	client *Config
}

type TaggedObjectResourceModel struct {
	ID         types.String                  `tfsdk:"id"`
	ObjectType types.String                  `tfsdk:"object_type"`
	ObjectIDs  types.Set                     `tfsdk:"object_ids"`
	Tags       CaseInsensitiveStringSetValue `tfsdk:"tags"`
}

func (r *TaggedObjectResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tagged_object"
}

func (r *TaggedObjectResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Tagged Object resource - manages tags on any SailPoint IdentityNow resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Tagged Object ID (composed as object_type/object_id1,object_id2,...)",
			},
			"object_type": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Type of the SailPoint object to tag (e.g. ACCESS_PROFILE, ROLE, SOURCE, IDENTITY, GOVERNANCE_GROUP, ENTITLEMENT, APPLICATION)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"object_ids": schema.SetAttribute{
				Required:            true,
				MarkdownDescription: "Set of IDs of the SailPoint objects to tag",
				ElementType:         types.StringType,
			},
			"tags": schema.SetAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Set of tags to apply to the objects (case-insensitive, stored as uppercase)",
				CustomType: CaseInsensitiveStringSetType{
					SetType: basetypes.SetType{ElemType: CaseInsensitiveStringType{}},
				},
				PlanModifiers: []planmodifier.Set{
					UseStateForCaseInsensitiveSet(),
				},
			},
		},
	}
}

func (r *TaggedObjectResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *TaggedObjectResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TaggedObjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tags []string
	resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	upperTags := toUpperSlice(tags)

	var objectIDs []string
	resp.Diagnostics.Append(data.ObjectIDs.ElementsAs(ctx, &objectIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	objectType := data.ObjectType.ValueString()

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	for _, objectID := range objectIDs {
		tflog.Info(ctx, "Creating Tagged Object", map[string]interface{}{
			"object_type": objectType,
			"object_id":   objectID,
			"tags":        upperTags,
		})

		taggedObject := &TaggedObject{
			ObjectRef: &TaggedObjectRef{
				Type: objectType,
				ID:   objectID,
			},
			Tags: upperTags,
		}

		_, err = client.SetTaggedObject(ctx, taggedObject)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to set tagged object %s/%s: %s", objectType, objectID, err))
			return
		}
	}

	data.Tags = stringSliceToCaseInsensitiveSet(ctx, upperTags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = types.StringValue(computeID(objectType, objectIDs))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TaggedObjectResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TaggedObjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var objectIDs []string
	resp.Diagnostics.Append(data.ObjectIDs.ElementsAs(ctx, &objectIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	objectType := data.ObjectType.ValueString()

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	// Read tags from the first object; all objects managed by this resource should have the same tags.
	var readTags []string
	for _, objectID := range objectIDs {
		tflog.Info(ctx, "Reading Tagged Object", map[string]interface{}{
			"object_type": objectType,
			"object_id":   objectID,
		})

		taggedObject, err := client.GetTaggedObject(ctx, objectType, objectID)
		if err != nil {
			if _, notFound := err.(*NotFoundError); notFound {
				resp.State.RemoveResource(ctx)
				return
			}
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read tagged object %s/%s: %s", objectType, objectID, err))
			return
		}

		if readTags == nil {
			readTags = taggedObject.Tags
		}
	}

	if readTags == nil {
		readTags = []string{}
	}

	data.Tags = stringSliceToCaseInsensitiveSet(ctx, readTags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = types.StringValue(computeID(objectType, objectIDs))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TaggedObjectResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TaggedObjectResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var tags []string
	resp.Diagnostics.Append(data.Tags.ElementsAs(ctx, &tags, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	upperTags := toUpperSlice(tags)

	var plannedIDs []string
	resp.Diagnostics.Append(data.ObjectIDs.ElementsAs(ctx, &plannedIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var stateData TaggedObjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var priorIDs []string
	resp.Diagnostics.Append(stateData.ObjectIDs.ElementsAs(ctx, &priorIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	objectType := data.ObjectType.ValueString()

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	// Set tags on all planned object IDs
	for _, objectID := range plannedIDs {
		tflog.Info(ctx, "Updating Tagged Object", map[string]interface{}{
			"object_type": objectType,
			"object_id":   objectID,
			"tags":        upperTags,
		})

		taggedObject := &TaggedObject{
			ObjectRef: &TaggedObjectRef{
				Type: objectType,
				ID:   objectID,
			},
			Tags: upperTags,
		}

		_, err = client.SetTaggedObject(ctx, taggedObject)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update tagged object %s/%s: %s", objectType, objectID, err))
			return
		}
	}

	// Delete tags from removed object IDs
	plannedSet := toStringSet(plannedIDs)
	for _, priorID := range priorIDs {
		if _, exists := plannedSet[priorID]; !exists {
			tflog.Info(ctx, "Deleting tags from removed object", map[string]interface{}{
				"object_type": objectType,
				"object_id":   priorID,
			})
			err = client.DeleteTaggedObject(ctx, objectType, priorID)
			if err != nil {
				if _, notFound := err.(*NotFoundError); !notFound {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete tagged object %s/%s: %s", objectType, priorID, err))
					return
				}
			}
		}
	}

	data.Tags = stringSliceToCaseInsensitiveSet(ctx, upperTags, &resp.Diagnostics)
	if resp.Diagnostics.HasError() {
		return
	}
	data.ID = types.StringValue(computeID(objectType, plannedIDs))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TaggedObjectResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TaggedObjectResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var objectIDs []string
	resp.Diagnostics.Append(data.ObjectIDs.ElementsAs(ctx, &objectIDs, false)...)
	if resp.Diagnostics.HasError() {
		return
	}

	objectType := data.ObjectType.ValueString()

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	for _, objectID := range objectIDs {
		tflog.Info(ctx, "Deleting Tagged Object", map[string]interface{}{
			"object_type": objectType,
			"object_id":   objectID,
		})

		_, err = client.GetTaggedObject(ctx, objectType, objectID)
		if err != nil {
			if _, notFound := err.(*NotFoundError); notFound {
				continue
			}
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get tagged object %s/%s: %s", objectType, objectID, err))
			return
		}

		err = client.DeleteTaggedObject(ctx, objectType, objectID)
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete tagged object %s/%s: %s", objectType, objectID, err))
			return
		}
	}
}

func (r *TaggedObjectResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID format: {object_type}/{object_id1},{object_id2},...
	parts := strings.SplitN(req.ID, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected import ID format: {object_type}/{object_id1},{object_id2},..., got: %s", req.ID),
		)
		return
	}

	objectType := parts[0]
	objectIDs := strings.Split(parts[1], ",")

	idElems := make([]attr.Value, len(objectIDs))
	for i, id := range objectIDs {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			resp.Diagnostics.AddError("Invalid Import ID", fmt.Sprintf("Empty object ID at position %d", i+1))
			return
		}
		idElems[i] = types.StringValue(trimmed)
	}

	idsSet, diags := types.SetValue(types.StringType, idElems)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_type"), objectType)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("object_ids"), idsSet)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

func stringSliceToCaseInsensitiveSet(ctx context.Context, tags []string, diags *diag.Diagnostics) CaseInsensitiveStringSetValue {
	elems := make([]attr.Value, len(tags))
	for i, t := range tags {
		elems[i] = CaseInsensitiveStringValue{StringValue: types.StringValue(t)}
	}
	setVal, d := basetypes.NewSetValue(CaseInsensitiveStringType{}, elems)
	diags.Append(d...)
	return CaseInsensitiveStringSetValue{SetValue: setVal}
}

func computeID(objectType string, objectIDs []string) string {
	sorted := make([]string, len(objectIDs))
	copy(sorted, objectIDs)
	sort.Strings(sorted)
	return fmt.Sprintf("%s/%s", objectType, strings.Join(sorted, ","))
}

func toUpperSlice(ss []string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = strings.ToUpper(s)
	}
	return out
}

func toStringSet(ss []string) map[string]struct{} {
	m := make(map[string]struct{}, len(ss))
	for _, s := range ss {
		m[s] = struct{}{}
	}
	return m
}
