# Migration to Terraform Plugin Framework v1.17.0

This guide documents the migration from `terraform-plugin-sdk/v2` to `terraform-plugin-framework` v1.17.0.

## What Has Been Completed

### ✅ Core Framework Migration

1. **Dependencies Updated** ([go.mod](go.mod))
   - Removed: `github.com/hashicorp/terraform-plugin-sdk/v2 v2.38.1`
   - Added: `github.com/hashicorp/terraform-plugin-framework v1.17.0`
   - Added: `github.com/hashicorp/terraform-plugin-go v0.29.0`

2. **Provider Entrypoint** ([main.go](main.go))
   - Migrated from `plugin.Serve()` to `providerserver.Serve()`
   - Updated to use framework's provider server
   - Added version variable support

3. **Provider Definition** ([provider.go](provider.go))
   - Completely rewritten to implement `provider.Provider` interface
   - Replaced `*schema.Provider` with Framework provider struct
   - Implemented required methods:
     - `Metadata()`: Returns provider type name and version
     - `Schema()`: Defines provider configuration schema
     - `Configure()`: Configures the provider
     - `Resources()`: Returns list of resource constructors
     - `DataSources()`: Returns list of data source constructors

### Key Differences: SDK v2 vs Framework

| Aspect | SDK v2 | Framework |
|--------|--------|-----------|
| **Provider Definition** | Function returning `*schema.Provider` | Struct implementing `provider.Provider` interface |
| **Schema Definition** | `map[string]*schema.Schema` | Typed schema with `schema.Attribute` objects |
| **Type Safety** | Uses `interface{}` and type assertions | Uses typed `types.String`, `types.Int64`, etc. |
| **Defaults** | `DefaultFunc` for env vars | Manual env var handling in `Configure()` |
| **Resource Registration** | `ResourcesMap` map | `Resources()` method returning constructors |
| **Data Source Registration** | `DataSourcesMap` map | `DataSources()` method returning constructors |

## What Needs To Be Done

### 🔲 Resources Migration

All resources need to be migrated from SDK v2 to Framework. Here's the pattern:

#### SDK v2 Pattern (OLD)
```go
func resourceExample() *schema.Resource {
    return &schema.Resource{
        CreateContext: resourceExampleCreate,
        ReadContext:   resourceExampleRead,
        UpdateContext: resourceExampleUpdate,
        DeleteContext: resourceExampleDelete,
        Schema: map[string]*schema.Schema{
            "name": {
                Type:     schema.TypeString,
                Required: true,
            },
        },
    }
}

func resourceExampleCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
    name := d.Get("name").(string)
    // ... create logic
    d.SetId("some-id")
    return nil
}
```

#### Framework Pattern (NEW)
```go
// Ensure the struct implements resource.Resource
var _ resource.Resource = &ExampleResource{}

// ExampleResource defines the resource implementation
type ExampleResource struct {
    client *Config
}

// ExampleResourceModel describes the resource data model
type ExampleResourceModel struct {
    Id   types.String `tfsdk:"id"`
    Name types.String `tfsdk:"name"`
}

// NewExampleResource is a constructor
func NewExampleResource() resource.Resource {
    return &ExampleResource{}
}

// Metadata returns the resource type name
func (r *ExampleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_example"
}

// Schema defines the resource schema
func (r *ExampleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Manages an example resource",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Resource ID",
                Computed:    true,
                PlanModifiers: []planmodifier.String{
                    stringplanmodifier.UseStateForUnknown(),
                },
            },
            "name": schema.StringAttribute{
                Description: "Resource name",
                Required:    true,
            },
        },
    }
}

// Configure adds the provider configured client to the resource
func (r *ExampleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }

    config, ok := req.ProviderData.(*Config)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Resource Configure Type",
            fmt.Sprintf("Expected *Config, got: %T", req.ProviderData),
        )
        return
    }

    r.client = config
}

// Create creates the resource
func (r *ExampleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var data ExampleResourceModel

    resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Create logic here
    name := data.Name.ValueString()
    
    client, err := r.client.IdentityNowClient(ctx)
    if err != nil {
        resp.Diagnostics.AddError("Client Error", err.Error())
        return
    }

    // Set ID and save to state
    data.Id = types.StringValue("some-id")
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource
func (r *ExampleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var data ExampleResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Read logic here
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource
func (r *ExampleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
    var data ExampleResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Update logic here
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource
func (r *ExampleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var data ExampleResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Delete logic here
}
```

### 🔲 Data Sources Migration

Data sources follow a similar pattern:

#### SDK v2 Pattern (OLD)
```go
func dataSourceExample() *schema.Resource {
    return &schema.Resource{
        ReadContext: dataSourceExampleRead,
        Schema: map[string]*schema.Schema{
            "name": {
                Type:     schema.TypeString,
                Required: true,
            },
        },
    }
}
```

