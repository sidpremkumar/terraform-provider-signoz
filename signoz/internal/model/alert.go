package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SigNoz/terraform-provider-signoz/signoz/internal/attr"
	"github.com/SigNoz/terraform-provider-signoz/signoz/internal/utils"
	tfattr "github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/helper/structure"
)

const (
	AlertTypeMetrics    = "METRIC_BASED_ALERT"
	AlertTypeLogs       = "LOGS_BASED_ALERT"
	AlertTypeTraces     = "TRACES_BASED_ALERT"
	AlertTypeExceptions = "EXCEPTIONS_BASED_ALERT"

	AlertRuleTypeThreshold = "threshold_rule"
	AlertRuleTypeProm      = "promql_rule"

	AlertSeverityCritical = "critical"
	AlertSeverityError    = "error"
	AlertSeverityWarning  = "warning"
	AlertSeverityInfo     = "info"

	AlertStateInactive = "inactive"
	AlertStatePending  = "pending"
	AlertStateFiring   = "firing"
	AlertStateDisabled = "disabled"

	AlertTerraformLabel = "managedBy:terraform"
)

//nolint:gochecknoglobals
var (
	AlertTypes      = []string{AlertTypeMetrics, AlertTypeLogs, AlertTypeTraces, AlertTypeExceptions}
	AlertRuleTypes  = []string{AlertRuleTypeThreshold, AlertRuleTypeProm}
	AlertSeverities = []string{AlertSeverityCritical, AlertSeverityError, AlertSeverityWarning, AlertSeverityInfo}
	AlertStates     = []string{AlertStateInactive, AlertStatePending, AlertStateFiring, AlertStateDisabled}
)

// Alert model.
type Alert struct {
	ID                string                 `json:"id"`
	Alert             string                 `json:"alert"`
	AlertType         string                 `json:"alertType"`
	Annotations       AlertAnnotations       `json:"annotations"`
	BroadcastToAll    bool                   `json:"broadcastToAll"`
	Condition         map[string]interface{} `json:"condition"`
	Disabled          bool                   `json:"disabled,omitempty"`
	EvalWindow        string                 `json:"evalWindow"`
	Frequency         string                 `json:"frequency"`
	Labels            map[string]string      `json:"labels"`
	PreferredChannels []string               `json:"preferredChannels"`
	RuleType          string                 `json:"ruleType"`
	Source            string                 `json:"source"`
	State             string                 `json:"state,omitempty"`
	Version           string                 `json:"version"`
	CreateAt          string                 `json:"createAt,omitempty"`
	CreateBy          string                 `json:"createBy,omitempty"`
	UpdateAt          string                 `json:"updateAt,omitempty"`
	UpdateBy          string                 `json:"updateBy,omitempty"`
}

// Alert Annotations model.
type AlertAnnotations struct {
	Description string `json:"description"`
	Summary     string `json:"summary"`
}

func (a Alert) GetID() string {
	return a.ID
}

func (a Alert) GetName() string {
	return a.Alert
}

func (a Alert) GetType() string {
	return a.AlertType
}

func (a Alert) ConditionToTerraform() (types.String, error) {
	// Normalize the condition to remove API-added default fields
	normalizedCondition := removeDefaultFields(a.Condition)
	
	// Convert back to map[string]interface{} for structure.FlattenJsonToString
	if normalizedMap, ok := normalizedCondition.(map[string]interface{}); ok {
		condition, err := structure.FlattenJsonToString(normalizedMap)
		if err != nil {
			return types.StringValue(""), err
		}
		return types.StringValue(condition), nil
	}
	
	// Fallback to original behavior if normalization fails
	condition, err := structure.FlattenJsonToString(a.Condition)
	if err != nil {
		return types.StringValue(""), err
	}
	return types.StringValue(condition), nil
}

func (a Alert) LabelsToTerraform() (types.Map, diag.Diagnostics) {
	elements := map[string]tfattr.Value{}
	terraformLabels := strings.Split(AlertTerraformLabel, ":")
	for key, value := range a.Labels {
		if key == attr.Severity || key == terraformLabels[0] {
			continue
		}
		elements[key] = types.StringValue(value)
	}
	return types.MapValue(types.StringType, elements)
}

func (a Alert) PreferredChannelsToTerraform() (types.List, diag.Diagnostics) {
	preferredChannels := utils.Map(a.PreferredChannels, func(value string) tfattr.Value {
		return types.StringValue(value)
	})

	return types.ListValue(types.StringType, preferredChannels)
}

