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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &WorkflowResource{}
var _ resource.ResourceWithImportState = &WorkflowResource{}

func NewWorkflowResource() resource.Resource {
	return &WorkflowResource{}
}

type WorkflowResource struct {
	client *Config
}

type WorkflowResourceModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Owner       types.List   `tfsdk:"owner"`
	Enabled     types.Bool   `tfsdk:"enabled"`
	Trigger     types.List   `tfsdk:"trigger"`
	Definition  types.List   `tfsdk:"definition"`
}

type WorkflowOwnerModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
	Name types.String `tfsdk:"name"`
}

type WorkflowTriggerModel struct {
	Type           types.String `tfsdk:"type"`
	DisplayName    types.String `tfsdk:"display_name"`
	AttributesJSON types.String `tfsdk:"attributes_json"`
}

type WorkflowDefinitionModel struct {
	Start     types.String `tfsdk:"start"`
	StepsJSON types.String `tfsdk:"steps_json"`
}

func (r *WorkflowResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (r *WorkflowResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Workflow resource",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				MarkdownDescription: "Workflow ID",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "Workflow name",
				Required:            true,
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Workflow description",
				Optional:            true,
			},
			"enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether the workflow is enabled",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"owner": schema.ListNestedBlock{
				MarkdownDescription: "Workflow owner",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							MarkdownDescription: "Owner ID",
							Required:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Owner type",
							Required:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "Owner name",
							Required:            true,
						},
					},
				},
			},
			"trigger": schema.ListNestedBlock{
				MarkdownDescription: "Workflow trigger configuration",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							MarkdownDescription: "Trigger type (EVENT, SCHEDULED, or EXTERNAL)",
							Required:            true,
						},
						"display_name": schema.StringAttribute{
							MarkdownDescription: "Trigger display name",
							Optional:            true,
						},
						"attributes_json": schema.StringAttribute{
							MarkdownDescription: "Trigger attributes as a JSON string",
							Optional:            true,
						},
					},
				},
			},
			"definition": schema.ListNestedBlock{
				MarkdownDescription: "Workflow definition",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"start": schema.StringAttribute{
							MarkdownDescription: "The name of the starting step",
							Required:            true,
						},
						"steps_json": schema.StringAttribute{
							MarkdownDescription: "Workflow steps as a JSON string",
							Required:            true,
						},
					},
				},
			},
		},
	}
}

