package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &TransformDataSource{}
var _ datasource.DataSourceWithValidateConfig = &TransformDataSource{}

func NewTransformDataSource() datasource.DataSource {
	return &TransformDataSource{}
}

type TransformDataSource struct {
	client *Config
}

func (d *TransformDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transform"
}

func (d *TransformDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a transform by ID or name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Transform ID. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Transform name. Exactly one of `id` or `name` must be set.",
				Optional:            true,
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Transform operation type.",
				Computed:            true,
			},
			"attributes_json": schema.StringAttribute{
				MarkdownDescription: "Transform attributes as a JSON object.",
				Computed:            true,
			},
			"internal": schema.BoolAttribute{
				MarkdownDescription: "Whether this is a SailPoint internal transform.",
				Computed:            true,
			},
		},
	}
}

func (d *TransformDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	validateExactlyOneOf(ctx, req.Config, resp, "id", "name")
}

func (d *TransformDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *TransformDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TransformResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
	var transform *Transform
	if !data.ID.IsNull() {
		transform, err = client.GetTransform(ctx, data.ID.ValueString())
	} else {
		transform, err = client.GetTransformByName(ctx, data.Name.ValueString())
	}
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddAttributeError(path.Root("name"), "Transform not found", err.Error())
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read transform: %s", err))
		return
	}
	data.AttributesJSON = types.StringNull()
	setTransformState(&data, transform)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
