package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/SigNoz/terraform-provider-signoz/signoz/internal/client"
	"github.com/SigNoz/terraform-provider-signoz/signoz/internal/model"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/SigNoz/terraform-provider-signoz/signoz/internal/attr"
)

// jsonSemanticEqualityModifier implements a plan modifier that compares JSON strings semantically
type jsonSemanticEqualityModifier struct{}

func (m jsonSemanticEqualityModifier) Description(_ context.Context) string {
	return "If the planned and state values are semantically equivalent JSON, use the state value to prevent unnecessary updates."
}

func (m jsonSemanticEqualityModifier) MarkdownDescription(ctx context.Context) string {
	return m.Description(ctx)
}

func (m jsonSemanticEqualityModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
	tflog.Debug(ctx, "jsonSemanticEquality: Starting plan modification", map[string]any{
		"stateValue":     req.StateValue.ValueString(),
		"planValue":      req.PlanValue.ValueString(),
		"stateIsNull":    req.StateValue.IsNull(),
		"stateIsUnknown": req.StateValue.IsUnknown(),
		"planIsNull":     req.PlanValue.IsNull(),
		"planIsUnknown":  req.PlanValue.IsUnknown(),
	})

	// Do nothing if there is no state value
	if req.StateValue.IsNull() || req.StateValue.IsUnknown() {
		tflog.Debug(ctx, "jsonSemanticEquality: State value is null or unknown, skipping")
		return
	}

	// Do nothing if there is no planned value
	if req.PlanValue.IsNull() || req.PlanValue.IsUnknown() {
		tflog.Debug(ctx, "jsonSemanticEquality: Plan value is null or unknown, skipping")
		return
	}

	// Compare JSONs semantically to handle formatting differences
	if areJSONsSemanticallyEqual(req.PlanValue.ValueString(), req.StateValue.ValueString()) {
		tflog.Debug(ctx, "jsonSemanticEquality: JSONs are semantically equal, using state value")
		resp.PlanValue = req.StateValue
	} else {
		tflog.Debug(ctx, "jsonSemanticEquality: JSONs are different, keeping plan value")
	}
}

// normalizeJSON normalizes JSON by removing API-added default fields and ensuring consistent formatting
func normalizeJSON(jsonStr string) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return "", err
	}

	// Remove API-added default fields that cause drift
	normalized := removeDefaultFields(data)

	// Marshal back to JSON with consistent formatting
	bytes, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}

	return string(bytes), nil
}

// removeDefaultFields recursively removes API-added default fields that cause drift
func removeDefaultFields(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			// Skip API-added default fields that cause drift
			if isDefaultField(key, value) {
				// Log what we're removing for debugging
				fmt.Printf("Removing default field: %s = %v\n", key, value)
				continue
			}
			result[key] = removeDefaultFields(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = removeDefaultFields(item)
		}
		return result
	default:
		return v
	}
}

// isDefaultField checks if a field is an API-added default that should be ignored
func isDefaultField(key string, value interface{}) bool {
	// Handle specific field types that can't be compared with ==
	switch key {
	case "groupBy":
		// Check if it's an empty slice
		if slice, ok := value.([]interface{}); ok {
			return len(slice) == 0
		}
		return false
	case "IsAnomaly":
		return value == false
	case "QueriesUsedInFormula":
		return value == nil
	case "absentFor":
		return value == 0
	case "alertOnAbsent":
		return value == false
	case "hidden":
		return value == true
	case "reduceTo", "spaceAggregation", "timeAggregation":
		return value == ""
	default:
		return false
	}
}

func jsonSemanticEquality() planmodifier.String {
	return jsonSemanticEqualityModifier{}
}

// Ensure the implementation satisfies the expected interfaces.
var (
	_ resource.Resource                = &alertResource{}
	_ resource.ResourceWithConfigure   = &alertResource{}
	_ resource.ResourceWithImportState = &alertResource{}
)

// NewAlertResource is a helper function to simplify the provider implementation.
func NewAlertResource() resource.Resource {
	return &alertResource{}
}

// alertResource is the resource implementation.
type alertResource struct {
	client *client.Client
}