func (r *WorkflowResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WorkflowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data WorkflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	workflow := &Workflow{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	// Parse owner
	var owners []WorkflowOwnerModel
	resp.Diagnostics.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(owners) > 0 {
		workflow.Owner = &WorkflowOwner{
			ID:   owners[0].ID.ValueString(),
			Type: owners[0].Type.ValueString(),
			Name: owners[0].Name.ValueString(),
		}
	}

	if !data.Enabled.IsNull() {
		enabled := data.Enabled.ValueBool()
		workflow.Enabled = &enabled
	}

	// Parse trigger
	if !data.Trigger.IsNull() {
		var triggers []WorkflowTriggerModel
		resp.Diagnostics.Append(data.Trigger.ElementsAs(ctx, &triggers, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(triggers) > 0 {
			workflow.Trigger = workflowTriggerModelToAPI(triggers[0], &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// Parse definition
	if !data.Definition.IsNull() {
		var defs []WorkflowDefinitionModel
		resp.Diagnostics.Append(data.Definition.ElementsAs(ctx, &defs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(defs) > 0 {
			workflow.Definition = workflowDefinitionModelToAPI(defs[0], &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	tflog.Info(ctx, "Creating Workflow", map[string]interface{}{"name": workflow.Name})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	newWorkflow, err := client.CreateWorkflow(ctx, workflow)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create workflow: %s", err))
		return
	}

	data.ID = types.StringValue(newWorkflow.ID)
	if newWorkflow.Enabled != nil {
		data.Enabled = types.BoolValue(*newWorkflow.Enabled)
	}

	tflog.Trace(ctx, "created a workflow resource")
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data WorkflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Reading Workflow", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	workflow, err := client.GetWorkflow(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read workflow: %s", err))
		return
	}

	data.Name = types.StringValue(workflow.Name)
	if workflow.Description != "" {
		data.Description = types.StringValue(workflow.Description)
	} else if data.Description.IsNull() {
		// keep null
	} else {
		data.Description = types.StringValue(workflow.Description)
	}

	if workflow.Enabled != nil {
		data.Enabled = types.BoolValue(*workflow.Enabled)
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

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data WorkflowResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Updating Workflow", map[string]interface{}{"id": data.ID.ValueString()})

	workflow := &Workflow{
		Name:        data.Name.ValueString(),
		Description: data.Description.ValueString(),
	}

	// Parse owner
	var owners []WorkflowOwnerModel
	resp.Diagnostics.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(owners) > 0 {
		workflow.Owner = &WorkflowOwner{
			ID:   owners[0].ID.ValueString(),
			Type: owners[0].Type.ValueString(),
			Name: owners[0].Name.ValueString(),
		}
	}

	if !data.Enabled.IsNull() {
		enabled := data.Enabled.ValueBool()
		workflow.Enabled = &enabled
	}

	// Parse trigger
	if !data.Trigger.IsNull() {
		var triggers []WorkflowTriggerModel
		resp.Diagnostics.Append(data.Trigger.ElementsAs(ctx, &triggers, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(triggers) > 0 {
			workflow.Trigger = workflowTriggerModelToAPI(triggers[0], &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	// Parse definition
	if !data.Definition.IsNull() {
		var defs []WorkflowDefinitionModel
		resp.Diagnostics.Append(data.Definition.ElementsAs(ctx, &defs, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(defs) > 0 {
			workflow.Definition = workflowDefinitionModelToAPI(defs[0], &resp.Diagnostics)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	updatedWorkflow, err := client.UpdateWorkflow(ctx, data.ID.ValueString(), workflow)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update workflow: %s", err))
		return
	}

	if updatedWorkflow.Enabled != nil {
		data.Enabled = types.BoolValue(*updatedWorkflow.Enabled)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *WorkflowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data WorkflowResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Info(ctx, "Deleting Workflow", map[string]interface{}{"id": data.ID.ValueString()})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to get IdentityNow client: %s", err))
		return
	}

	err = client.DeleteWorkflow(ctx, data.ID.ValueString())
	if err != nil {
		if _, notFound := err.(*NotFoundError); notFound {
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete workflow: %s", err))
		return
	}
}

func (r *WorkflowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// workflowTriggerModelToAPI converts a WorkflowTriggerModel to the API WorkflowTrigger struct.
func workflowTriggerModelToAPI(m WorkflowTriggerModel, diags *diag.Diagnostics) *WorkflowTrigger {
	trigger := &WorkflowTrigger{
		Type: m.Type.ValueString(),
	}
	if !m.DisplayName.IsNull() {
		trigger.DisplayName = m.DisplayName.ValueString()
	}
	if !m.AttributesJSON.IsNull() && m.AttributesJSON.ValueString() != "" {
		var attrs interface{}
		if err := json.Unmarshal([]byte(m.AttributesJSON.ValueString()), &attrs); err != nil {
			diags.AddError("Invalid JSON", fmt.Sprintf("Unable to parse trigger attributes_json: %s", err))
			return nil
		}
		trigger.Attributes = attrs
	}
	return trigger
}

// workflowDefinitionModelToAPI converts a WorkflowDefinitionModel to the API WorkflowDefinition struct.
func workflowDefinitionModelToAPI(m WorkflowDefinitionModel, diags *diag.Diagnostics) *WorkflowDefinition {
	def := &WorkflowDefinition{
		Start: m.Start.ValueString(),
	}
	if !m.StepsJSON.IsNull() && m.StepsJSON.ValueString() != "" {
		var steps interface{}
		if err := json.Unmarshal([]byte(m.StepsJSON.ValueString()), &steps); err != nil {
			diags.AddError("Invalid JSON", fmt.Sprintf("Unable to parse definition steps_json: %s", err))
			return nil
		}
		def.Steps = steps
	}
	return def
}

// stringValueOrNull returns a types.StringValue if the string is non-empty, otherwise types.StringNull.
func stringValueOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