func (a Alert) ToTerraform() interface{} {
	return map[string]interface{}{
		attr.ID:                a.ID,
		attr.Alert:             a.Alert,
		attr.AlertType:         a.AlertType,
		attr.Annotations:       a.Annotations,
		attr.BroadcastToAll:    a.BroadcastToAll,
		attr.Condition:         a.Condition,
		attr.Disabled:          a.Disabled,
		attr.EvalWindow:        a.EvalWindow,
		attr.Frequency:         a.Frequency,
		attr.Labels:            a.Labels,
		attr.PreferredChannels: a.PreferredChannels,
		attr.RuleType:          a.RuleType,
		attr.Source:            a.Source,
		attr.State:             a.State,
		attr.Version:           a.Version,
		attr.CreateAt:          a.CreateAt,
		attr.CreateBy:          a.CreateBy,
		attr.UpdateAt:          a.UpdateAt,
		attr.UpdateBy:          a.UpdateBy,
		// attr.Description:       a.Description,
		// attr.Summary:           a.Summary,
		// attr.Severity:          a.Severity,
	}
}

func (a *Alert) SetCondition(tfCondition types.String) error {
	fmt.Printf("SetCondition: Original condition: %s\n", tfCondition.ValueString())
	
	condition, err := structure.ExpandJsonFromString(tfCondition.ValueString())
	if err != nil {
		return err
	}

	// Normalize the condition to match API format
	normalizedCondition := normalizeCondition(condition)
	
	// Debug: Print the normalized condition
	normalizedBytes, _ := json.Marshal(normalizedCondition)
	fmt.Printf("SetCondition: Normalized condition: %s\n", string(normalizedBytes))
	
	a.Condition = normalizedCondition
	return nil
}

// normalizeCondition ensures the condition matches the API's expected format
func normalizeCondition(condition map[string]interface{}) map[string]interface{} {
	// Add default fields that the API expects
	if condition["compositeQuery"] != nil {
		if compositeQuery, ok := condition["compositeQuery"].(map[string]interface{}); ok {
			if builderQueries, ok := compositeQuery["builderQueries"].(map[string]interface{}); ok {
				for queryName, query := range builderQueries {
					if queryMap, ok := query.(map[string]interface{}); ok {
						// Add default fields that API adds
						if queryMap["IsAnomaly"] == nil {
							queryMap["IsAnomaly"] = false
						}
						if queryMap["QueriesUsedInFormula"] == nil {
							queryMap["QueriesUsedInFormula"] = nil
						}
						if queryMap["groupBy"] == nil {
							queryMap["groupBy"] = []interface{}{}
						}
						if queryName == "A" && queryMap["hidden"] == nil {
							queryMap["hidden"] = true
						}
						if queryName == "F1" {
							if queryMap["reduceTo"] == nil {
								queryMap["reduceTo"] = ""
							}
							if queryMap["spaceAggregation"] == nil {
								queryMap["spaceAggregation"] = ""
							}
							if queryMap["timeAggregation"] == nil {
								queryMap["timeAggregation"] = ""
							}
						}
					}
				}
			}
		}
	}

	// Add root-level default fields
	if condition["absentFor"] == nil {
		condition["absentFor"] = 0
	}
	if condition["alertOnAbsent"] == nil {
		condition["alertOnAbsent"] = false
	}

	return condition
}

// removeDefaultFields recursively removes API-added default fields that cause drift
func removeDefaultFields(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			// Skip API-added default fields that cause drift
			if isDefaultField(key, value) {
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

func (a *Alert) SetLabels(tfLabels types.Map, tfSeverity types.String) {
	labels := make(map[string]string)

	for key, value := range tfLabels.Elements() {
		labels[key] = strings.Trim(value.String(), "\"")
	}

	terraformLabel := strings.Split(AlertTerraformLabel, ":")
	labels[strings.TrimSpace(terraformLabel[0])] = strings.TrimSpace(terraformLabel[1])

	if tfSeverity.ValueString() != "" {
		labels[attr.Severity] = tfSeverity.ValueString()
	}

	a.Labels = labels
}

func (a *Alert) SetPreferredChannels(tfPreferredChannels types.List) {
	preferredChannels := utils.Map(tfPreferredChannels.Elements(), func(value tfattr.Value) string {
		return strings.Trim(value.String(), "\"")
	})
	a.PreferredChannels = preferredChannels
}

func (a *Alert) SetSourceIfEmpty(hostURL string) {
	a.Source = utils.WithDefault(a.Source, hostURL+"/alerts")
}