// alertResourceModel maps the resource schema data.
type alertResourceModel struct {
	ID                types.String `tfsdk:"id"`
	Alert             types.String `tfsdk:"alert"`
	AlertType         types.String `tfsdk:"alert_type"`
	BroadcastToAll    types.Bool   `tfsdk:"broadcast_to_all"`
	Condition         types.String `tfsdk:"condition"`
	Description       types.String `tfsdk:"description"`
	Disabled          types.Bool   `tfsdk:"disabled"`
	EvalWindow        types.String `tfsdk:"eval_window"`
	Frequency         types.String `tfsdk:"frequency"`
	Labels            types.Map    `tfsdk:"labels"`
	PreferredChannels types.List   `tfsdk:"preferred_channels"`
	RuleType          types.String `tfsdk:"rule_type"`
	Severity          types.String `tfsdk:"severity"`
	Source            types.String `tfsdk:"source"`
	State             types.String `tfsdk:"state"`
	Summary           types.String `tfsdk:"summary"`
	Version           types.String `tfsdk:"version"`
	CreateAt          types.String `tfsdk:"create_at"`
	CreateBy          types.String `tfsdk:"create_by"`
	UpdateAt          types.String `tfsdk:"update_at"`
	UpdateBy          types.String `tfsdk:"update_by"`
}

// Configure adds the provider configured client to the resource.
func (r *alertResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	client, ok := req.ProviderData.(*client.Client)
	if !ok {
		addErr(
			&resp.Diagnostics,
			fmt.Errorf("unexpected data source configure type. Expected *client.Client, got: %T. "+
				"Please report this issue to the provider developers", req.ProviderData),
			operationConfigure, SigNozAlert,
		)

		return
	}

	r.client = client
}

// Metadata returns the resource type name.
func (r *alertResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = SigNozAlert
}

