package provider

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ datasource.DataSource = &WorkflowDataSource{}

func NewWorkflowDataSource() datasource.DataSource {
	return &WorkflowDataSource{}
}

type WorkflowDataSource struct {
	client *Config
}

type WorkflowDataSourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Owner       types.List   `tfsdk:"owner"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Trigger     types.List   `tfsdk:"trigger"`
	Definition  types.List   `tfsdk:"definition"`
	Created     types.String `tfsdk:"created"`
	Modified    types.String `tfsdk:"modified"`
}

func (d *WorkflowDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (d *WorkflowDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to access information about an existing Workflow. The lookup fails when several workflows share the name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The ID of the workflow.",
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the workflow.",
			},
			"description": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Description of the workflow.",
			},
			"owner": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List with the owner of the workflow.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true, MarkdownDescription: "Owner identity ID."},
						"type": schema.StringAttribute{Computed: true, MarkdownDescription: "Owner type."},
						"name": schema.StringAttribute{Computed: true, MarkdownDescription: "Owner name."},
					},
				},
			},
			"enabled": schema.BoolAttribute{
				Computed:            true,
				MarkdownDescription: "Whether the workflow is enabled.",
			},
			"trigger": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List with the trigger configuration.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Trigger type (EVENT, SCHEDULED, or EXTERNAL).",
						},
						"display_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Trigger display name.",
						},
						"attributes_json": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Trigger attributes as a JSON string.",
						},
					},
				},
			},
			"definition": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List with the workflow definition.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"start": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The name of the starting step.",
						},
						"steps_json": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "Workflow steps as a JSON string.",
						},
					},
				},
			},
			"created": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The date and time the workflow was created.",
			},
			"modified": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The date and time the workflow was modified.",
			},
		},
	}
}

func (d *WorkflowDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *WorkflowDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WorkflowDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Workflow data source", map[string]interface{}{"name": data.Name.ValueString()})

	client, err := d.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	workflow, err := client.GetWorkflowByName(ctx, data.Name.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.Diagnostics.AddError("Not Found", fmt.Sprintf("Workflow with name %s not found", data.Name.ValueString()))
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.ID = types.StringValue(workflow.ID)
	data.Name = types.StringValue(workflow.Name)
	data.Description = types.StringValue(workflow.Description)

	if workflow.Enabled != nil {
		data.Enabled = types.BoolValue(*workflow.Enabled)
	} else {
		data.Enabled = types.BoolNull()
	}

	// Map owner
	ownerObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"type": types.StringType,
		"name": types.StringType,
	}}
	if workflow.Owner != nil {
		ownerModels := []WorkflowOwnerModel{
			{
				ID:   types.StringValue(workflow.Owner.ID),
				Type: types.StringValue(workflow.Owner.Type),
				Name: types.StringValue(workflow.Owner.Name),
			},
		}
		ownerList, diags := types.ListValueFrom(ctx, ownerObjType, ownerModels)
		resp.Diagnostics.Append(diags...)
		data.Owner = ownerList
	} else {
		data.Owner = types.ListNull(ownerObjType)
	}

	// Map trigger
	triggerObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"type":            types.StringType,
		"display_name":    types.StringType,
		"attributes_json": types.StringType,
	}}
	if workflow.Trigger != nil {
		attrsJSON := ""
		if workflow.Trigger.Attributes != nil {
			b, err := json.Marshal(workflow.Trigger.Attributes)
			if err == nil {
				attrsJSON = string(b)
			}
		}
		triggerModels := []WorkflowTriggerModel{
			{
				Type:           types.StringValue(workflow.Trigger.Type),
				DisplayName:    stringValueOrNull(workflow.Trigger.DisplayName),
				AttributesJSON: stringValueOrNull(attrsJSON),
			},
		}
		triggerList, diags := types.ListValueFrom(ctx, triggerObjType, triggerModels)
		resp.Diagnostics.Append(diags...)
		data.Trigger = triggerList
	} else {
		data.Trigger = types.ListNull(triggerObjType)
	}

	// Map definition
	defObjType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"start":      types.StringType,
		"steps_json": types.StringType,
	}}
	if workflow.Definition != nil {
		stepsJSON := ""
		if workflow.Definition.Steps != nil {
			b, err := json.Marshal(workflow.Definition.Steps)
			if err == nil {
				stepsJSON = string(b)
			}
		}
		defModels := []WorkflowDefinitionModel{
			{
				Start:     types.StringValue(workflow.Definition.Start),
				StepsJSON: types.StringValue(stepsJSON),
			},
		}
		defList, diags := types.ListValueFrom(ctx, defObjType, defModels)
		resp.Diagnostics.Append(diags...)
		data.Definition = defList
	} else {
		data.Definition = types.ListNull(defObjType)
	}

	data.Created = stringValueOrNull(workflow.Created)
	data.Modified = stringValueOrNull(workflow.Modified)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
