package resource

import (
	"context"
	"fmt"

	"github.com/SigNoz/terraform-provider-signoz/signoz/internal/attr"
	"github.com/SigNoz/terraform-provider-signoz/signoz/internal/client"
	"github.com/SigNoz/terraform-provider-signoz/signoz/internal/model"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &dashboardResource{}
	_ resource.ResourceWithConfigure   = &dashboardResource{}
	_ resource.ResourceWithImportState = &dashboardResource{}
)

// NewDashboardResource is a helper function to simplify the provider implementation.
func NewDashboardResource() resource.Resource {
	return &dashboardResource{}
}

// dashboardResource is the resource implementation.
type dashboardResource struct {
	client *client.Client
}

// dashboardResourceModel maps the resource schema data.
type dashboardResourceModel struct {
	CollapsableRowsMigrated types.Bool   `tfsdk:"collapsable_rows_migrated"`
	CreatedAt               types.String `tfsdk:"created_at"`
	CreatedBy               types.String `tfsdk:"created_by"`
	Description             types.String `tfsdk:"description"`
	ID                      types.String `tfsdk:"id"`
	Layout                  types.String `tfsdk:"layout"`
	Name                    types.String `tfsdk:"name"`
	PanelMap                types.String `tfsdk:"panel_map"`
	Source                  types.String `tfsdk:"source"`
	Tags                    types.List   `tfsdk:"tags"`
	Title                   types.String `tfsdk:"title"`
	UpdatedAt               types.String `tfsdk:"updated_at"`
	UpdatedBy               types.String `tfsdk:"updated_by"`
	UploadedGrafana         types.Bool   `tfsdk:"uploaded_grafana"`
	Variables               types.String `tfsdk:"variables"`
	Version                 types.String `tfsdk:"version"`
	Widgets                 types.String `tfsdk:"widgets"`
}

// Configure adds the provider configured client to the resource.
func (r *dashboardResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		addErr(
			&resp.Diagnostics,
			fmt.Errorf("unexpected resource configure type. Expected *client.Client, got: %T. "+
				"Please report this issue to the provider developers", req.ProviderData),
			operationConfigure, SigNozDashboard,
		)

		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *dashboardResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = SigNozDashboard
}