#### Framework Pattern (NEW)
```go
var _ datasource.DataSource = &ExampleDataSource{}

type ExampleDataSource struct {
    client *Config
}

type ExampleDataSourceModel struct {
    Id   types.String `tfsdk:"id"`
    Name types.String `tfsdk:"name"`
}

func NewExampleDataSource() datasource.DataSource {
    return &ExampleDataSource{}
}

func (d *ExampleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_example"
}

func (d *ExampleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Description: "Fetches an example data source",
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Description: "Resource ID",
                Computed:    true,
            },
            "name": schema.StringAttribute{
                Description: "Resource name",
                Required:    true,
            },
        },
    }
}

func (d *ExampleDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
    if req.ProviderData == nil {
        return
    }

    config, ok := req.ProviderData.(*Config)
    if !ok {
        resp.Diagnostics.AddError(
            "Unexpected Data Source Configure Type",
            fmt.Sprintf("Expected *Config, got: %T", req.ProviderData),
        )
        return
    }

    d.client = config
}

func (d *ExampleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
    var data ExampleDataSourceModel

    resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() {
        return
    }

    // Read logic here
    data.Id = types.StringValue("some-id")
    
    resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

## Migration Checklist

### Resources to Migrate
- [ ] `resource_source.go` → New framework resource
- [ ] `resource_access_profile.go` → New framework resource
- [ ] `resource_role.go` → New framework resource
- [ ] `resource_account_schema.go` → New framework resource
- [ ] `resource_password_policy.go` → New framework resource
- [ ] `resource_governance_group.go` → New framework resource
- [ ] `resource_source_app.go` → New framework resource
- [ ] `resource_access_profile_attachment.go` → New framework resource
- [ ] `resource_governance_group_members.go` → New framework resource
- [ ] `resource_schedule_account_aggregation.go` → New framework resource

### Data Sources to Migrate
- [ ] `data_source_source.go` → New framework data source
- [ ] `data_source_access_profile.go` → New framework data source
- [ ] `data_source_source_entitlements.go` → New framework data source
- [ ] `data_source_Identity.go` → New framework data source
- [ ] `data_source_role.go` → New framework data source
- [ ] `data_source_governance_group.go` → New framework data source
- [ ] `data_source_source_app.go` → New framework data source

### Supporting Files to Migrate/Update
- [ ] All `schema_*.go` files → Convert to framework schema definitions
- [ ] All `structure_*.go` files → Convert to framework model types
- [ ] All `import_*.go` files → Add `ImportState` method to resources
- [ ] Test files (`*_test.go`) → Update to use framework testing

## Important Notes

### Type Conversions

| SDK v2 | Framework |
|--------|-----------|
| `d.Get("field").(string)` | `data.Field.ValueString()` |
| `d.Get("field").(int)` | `int(data.Field.ValueInt64())` |
| `d.Get("field").(bool)` | `data.Field.ValueBool()` |
| `d.Set("field", value)` | `data.Field = types.StringValue(value)` then `resp.State.Set(ctx, &data)` |
| `d.SetId(id)` | `data.Id = types.StringValue(id)` |
| `d.Id()` | `data.Id.ValueString()` |
| `return diag.FromErr(err)` | `resp.Diagnostics.AddError("Title", err.Error()); return` |

### Schema Attributes

| SDK v2 | Framework |
|--------|-----------|
| `schema.TypeString` | `schema.StringAttribute{}` |
| `schema.TypeInt` | `schema.Int64Attribute{}` |
| `schema.TypeBool` | `schema.BoolAttribute{}` |
| `schema.TypeList` with primitives | `schema.ListAttribute{ElementType: types.StringType}` |
| `schema.TypeList` with objects | `schema.ListNestedAttribute{NestedObject: ...}` |
| `schema.TypeSet` | `schema.SetAttribute{}` or `schema.SetNestedAttribute{}` |
| `schema.TypeMap` | `schema.MapAttribute{}` |
| `Required: true` | `Required: true` |
| `Optional: true` | `Optional: true` |
| `Computed: true` | `Computed: true` |
| `Sensitive: true` | `Sensitive: true` |

### Latest Framework Features (v1.17.0)

The migration to v1.17.0 includes access to these modern features:

1. **Improved Nested Attributes**: Better support for complex nested structures
2. **Enhanced Validators**: Built-in validators for common use cases
3. **Plan Modifiers**: Control attribute behavior during planning (e.g., `UseStateForUnknown()`)
4. **Function Support**: Define provider-level functions (new in v1.8+)
5. **Improved Diagnostics**: Better error messages and warnings
6. **Deferred Actions**: Support for deferred resource actions (v1.14+)
7. **Ephemeral Resources**: Support for ephemeral resources (v1.10+)

## Next Steps

1. **Choose a migration approach**:
   - Option A: Migrate resources one-by-one, testing each
   - Option B: Migrate all at once (higher risk)
   - Recommended: Start with a simple resource like `resource_role.go`

2. **Register migrated resources** in `provider.go`:
   - Uncomment the resource/data source constructors in `Resources()` and `DataSources()` methods as you migrate them

3. **Test thoroughly**:
   - Unit tests
   - Acceptance tests
   - Manual testing with real configurations

4. **Update documentation**:
   - Provider documentation
   - Resource/data source examples
   - CHANGELOG.md

## References

- [Plugin Framework Documentation](https://developer.hashicorp.com/terraform/plugin/framework)
- [Migration Guide (Official)](https://developer.hashicorp.com/terraform/plugin/framework/migrating)
- [Framework vs SDK Comparison](https://developer.hashicorp.com/terraform/plugin/framework-benefits)
- [Plugin Framework Repository](https://github.com/hashicorp/terraform-plugin-framework)