// Schema defines the schema for the resource.
func (r *alertResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Creates and manages alert resources in SigNoz.",
		Attributes: map[string]schema.Attribute{
			attr.Alert: schema.StringAttribute{
				Required:    true,
				Description: "Name of the alert.",
			},
			attr.AlertType: schema.StringAttribute{
				Required: true,
				Description: fmt.Sprintf("Type of the alert. Possible values are: %s, %s, %s, and %s.",
					model.AlertTypeMetrics, model.AlertTypeLogs, model.AlertTypeTraces, model.AlertTypeExceptions),
				Validators: []validator.String{
					stringvalidator.OneOf(model.AlertTypes...),
				},
			},
			attr.BroadcastToAll: schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "Whether to broadcast the alert to all the alerting channels. " +
					"By default, the alert is only sent to the preferred channels.",
			},
			attr.Condition: schema.StringAttribute{
				Required:    true,
				Description: "Condition of the alert.",
				PlanModifiers: []planmodifier.String{
					jsonSemanticEquality(),
				},
			},
			attr.Description: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Description of the alert.",
				Default:     stringdefault.StaticString(alertDefaultDescription),
			},
			attr.Disabled: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Whether the alert is disabled.",
				Default:     booldefault.StaticBool(false),
			},
			attr.EvalWindow: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The evaluation window of the alert. By default, it is 5m0s.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^([0-9]+h)?([0-9]+m)?([0-9]+s)?$`), "invalid alert evaluation window. It should be in format of 5m0s or 15m30s"),
				},
				Default: stringdefault.StaticString(alertDefaultEvalWindow),
			},
			attr.Frequency: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "The frequency of the alert. By default, it is 1m0s.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`^([0-9]+h)?([0-9]+m)?([0-9]+s)?$`), "invalid alert frequency. It should be in format of 1m0s or 10m30s"),
				},
				Default: stringdefault.StaticString(alertDefaultFrequency),
			},
			attr.Labels: schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Labels of the alert. Severity is a required label.",
			},
			attr.PreferredChannels: schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Preferred channels of the alert. By default, it is empty.",
			},
			attr.RuleType: schema.StringAttribute{
				Optional: true,
				Computed: true,
				Description: fmt.Sprintf("Type of the alert. Possible values are: %s and %s.",
					model.AlertRuleTypeThreshold, model.AlertRuleTypeProm),
				Validators: []validator.String{
					stringvalidator.OneOf(model.AlertRuleTypes...),
				},
			},
			attr.Severity: schema.StringAttribute{
				Required: true,
				Description: fmt.Sprintf("Severity of the alert. Possible values are: %s, %s, %s, and %s.",
					model.AlertSeverityInfo, model.AlertSeverityWarning, model.AlertSeverityError, model.AlertSeverityCritical),
				Validators: []validator.String{
					stringvalidator.OneOf(model.AlertSeverities...),
				},
			},
			attr.Source: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Source of the alert. By default, it is <SIGNOZ_ENDPOINT>/alerts.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.Summary: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Summary of the alert.",
				Default:     stringdefault.StaticString(alertDefaultSummary),
			},
			attr.Version: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Version of the alert. By default, it is v4.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(regexp.MustCompile(`v\d+`), "alert version should be of the form v3, v4, etc."),
				},
				Default: stringdefault.StaticString(alertDefaultVersion),
			},
			// computed.
			attr.ID: schema.StringAttribute{
				Computed:    true,
				Description: "Autogenerated unique ID for the alert.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.State: schema.StringAttribute{
				Computed:    true,
				Description: "State of the alert.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.CreateAt: schema.StringAttribute{
				Computed:    true,
				Description: "Creation time of the alert.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.CreateBy: schema.StringAttribute{
				Computed:    true,
				Description: "Creator of the alert.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.UpdateAt: schema.StringAttribute{
				Computed:    true,
				Description: "Last update time of the alert.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			attr.UpdateBy: schema.StringAttribute{
				Computed:    true,
				Description: "Last updater of the alert.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Create creates the resource and sets the initial Terraform state.
func (r *alertResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Retrieve values from plan.
	var plan alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body.
	alertPayload := &model.Alert{
		Alert:     plan.Alert.ValueString(),
		AlertType: plan.AlertType.ValueString(),
		Annotations: model.AlertAnnotations{
			Description: plan.Description.ValueString(),
			Summary:     plan.Summary.ValueString(),
		},
		BroadcastToAll: plan.BroadcastToAll.ValueBool(),
		EvalWindow:     plan.EvalWindow.ValueString(),
		Frequency:      plan.Frequency.ValueString(),
		RuleType:       plan.RuleType.ValueString(),
		Source:         plan.Source.ValueString(),
		Version:        plan.Version.ValueString(),
	}

	err := alertPayload.SetCondition(plan.Condition)
	if err != nil {
		addErr(&resp.Diagnostics, err, operationCreate, SigNozAlert)
		return
	}

	alertPayload.SetLabels(plan.Labels, plan.Severity)
	alertPayload.SetPreferredChannels(plan.PreferredChannels)

	tflog.Debug(ctx, "Creating alert", map[string]any{"alert": alertPayload})

	// Create new alert
	alert, err := r.client.CreateAlert(ctx, alertPayload)
	if err != nil {
		resp.Diagnostics.AddError(
			"Error creating alert",
			"Could not create alert, unexpected error: "+err.Error(),
		)
		return
	}

	tflog.Debug(ctx, "Created alert", map[string]any{"alert": alert})

	// Map response to schema and populate Computed attributes.
	plan.ID = types.StringValue(alert.ID)
	plan.Disabled = types.BoolValue(alert.Disabled)
	plan.Source = types.StringValue(alert.Source)
	plan.State = types.StringValue(alert.State)
	plan.CreateAt = types.StringValue(alert.CreateAt)
	plan.CreateBy = types.StringValue(alert.CreateBy)
	plan.UpdateAt = types.StringValue(alert.UpdateAt)
	plan.UpdateBy = types.StringValue(alert.UpdateBy)

	// Set state to populated data.
	resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read refreshes the Terraform state with the latest data.
func (r *alertResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get current state
	var state alertResourceModel
	var diag diag.Diagnostics
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Reading alert", map[string]any{"alert": state.ID.ValueString()})

	// Get refreshed alert from SigNoz.
	alert, err := r.client.GetAlert(ctx, state.ID.ValueString())
	if err != nil {
		addErr(&resp.Diagnostics, err, operationRead, SigNozAlert)
		return
	}

	// Overwrite items with refreshed state.
	state.Alert = types.StringValue(alert.Alert)
	state.AlertType = types.StringValue(alert.AlertType)
	state.BroadcastToAll = types.BoolValue(alert.BroadcastToAll)
	state.Description = types.StringValue(alert.Annotations.Description)
	state.Disabled = types.BoolValue(alert.Disabled)
	state.EvalWindow = types.StringValue(alert.EvalWindow)
	state.Frequency = types.StringValue(alert.Frequency)
	state.RuleType = types.StringValue(alert.RuleType)
	state.Severity = types.StringValue(alert.Labels[attr.Severity])
	state.Source = types.StringValue(alert.Source)
	state.State = types.StringValue(alert.State)
	state.Summary = types.StringValue(alert.Annotations.Summary)
	state.Version = types.StringValue(alert.Version)
	state.CreateAt = types.StringValue(alert.CreateAt)
	state.CreateBy = types.StringValue(alert.CreateBy)
	state.UpdateAt = types.StringValue(alert.UpdateAt)
	state.UpdateBy = types.StringValue(alert.UpdateBy)

	state.Condition, err = alert.ConditionToTerraform()
	if err != nil {
		addErr(&resp.Diagnostics, err, operationRead, SigNozAlert)
		return
	}

	state.Labels, diag = alert.LabelsToTerraform()
	resp.Diagnostics.Append(diag...)

	state.PreferredChannels, diag = alert.PreferredChannelsToTerraform()
	resp.Diagnostics.Append(diag...)

	// Set refreshed state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update updates the resource and sets the updated Terraform state on success.
func (r *alertResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Retrieve values from plan.
	var plan, state alertResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Generate API request body from plan.
	var err error
	alertUpdate := &model.Alert{
		ID:        state.ID.ValueString(),
		Alert:     plan.Alert.ValueString(),
		AlertType: plan.AlertType.ValueString(),
		Annotations: model.AlertAnnotations{
			Description: plan.Description.ValueString(),
			Summary:     plan.Summary.ValueString(),
		},
		BroadcastToAll: plan.BroadcastToAll.ValueBool(),
		Disabled:       plan.Disabled.ValueBool(),
		EvalWindow:     plan.EvalWindow.ValueString(),
		Frequency:      plan.Frequency.ValueString(),
		RuleType:       plan.RuleType.ValueString(),
		Source:         plan.Source.ValueString(),
		State:          state.State.ValueString(),
		Version:        plan.Version.ValueString(),
		CreateAt:       state.CreateAt.ValueString(),
		CreateBy:       state.CreateBy.ValueString(),
		UpdateAt:       state.UpdateAt.ValueString(),
		UpdateBy:       state.UpdateBy.ValueString(),
	}

	err = alertUpdate.SetCondition(plan.Condition)
	if err != nil {
		addErr(&resp.Diagnostics, err, operationUpdate, SigNozAlert)
		return
	}

	alertUpdate.SetLabels(plan.Labels, plan.Severity)
	alertUpdate.SetPreferredChannels(plan.PreferredChannels)

	// Update existing alert.
	err = r.client.UpdateAlert(ctx, state.ID.ValueString(), alertUpdate)
	if err != nil {
		addErr(&resp.Diagnostics, err, operationUpdate, SigNozAlert)
		return
	}

	// Instead of fetching fresh state (which causes timestamp inconsistencies),
	// we'll use the plan data and preserve the original timestamps from state.
	// This avoids the "inconsistent result" error while maintaining data integrity.

	// Debug: Log what we're comparing
	tflog.Debug(ctx, "Update: Comparing condition values", map[string]any{
		"planCondition":  plan.Condition.ValueString(),
		"stateCondition": state.Condition.ValueString(),
		"areEqual":       plan.Condition.ValueString() == state.Condition.ValueString(),
	})

	// Only update condition if the user explicitly changed it in their config
	// This prevents drift from API formatting differences
	if !state.Condition.IsNull() && !state.Condition.IsUnknown() {
		// Compare JSON semantically to handle formatting differences
		if areJSONsSemanticallyEqual(plan.Condition.ValueString(), state.Condition.ValueString()) {
			plan.Condition = state.Condition
		}
		// If they're semantically different, let the plan value go through (user made a change)
	}

	// Preserve server-managed fields from current state
	plan.ID = state.ID
	plan.CreateAt = state.CreateAt
	plan.CreateBy = state.CreateBy
	plan.UpdateAt = state.UpdateAt
	plan.UpdateBy = state.UpdateBy
	plan.Source = state.Source
	plan.State = state.State

	// Set refreshed state.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
}



// areJSONsSemanticallyEqual compares two JSON strings semantically
func areJSONsSemanticallyEqual(json1, json2 string) bool {
	var data1, data2 interface{}
	
	if err := json.Unmarshal([]byte(json1), &data1); err != nil {
		return false
	}
	
	if err := json.Unmarshal([]byte(json2), &data2); err != nil {
		return false
	}
	
	// Normalize both by removing default fields
	normalized1 := removeDefaultFields(data1)
	normalized2 := removeDefaultFields(data2)
	
	// Marshal back to JSON for comparison
	bytes1, err := json.Marshal(normalized1)
	if err != nil {
		return false
	}
	
	bytes2, err := json.Marshal(normalized2)
	if err != nil {
		return false
	}
	
	normalized1Str := string(bytes1)
	normalized2Str := string(bytes2)
	
	// Debug: Print the normalized JSONs
	fmt.Printf("Normalized JSON 1: %s\n", normalized1Str)
	fmt.Printf("Normalized JSON 2: %s\n", normalized2Str)
	fmt.Printf("Are equal: %t\n", normalized1Str == normalized2Str)
	
	return normalized1Str == normalized2Str
}

// Delete deletes the resource and removes the Terraform state on success.
func (r *alertResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Retrieve values from state.
	var state alertResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Delete existing alert.
	err := r.client.DeleteAlert(ctx, state.ID.ValueString())
	if err != nil {
		addErr(&resp.Diagnostics, err, operationDelete, SigNozAlert)
		return
	}
}

// ImportState imports Terraform state into the resource.
func (r *alertResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Retrieve import ID and save to id attribute.
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
