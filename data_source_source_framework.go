package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &SourceDataSource{}

func NewSourceDataSource() datasource.DataSource {
	return &SourceDataSource{}
}

type SourceDataSource struct {
	client *Config
}

type SourceDataSourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Connector       types.String `tfsdk:"connector"`
	DeleteThreshold types.Int64  `tfsdk:"delete_threshold"`
	Authoritative   types.Bool   `tfsdk:"authoritative"`
	Owner           types.List   `tfsdk:"owner"`
	Cluster         types.List   `tfsdk:"cluster"`
}

func (d *SourceDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source"
}

func (d *SourceDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Source data source - looks up a source by name",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source ID",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Source name",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source description",
			},
			"connector": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Source connector type",
			},
			"delete_threshold": schema.Int64Attribute{
				Computed:            true,
				MarkdownDescription: "Delete threshold",
			},
			"authoritative": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the source is authoritative",
			},
			"owner": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Source owner",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
			"cluster": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Source cluster",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"type": schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

func (d *SourceDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *SourceDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SourceDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Source data source", map[string]interface{}{"name": data.Name.ValueString()})

	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	sources, err := client.GetSourceByName(ctx, data.Name.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Source with name %s not found", data.Name.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	if len(sources) == 0 {
		resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Source with name %s not found", data.Name.ValueString()))
		return
	}

	source := sources[0]
	data.ID = types.StringValue(source.ID)
	data.Name = types.StringValue(source.Name)
	data.Description = types.StringValue(source.Description)
	data.Connector = types.StringValue(source.Connector)
	data.DeleteThreshold = types.Int64Value(int64(source.DeleteThreshold))
	data.Authoritative = types.BoolValue(source.Authoritative)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
