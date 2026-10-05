package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &FormDefinitionDataSource{}

func NewFormDefinitionDataSource() datasource.DataSource {
	return &FormDefinitionDataSource{}
}

type FormDefinitionDataSource struct {
	client *Config
}

type FormDefinitionDataSourceModel struct {
	ID                 types.String `tfsdk:"id"`
	Name               types.String `tfsdk:"name"`
	Description        types.String `tfsdk:"description"`
	Owner              types.List   `tfsdk:"owner"`
	FormInput          types.List   `tfsdk:"form_input"`
	FormElementsJSON   types.String `tfsdk:"form_elements_json"`
	FormConditionsJSON types.String `tfsdk:"form_conditions_json"`
	UsedBy             types.List   `tfsdk:"used_by"`
	Created            types.String `tfsdk:"created"`
	Modified           types.String `tfsdk:"modified"`
}

func (d *FormDefinitionDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_form_definition"
}

func (d *FormDefinitionDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Looks up a SailPoint custom form definition by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The form definition ID.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the form definition.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The form definition description.",
			},
			"owner": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "The form definition owner. Contains `id`, `type` and `name`.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, MarkdownDescription: "Owner identity ID."},
						"type": schema.StringAttribute{Computed: true, MarkdownDescription: "Owner type."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Owner name."},
					},
				},
			},
			"form_input": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Form inputs required when creating a form instance. Each item contains `id`, `type`, `label` and `description`.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true, MarkdownDescription: "Form input identifier."},
						"type":        schema.StringAttribute{Computed: true, MarkdownDescription: "Form input type."},
						"label":       schema.StringAttribute{Computed: true, MarkdownDescription: "Form input name."},
						"description": schema.StringAttribute{Computed: true, MarkdownDescription: "Form input description."},
					},
				},
			},
			"form_elements_json": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The form elements as a JSON array.",
			},
			"form_conditions_json": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The form conditions as a JSON array.",
			},
			"used_by": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "Systems currently using the form definition. Each item contains `id`, `type` and `name`.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, MarkdownDescription: "ID of the system using the form definition."},
						"type": schema.StringAttribute{Computed: true, MarkdownDescription: "Type of the system using the form definition."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Name of the system using the form definition."},
					},
				},
			},
			"created": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The date and time the form definition was created.",
			},
			"modified": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The date and time the form definition was last modified.",
			},
		},
	}
}

func (d *FormDefinitionDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *FormDefinitionDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FormDefinitionDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Form Definition data source", map[string]interface{}{"name": data.Name.ValueString()})

	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	form, err := client.GetFormDefinitionByName(ctx, data.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Form definition with name %s not found", data.Name.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(form.ID)
	data.Name = types.StringValue(form.Name)
	data.Description = types.StringValue(form.Description)
	data.Owner = formOwnerState(ctx, form.Owner, types.ListNull(formOwnerObjectType), &resp.Diagnostics)
	data.FormInput = formInputState(ctx, form.FormInput, types.ListNull(formInputObjectType), &resp.Diagnostics)
	data.FormElementsJSON = formDefinitionJSONState(types.StringNull(), form.FormElements)
	data.FormConditionsJSON = formDefinitionJSONState(types.StringNull(), form.FormConditions)
	data.UsedBy = formUsedByState(ctx, form.UsedBy, &resp.Diagnostics)
	data.Created = stringValueOrNull(form.Created)
	data.Modified = stringValueOrNull(form.Modified)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
