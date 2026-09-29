package main

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var _ resource.Resource = &SourceResource{}
var _ resource.ResourceWithImportState = &SourceResource{}

func NewSourceResource() resource.Resource {
	return &SourceResource{}
}

type SourceResource struct {
	client *Config
}

type SourceResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	Owner           types.List   `tfsdk:"owner"`
	Cluster         types.List   `tfsdk:"cluster"`
	Connector       types.String `tfsdk:"connector"`
	DeleteThreshold types.Int64  `tfsdk:"delete_threshold"`
	Authoritative   types.Bool   `tfsdk:"authoritative"`
}

type SourceOwnerModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
	Name types.String `tfsdk:"name"`
}

type ClusterModel struct {
	ID   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`
	Name types.String `tfsdk:"name"`
}

func (r *SourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_source"
}

func (r *SourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Source resource",
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
			"connector": schema.StringAttribute{
				Required: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"delete_threshold": schema.Int64Attribute{
				Required: true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"authoritative": schema.BoolAttribute{
				Required: true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
		},
		Blocks: map[string]schema.Block{
			"owner": schema.ListNestedBlock{
				Validators: []validator.List{listSizeBetween(1, 1)},
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Required: true},
						"type": schema.StringAttribute{Required: true},
						"name": schema.StringAttribute{Required: true},
					},
				},
			},
			"cluster": schema.ListNestedBlock{
				Validators: []validator.List{listSizeBetween(0, 1)},
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

func (r *SourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	source := &Source{
		Name:            data.Name.ValueString(),
		Description:     data.Description.ValueString(),
		Connector:       data.Connector.ValueString(),
		DeleteThreshold: int(data.DeleteThreshold.ValueInt64()),
		Authoritative:   data.Authoritative.ValueBool(),
	}

	var owners []SourceOwnerModel
	resp.Diagnostics.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if len(owners) > 0 {
		source.Owner = &Owner{
			ID:   owners[0].ID.ValueString(),
			Type: owners[0].Type.ValueString(),
			Name: owners[0].Name.ValueString(),
		}
	}

	if !data.Cluster.IsNull() {
		var clusters []ClusterModel
		resp.Diagnostics.Append(data.Cluster.ElementsAs(ctx, &clusters, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
		if len(clusters) > 0 {
			source.Cluster = &Cluster{
				ID:   clusters[0].ID.ValueString(),
				Type: clusters[0].Type.ValueString(),
				Name: clusters[0].Name.ValueString(),
			}
		}
	}

	tflog.Info(ctx, "Creating Source", map[string]interface{}{"name": source.Name})

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	newSource, err := client.CreateSource(ctx, source)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create source: %s", err))
		if newSource != nil && newSource.ID != "" {
			// The source was created but not fully configured, keep it in state so Terraform marks it tainted.
			data.ID = types.StringValue(newSource.ID)
			resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		}
		return
	}

	data.ID = types.StringValue(newSource.ID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	source, err := client.GetSource(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	data.Name = types.StringValue(source.Name)
	data.Description = types.StringValue(source.Description)
	data.Connector = types.StringValue(source.Connector)
	data.DeleteThreshold = types.Int64Value(int64(source.DeleteThreshold))
	data.Authoritative = types.BoolValue(source.Authoritative)

	objType := types.ObjectType{AttrTypes: map[string]attr.Type{
		"id":   types.StringType,
		"type": types.StringType,
		"name": types.StringType,
	}}

	if source.Owner != nil {
		ownerModels := []SourceOwnerModel{
			{
				ID:   types.StringValue(source.Owner.ID),
				Type: types.StringValue(source.Owner.Type),
				Name: types.StringValue(source.Owner.Name),
			},
		}
		ownerList, diags := types.ListValueFrom(ctx, objType, ownerModels)
		resp.Diagnostics.Append(diags...)
		data.Owner = ownerList
	} else {
		data.Owner, _ = types.ListValue(objType, []attr.Value{})
	}

	if source.Cluster != nil {
		clusterModels := []ClusterModel{
			{
				ID:   types.StringValue(source.Cluster.ID),
				Type: types.StringValue(source.Cluster.Type),
				Name: types.StringValue(source.Cluster.Name),
			},
		}
		clusterList, diags := types.ListValueFrom(ctx, objType, clusterModels)
		resp.Diagnostics.Append(diags...)
		data.Cluster = clusterList
	} else {
		data.Cluster, _ = types.ListValue(objType, []attr.Value{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SourceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	// Patch only the managed fields, so connector attributes, schemas and other configuration are kept.
	// connector and authoritative are immutable in the API and force replacement instead.
	patches := []*UpdateSource{
		{Op: "replace", Path: "/name", Value: data.Name.ValueString()},
		{Op: "replace", Path: "/description", Value: data.Description.ValueString()},
		{Op: "replace", Path: "/deleteThreshold", Value: data.DeleteThreshold.ValueInt64()},
	}

	var owners []SourceOwnerModel
	resp.Diagnostics.Append(data.Owner.ElementsAs(ctx, &owners, false)...)
	if len(owners) > 0 {
		patches = append(patches, &UpdateSource{Op: "replace", Path: "/owner", Value: &Owner{
			ID:   owners[0].ID.ValueString(),
			Type: owners[0].Type.ValueString(),
			Name: owners[0].Name.ValueString(),
		}})
	}

	var state SourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	var clusters []ClusterModel
	resp.Diagnostics.Append(data.Cluster.ElementsAs(ctx, &clusters, false)...)
	if len(clusters) > 0 {
		patches = append(patches, &UpdateSource{Op: "replace", Path: "/cluster", Value: &Cluster{
			ID:   clusters[0].ID.ValueString(),
			Type: clusters[0].Type.ValueString(),
			Name: clusters[0].Name.ValueString(),
		}})
	} else if len(state.Cluster.Elements()) > 0 {
		patches = append(patches, &UpdateSource{Op: "remove", Path: "/cluster"})
	}
	if resp.Diagnostics.HasError() {
		return
	}

	if _, err := client.UpdateSource(ctx, data.ID.ValueString(), patches); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update source: %s", err))
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *SourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SourceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	client, err := r.client.IdentityNowClient(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	source, err := client.GetSource(ctx, data.ID.ValueString())
	if err != nil {
		if isNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}

	err = client.DeleteSource(ctx, source)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", err.Error())
		return
	}
}

func (r *SourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
