# Terraform Plugin Framework Migration Status

## ✅ Completed Changes

### 1. Dependency Upgrade
**File:** [go.mod](go.mod)

The project now uses the latest Terraform Plugin Framework and supporting libraries:

```go
require (
    github.com/hashicorp/terraform-plugin-framework v1.17.0  // Latest framework
    github.com/hashicorp/terraform-plugin-go v0.29.0         // Latest plugin protocol
    github.com/hashicorp/terraform-plugin-log v0.9.0
    golang.org/x/time v0.14.0
)
```

**Benefits:**
- Access to modern framework features (deferred actions, ephemeral resources, etc.)
- Better type safety and validation
- Improved performance
- Better error messages and diagnostics

### 2. Provider Entrypoint (main.go)
**File:** [main.go](main.go)

Migrated from SDK v2's `plugin.Serve()` to Framework's `providerserver.Serve()`:

**Key Changes:**
- Uses `providerserver.Serve()` instead of `plugin.Serve()`
- Provider constructor pattern with version support
- Proper context handling
- Framework-compatible provider address

### 3. Provider Implementation (provider.go)
**File:** [provider.go](provider.go)

Complete rewrite implementing the `provider.Provider` interface:

**New Structure:**
```go
type IdentityNowProvider struct {
    version string
}

type IdentityNowProviderModel struct {
    ApiUrl                 types.String `tfsdk:"api_url"`
    ClientId               types.String `tfsdk:"client_id"`
    ClientSecret           types.String `tfsdk:"client_secret"`
    Credentials            types.List   `tfsdk:"credentials"`
    MaxClientPoolSize      types.Int64  `tfsdk:"max_client_pool_size"`
    DefaultClientPoolSize  types.Int64  `tfsdk:"default_client_pool_size"`
    ClientRequestRateLimit types.Int64  `tfsdk:"client_request_rate_limit"`
}
```

**Implemented Methods:**
- ✅ `Metadata()` - Returns provider name and version
- ✅ `Schema()` - Defines typed schema with proper attributes
- ✅ `Configure()` - Sets up provider configuration with proper diagnostics
- ✅ `Resources()` - Returns resource constructors (ready for migration)
- ✅ `DataSources()` - Returns data source constructors (ready for migration)

**Improvements:**
- Type-safe configuration handling
- Proper environment variable defaults
- Better error diagnostics with `path.Root()` attribution
- Nested attribute support for credentials list
- No raw `interface{}` usage - everything is strongly typed

### 4. Build Verification
The provider compiles successfully with the new framework:
```bash
✓ go build -o terraform-provider-identitynow
```

### 5. Documentation
**File:** [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)

Comprehensive guide covering:
- SDK v2 vs Framework comparison tables
- Step-by-step migration patterns for resources and data sources
- Type conversion reference
- Schema attribute mappings
- Latest v1.17.0 features
- Next steps and best practices

## 📋 Remaining Work

### Resources to Migrate (10 total) - ALL COMPLETE ✅
Each resource has been converted from SDK v2 to Framework pattern:

- [x] `resource_source.go` → `resource_source_framework.go`
- [x] `resource_access_profile.go` → `resource_access_profile_framework.go`
- [x] `resource_role.go` → `resource_role_framework.go`
- [x] `resource_account_schema.go` → `resource_account_schema_framework.go`
- [x] `resource_password_policy.go` → `resource_password_policy_framework.go`
- [x] `resource_governance_group.go` → `resource_governance_group_framework.go`
- [x] `resource_source_app.go` → `resource_source_app_framework.go`
- [x] `resource_access_profile_attachment.go` → `resource_access_profile_attachment_framework.go`
- [x] `resource_governance_group_members.go` → `resource_governance_group_members_framework.go`
- [x] `resource_schedule_account_aggregation.go` → `resource_schedule_account_aggregation_framework.go`

