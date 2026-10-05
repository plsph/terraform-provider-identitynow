package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SegmentDataSource{}

func NewSegmentDataSource() datasource.DataSource {
	return &SegmentDataSource{}
}

type SegmentDataSource struct {
	client *Config
}

type SegmentDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Owner              types.List   `tfsdk:"owner"`
	VisibilityCriteria types.List   `tfsdk:"visibility_criteria"`
	Active             types.Bool   `tfsdk:"active"`
	Created            types.String `tfsdk:"created"`
	Modified           types.String `tfsdk:"modified"`
}

func (d *SegmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_segment"
}

func (d *SegmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a SailPoint identity segment by name.",
		Attributes: map[string]schema.Attribute{
			"id":                  schema.StringAttribute{Computed: true, MarkdownDescription: "The segment ID."},
			"name":                schema.StringAttribute{Required: true, MarkdownDescription: "The segment business name."},
			"description":         schema.StringAttribute{Computed: true, MarkdownDescription: "The segment description."},
			"visibility_criteria": visibilityCriteriaAttribute(segmentVisibilityDepth),
			"active":              schema.BoolAttribute{Computed: true, MarkdownDescription: "Whether the segment is active."},
			"created":             schema.StringAttribute{Computed: true, MarkdownDescription: "The segment creation timestamp."},
			"modified":            schema.StringAttribute{Computed: true, MarkdownDescription: "The segment modification timestamp."},
			"owner": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List with the segment owner.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, MarkdownDescription: "Owner identity ID."},
						"type": schema.StringAttribute{Computed: true, MarkdownDescription: "Owner type, `IDENTITY`."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Owner name."},
					},
				},
			},
		},
	}
}

func visibilityCriteriaAttribute(depth int) schema.Attribute {
	return schema.ListNestedAttribute{
		Computed:            true,
		MarkdownDescription: "The segment visibility criteria as structured nested attributes.",
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"expression": visibilityExpressionAttribute(depth),
			},
		},
	}
}

func visibilityExpressionAttribute(depth int) schema.Attribute {
	attributes := map[string]schema.Attribute{
		"operator":  schema.StringAttribute{Computed: true, MarkdownDescription: "Operator, e.g. `EQUALS`, `AND` or `OR`."},
		"attribute": schema.StringAttribute{Computed: true, MarkdownDescription: "Identity attribute to compare, for comparison operators."},
		"value": schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Value to compare with.",
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"type":  schema.StringAttribute{Computed: true, MarkdownDescription: "Value type, e.g. `STRING`."},
				"value": schema.StringAttribute{Computed: true, MarkdownDescription: "The value."},
			}},
		},
	}
	if depth > 1 {
		attributes["children"] = schema.ListNestedAttribute{
			Computed:            true,
			MarkdownDescription: "Child criteria for `AND` and `OR` operators, each with a nested `expression` (up to 3 levels).",
			NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"expression": visibilityExpressionAttribute(depth - 1),
			}},
		}
	}
	return schema.ListNestedAttribute{Computed: true, MarkdownDescription: "Expression of the visibility criteria.", NestedObject: schema.NestedAttributeObject{Attributes: attributes}}
}

func (d *SegmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SegmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SegmentDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	segment, err := client.GetSegmentByName(ctx, data.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Segment with name %s not found", data.Name.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(segment.ID)
	data.Name = types.StringValue(segment.Name)
	data.Description = types.StringValue(segment.Description)
	data.Active = types.BoolValue(segment.Active)
	data.Created = types.StringValue(segment.Created)
	data.Modified = types.StringValue(segment.Modified)
	data.Owner = segmentOwnerState(ctx, segment.Owner, &resp.Diagnostics)
	data.VisibilityCriteria = segmentVisibilityCriteriaState(ctx, segment.VisibilityCriteria, &resp.Diagnostics)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