// Schema defines the schema for the resource.
func (r *dashboardResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages dashboard resources in SigNoz.",
		Attributes: map[string]schema.Attribute{
			attr.CollapsableRowsMigrated: schema.BoolAttribute{
				Required: true,
			},
			attr.Description: schema.StringAttribute{
				Required:    true,
				Description: "Description of the dashboard.",
			},
			attr.Layout: schema.StringAttribute{
				Required:    true,
				Description: "Layout of the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.Name: schema.StringAttribute{
				Required:    true,
				Description: "Name of the dashboard.",
			},
			attr.PanelMap: schema.StringAttribute{
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.Source: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Source of the dashboard. By default, it is <SIGNOZ_ENDPOINT>/dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.Tags: schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Tags of the dashboard.",
			},
			attr.Title: schema.StringAttribute{
				Required:    true,
				Description: "Title of the dashboard.",
			},
			attr.UploadedGrafana: schema.BoolAttribute{
				Required: true,
			},
			attr.Variables: schema.StringAttribute{
				Required:    true,
				Description: "Variables for the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.Widgets: schema.StringAttribute{
				Required:    true,
				Description: "Widgets for the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.Version: schema.StringAttribute{
				Required:    true,
				Description: "Version of the dashboard.",
			},

			// computed.
			attr.ID: schema.StringAttribute{
				Computed:    true,
				Description: "Autogenerated unique ID for the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.CreatedAt: schema.StringAttribute{
				Computed:    true,
				Description: "Creation time of the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.CreatedBy: schema.StringAttribute{
				Computed:    true,
				Description: "Creator of the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.UpdatedAt: schema.StringAttribute{
				Computed:    true,
				Description: "Last update time of the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.UpdatedBy: schema.StringAttribute{
				Computed:    true,
				Description: "Last updater of the dashboard.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *dashboardResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan.
	var plan dashboardResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body.
	dashboardPayload := &model.Dashboard{
		CollapsableRowsMigrated: plan.CollapsableRowsMigrated.ValueBool(),
		Description:             plan.Description.ValueString(),
		Name:                    plan.Name.ValueString(),
		Title:                   plan.Title.ValueString(),
		UploadedGrafana:         plan.UploadedGrafana.ValueBool(),
		Version:                 plan.Version.ValueString(),
	}

	err := dashboardPayload.SetLayout(plan.Layout)
	if err != nil {
		addErr(&resp.Diagnostics, err, operationCreate, SigNozDashboard)
		return
	}
	err = dashboardPayload.SetPanelMap(plan.PanelMap)
	if err != nil {
		addErr(&resp.Diagnostics, err, operationCreate, SigNozDashboard)
		return
	}
	dashboardPayload.SetTags(plan.Tags)
	err = dashboardPayload.SetVariables(plan.Variables)
	if err != nil {
		addErr(&resp.Diagnostics, err, operationCreate, SigNozDashboard)
		return
	}
	err = dashboardPayload.SetWidgets(plan.Widgets)
	if err != nil {
		addErr(&resp.Diagnostics, err, operationCreate, SigNozDashboard)
		return
	}

	tflog.Debug(ctx, "Creating dashboard", map[string]any{"dashboard": dashboardPayload})

	// Create new dashboard.
	dashboard, err := r.client.CreateDashboard(ctx, dashboardPayload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating dashboard",
			"Could not create dashboard, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Created dashboard", map[string]any{"dashboard": dashboard})

	// Map response to schema and populate Computed attributes.
	plan.ID = types.StringValue(dashboard.ID)
	plan.Source = types.StringValue(dashboard.Data.Source)
	plan.CreatedAt = types.StringValue(dashboard.CreatedAt)
	plan.CreatedBy = types.StringValue(dashboard.CreatedBy)
	plan.UpdatedAt = types.StringValue(dashboard.UpdatedAt)
	plan.UpdatedBy = types.StringValue(dashboard.UpdatedBy)
	plan.Version = types.StringValue(dashboard.Data.Version)

	// Set state to populated data.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *dashboardResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state dashboardResourceModel
	var diag diag.Diagnostics
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading dashboard", map[string]any{"dashboard": state.ID.ValueString()})

	// Get refreshed dashboard from SigNoz.
	dashboard, err := r.client.GetDashboard(ctx, state.ID.ValueString())
	if err != nil {
		addErr(&resp.Diagnostics, err, operationRead, SigNozDashboard)
		return
	}

	// Preserve original state values for complex JSON fields to avoid drift
	originalWidgets := state.Widgets
	originalLayout := state.Layout
	originalPanelMap := state.PanelMap
	originalVariables := state.Variables

	// Overwrite items with refreshed state.
	state.CollapsableRowsMigrated = types.BoolValue(dashboard.Data.CollapsableRowsMigrated)
	state.CreatedAt = types.StringValue(dashboard.CreatedAt)
	state.CreatedBy = types.StringValue(dashboard.CreatedBy)
	state.Description = types.StringValue(dashboard.Data.Description)
	state.ID = types.StringValue(dashboard.ID)
	state.Name = types.StringValue(dashboard.Data.Name)
	state.Source = types.StringValue(dashboard.Data.Source)
	state.Title = types.StringValue(dashboard.Data.Title)
	state.UpdatedAt = types.StringValue(dashboard.UpdatedAt)
	state.UpdatedBy = types.StringValue(dashboard.UpdatedBy)
	state.UploadedGrafana = types.BoolValue(dashboard.Data.UploadedGrafana)
	state.Version = types.StringValue(dashboard.Data.Version)

	// Preserve original complex JSON fields to avoid API reformatting drift
	state.Widgets = originalWidgets
	state.Layout = originalLayout
	state.PanelMap = originalPanelMap
	state.Variables = originalVariables

	state.Tags, diag = dashboard.Data.TagsToTerraform()
	resp.Diagnostics.Append(diag...)

	// Set refreshed state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *dashboardResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	tflog.Debug(ctx, "Starting dashboard update")

	// Retrieve values from plan.
	var plan, state dashboardResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to get plan from request", map[string]any{"errors": resp.Diagnostics.Errors()})
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		tflog.Error(ctx, "Failed to get state from request", map[string]any{"errors": resp.Diagnostics.Errors()})
		return
	}

	tflog.Debug(ctx, "Retrieved plan and state successfully")

	// Generate API request body from plan.
	var err error
	dashboardUpdate := &model.Dashboard{
		CollapsableRowsMigrated: plan.CollapsableRowsMigrated.ValueBool(),
		Description:             plan.Description.ValueString(),
		Name:                    plan.Name.ValueString(),
		Title:                   plan.Title.ValueString(),
		UploadedGrafana:         plan.UploadedGrafana.ValueBool(),
		Version:                 plan.Version.ValueString(),
	}

	tflog.Debug(ctx, "Setting layout")
	err = dashboardUpdate.SetLayout(plan.Layout)
	if err != nil {
		tflog.Error(ctx, "Failed to set layout", map[string]any{"error": err.Error()})
		addErr(&resp.Diagnostics, err, operationUpdate, SigNozDashboard)
		return
	}

	tflog.Debug(ctx, "Setting panel map")
	err = dashboardUpdate.SetPanelMap(plan.PanelMap)
	if err != nil {
		tflog.Error(ctx, "Failed to set panel map", map[string]any{"error": err.Error()})
		addErr(&resp.Diagnostics, err, operationUpdate, SigNozDashboard)
		return
	}

	tflog.Debug(ctx, "Setting tags")
	dashboardUpdate.SetTags(plan.Tags)

	tflog.Debug(ctx, "Setting variables")
	err = dashboardUpdate.SetVariables(plan.Variables)
	if err != nil {
		tflog.Error(ctx, "Failed to set variables", map[string]any{"error": err.Error()})
		addErr(&resp.Diagnostics, err, operationUpdate, SigNozDashboard)
		return
	}

	tflog.Debug(ctx, "Setting widgets")
	err = dashboardUpdate.SetWidgets(plan.Widgets)
	if err != nil {
		tflog.Error(ctx, "Failed to set widgets", map[string]any{"error": err.Error()})
		addErr(&resp.Diagnostics, err, operationUpdate, SigNozDashboard)
		return
	}

	// Update existing dashboard.
	tflog.Debug(ctx, "Updating dashboard", map[string]any{"dashboardID": state.ID.ValueString()})
	err = r.client.UpdateDashboard(ctx, state.ID.ValueString(), dashboardUpdate)
	if err != nil {
		addErr(&resp.Diagnostics, err, operationUpdate, SigNozDashboard)
		return
	}

	// Instead of fetching fresh state (which causes inconsistencies),
	// we'll use the plan data and preserve the original server-managed fields from state.
	// This avoids the "inconsistent result" error while maintaining data integrity.

	// Preserve server-managed fields from current state
	plan.ID = state.ID
	plan.CreatedAt = state.CreatedAt
	plan.CreatedBy = state.CreatedBy
	plan.UpdatedAt = state.UpdatedAt
	plan.UpdatedBy = state.UpdatedBy
	plan.Source = state.Source

	// Set refreshed state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *dashboardResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state.
	var state dashboardResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing dashboard.
	err := r.client.DeleteDashboard(ctx, state.ID.ValueString())
	if err != nil {
		addErr(&resp.Diagnostics, err, operationDelete, SigNozDashboard)
		return
	}
}

// ImportState imports Terraform state into the resource.
func (r *dashboardResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