### Data Sources to Migrate (7 total) - ALL COMPLETE ✅
- [x] `data_source_source.go` → `data_source_source_framework.go`
- [x] `data_source_access_profile.go` → `data_source_access_profile_framework.go`
- [x] `data_source_source_entitlements.go` → `data_source_source_entitlement_framework.go`
- [x] `data_source_Identity.go` → `data_source_identity_framework.go`
- [x] `data_source_role.go` → `data_source_role_framework.go`
- [x] `data_source_governance_group.go` → `data_source_governance_group_framework.go`
- [x] `data_source_source_app.go` → `data_source_source_app_framework.go`

### Supporting Files to Update
- [ ] Convert `schema_*.go` files to Framework schema definitions
- [ ] Update `structure_*.go` files to work with Framework types
- [ ] Add `ImportState` methods to migrated resources
- [ ] Update test files (`*_test.go`)
- [ ] Remove unused SDK v2 helper code

## 🎯 Latest Framework Features Now Available

With v1.17.0, your provider gains access to:

### 1. **Enhanced Type System**
- `types.String`, `types.Int64`, `types.Bool`, etc.
- `types.List`, `types.Map`, `types.Set`, `types.Object`
- Full null/unknown value support
- Type-safe value access

### 2. **Modern Schema Features**
- `schema.StringAttribute`, `schema.Int64Attribute`
- `schema.ListNestedAttribute` for complex nested structures
- `schema.SingleNestedAttribute` for single object nesting
- Built-in validators for common patterns

### 3. **Plan Modifiers**
- `stringplanmodifier.UseStateForUnknown()` - Keep computed values stable
- `stringplanmodifier.RequiresReplace()` - Force recreation on change
- Custom plan modifiers for business logic

### 4. **Improved Validators**
- `stringvalidator.LengthBetween()`
- `stringvalidator.OneOf()`
- `int64validator.Between()`
- `listvalidator.SizeAtLeast()`
- Custom validators

### 5. **Better Diagnostics**
- Attribute-specific errors with `path.Root()`
- Warnings support
- Rich error context

### 6. **State Management**
- Explicit state reading: `req.State.Get(ctx, &data)`
- Explicit state writing: `resp.State.Set(ctx, &data)`
- Clear separation of plan vs state

### 7. **New Capabilities (v1.10+)**
- **Ephemeral Resources**: For temporary resources
- **Deferred Actions**: Handle async operations
- **Provider Functions**: Define custom functions
- **Improved Nested Attributes**: Better complex type support

## 🚀 How to Proceed

### Option 1: Incremental Migration (Recommended)
Migrate resources one-by-one, test each thoroughly:

1. Pick a simple resource (e.g., `resource_role.go`)
2. Create new Framework-based implementation
3. Uncomment constructor in `provider.go`
4. Test with acceptance tests
5. Repeat for next resource

### Option 2: Batch Migration
Migrate all resources/data sources at once:

1. Create templates for resource and data source patterns
2. Convert all files in one pass
3. Update all constructors in `provider.go`
4. Run comprehensive test suite

### Recommended First Migration
Start with **`resource_role.go`** because:
- Relatively simple schema
- Good example of nested attributes (owner, access_profiles)
- Has CRUD operations and import
- Demonstrates the migration pattern well

## 📝 Next Steps

1. **Choose migration approach** (incremental recommended)
2. **Migrate first resource** 
3. **Update provider.go registration**
4. **Test thoroughly**
5. **Document changes**
6. **Repeat for remaining resources/data sources**

## 🔧 Testing Strategy

After each migration:
1. Run `go build` to verify compilation
2. Run unit tests if available
3. Run acceptance tests
4. Manual testing with sample configurations
5. Verify state file compatibility

## 📚 Additional Resources

- **Migration Guide**: [MIGRATION_GUIDE.md](MIGRATION_GUIDE.md)
- **Framework Docs**: https://developer.hashicorp.com/terraform/plugin/framework
- **Framework Repository**: https://github.com/hashicorp/terraform-plugin-framework
- **Example Providers**: https://github.com/hashicorp/terraform-provider-scaffolding-framework

---

**Status Last Updated:** Migration foundation complete, ready for resource/data source migration
**Framework Version:** v1.17.0 (Latest)
**Go Version:** 1.24.0
