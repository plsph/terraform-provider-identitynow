package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

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
	registerResource(NewTransformResource)
	registerDataSource(NewTransformDataSource)
}

// Transform is a transform as returned by the /v2026/transforms API.
type Transform struct {
	ID         string                 `json:"id,omitempty"`
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Attributes map[string]interface{} `json:"attributes"`
	Internal   bool                   `json:"internal,omitempty"`
}

func (c *Client) GetTransform(ctx context.Context, id string) (*Transform, error) {
	var transform Transform
	if err := c.doJSON(ctx, http.MethodGet, apiPath("/v2026/transforms/%s", id), nil, &transform); err != nil {
		return nil, err
	}
	return &transform, nil
}

func (c *Client) GetTransformByName(ctx context.Context, name string) (*Transform, error) {
	return findByName(ctx, c, "/v2026/transforms", "transform", name, func(t Transform) string { return t.Name })
}

func (c *Client) CreateTransform(ctx context.Context, transform *Transform) (*Transform, error) {
	var created Transform
	if err := c.doJSON(ctx, http.MethodPost, "/v2026/transforms", transform, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func (c *Client) UpdateTransform(ctx context.Context, id string, transform *Transform) (*Transform, error) {
	var updated Transform
	if err := c.doJSON(ctx, http.MethodPut, apiPath("/v2026/transforms/%s", id), transform, &updated); err != nil {
		return nil, err
	}
	return &updated, nil
}

func (c *Client) DeleteTransform(ctx context.Context, id string) error {
	return c.doJSON(ctx, http.MethodDelete, apiPath("/v2026/transforms/%s", id), nil, nil)
}

var _ resource.Resource = &TransformResource{}
var _ resource.ResourceWithImportState = &TransformResource{}

func NewTransformResource() resource.Resource {
	return &TransformResource{}
}

type TransformResource struct {
	client *Config
}

type TransformResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Name           types.String `tfsdk:"name"`
	Type           types.String `tfsdk:"type"`
	AttributesJSON types.String `tfsdk:"attributes_json"`
	Internal       types.Bool   `tfsdk:"internal"`
}

func (r *TransformResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transform"
}

func (r *TransformResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a transform, used in identity profile attribute mappings.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Transform ID.",
				Computed:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Unique name of the transform. Changing this forces a new transform to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Transform operation type, e.g. `lookup`, `concat` or `dateFormat`. See the [transform operations](https://developer.sailpoint.com/docs/extensibility/transforms/operations). Changing this forces a new transform to be created.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"attributes_json": schema.StringAttribute{
				MarkdownDescription: "Transform attributes as a JSON object. The attributes depend on the transform type. Use `jsonencode()` for convenience. The value is compared semantically, so formatting and key order do not produce a diff.",
				Optional:            true,
				Validators:          []validator.String{jsonObjectStringValidator{}},
			},
			"internal": schema.BoolAttribute{
				MarkdownDescription: "Whether this is a SailPoint internal transform.",
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *TransformResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// transformFromModel converts the model to the API transform.
func transformFromModel(data TransformResourceModel) (*Transform, error) {
	transform := &Transform{Name: data.Name.ValueString(), Type: data.Type.ValueString(), Attributes: map[string]interface{}{}}
	if !data.AttributesJSON.IsNull() && !data.AttributesJSON.IsUnknown() {
		if err := json.Unmarshal([]byte(data.AttributesJSON.ValueString()), &transform.Attributes); err != nil {
			return nil, err
		}
	}
	return transform, nil
}

// setTransformState maps an API transform onto the model, keeping a semantically equal prior JSON value.
func setTransformState(data *TransformResourceModel, transform *Transform) {
	data.ID = types.StringValue(transform.ID)
	data.Name = types.StringValue(transform.Name)
	data.Type = types.StringValue(transform.Type)
	data.AttributesJSON = jsonStringState(data.AttributesJSON, transform.Attributes)
	data.Internal = types.BoolValue(transform.Internal)
}

func (r *TransformResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TransformResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	transform, err := transformFromModel(data)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("attributes_json"), "Invalid JSON", err.Error())
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	created, err := client.CreateTransform(ctx, transform)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create transform: %s", err))
		return
	}
	data.ID = types.StringValue(created.ID)
	data.Internal = types.BoolValue(created.Internal)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TransformResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TransformResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	transform, err := client.GetTransform(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read transform: %s", err))
		return
	}
	setTransformState(&data, transform)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TransformResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TransformResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	transform, err := transformFromModel(data)
	if err != nil {
		resp.Diagnostics.AddAttributeError(path.Root("attributes_json"), "Invalid JSON", err.Error())
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	// Only the attributes are mutable, name and type force replacement.
	updated, err := client.UpdateTransform(ctx, data.ID.ValueString(), transform)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update transform: %s", err))
		return
	}
	data.Internal = computedBoolFromAPI(data.Internal, &updated.Internal)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *TransformResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TransformResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	if err := client.DeleteTransform(ctx, data.ID.ValueString()); err != nil && !isNotFound(err) {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete transform: %s", err))
	}
}

func (r *TransformResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
