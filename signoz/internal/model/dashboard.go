package model

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/SigNoz/terraform-provider-signoz/signoz/internal/utils"
	tfattr "github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-sdk/helper/structure"
)

// Dashboard model.
type Dashboard struct {
	CollapsableRowsMigrated bool                     `json:"collapsableRowsMigrated"`
	Description             string                   `json:"description"`
	Layout                  []map[string]interface{} `json:"layout"`
	Name                    string                   `json:"name"`
	PanelMap                map[string]interface{}   `json:"panelMap,omitempty"`
	Source                  string                   `json:"source"`
	Tags                    []string                 `json:"tags"`
	Title                   string                   `json:"title"`
	UploadedGrafana         bool                     `json:"uploadedGrafana"`
	Variables               map[string]interface{}   `json:"variables"`
	Version                 string                   `json:"version,omitempty"`
	Widgets                 []map[string]interface{} `json:"widgets"`
}

// UnmarshalJSON implements custom JSON unmarshaling for Dashboard
func (d *Dashboard) UnmarshalJSON(data []byte) error {
	// Create a temporary struct to unmarshal the JSON
	type DashboardAlias Dashboard
	aux := &struct {
		*DashboardAlias
		Widgets interface{} `json:"widgets"`
	}{
		DashboardAlias: (*DashboardAlias)(d),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	// Handle widgets field - it might be a string or an array
	if aux.Widgets != nil {
		switch v := aux.Widgets.(type) {
		case string:
			// If widgets is a string, try to unmarshal it as JSON
			if v != "" {
				var widgets []map[string]interface{}
				if err := json.Unmarshal([]byte(v), &widgets); err != nil {
					return fmt.Errorf("failed to unmarshal widgets string: %w", err)
				}
				d.Widgets = widgets
			} else {
				d.Widgets = []map[string]interface{}{}
			}
		case []interface{}:
			// If widgets is already an array, convert it to the expected type
			widgets := make([]map[string]interface{}, len(v))
			for i, item := range v {
				if widgetMap, ok := item.(map[string]interface{}); ok {
					widgets[i] = widgetMap
				} else {
					return fmt.Errorf("widget item %d is not a map", i)
				}
			}
			d.Widgets = widgets
		default:
			return fmt.Errorf("unexpected type for widgets: %T", v)
		}
	}

	return nil
}

func (d Dashboard) PanelMapToTerraform() (types.String, error) {
	if d.PanelMap == nil {
		return types.StringNull(), nil
	}
	panelMap, err := structure.FlattenJsonToString(d.PanelMap)
	if err != nil {
		return types.StringNull(), err
	}

	return types.StringValue(panelMap), nil
}

func (d Dashboard) VariablesToTerraform() (types.String, error) {
	variables, err := structure.FlattenJsonToString(d.Variables)
	if err != nil {
		return types.StringValue(""), err
	}

	return types.StringValue(variables), nil
}

func (d Dashboard) TagsToTerraform() (types.List, diag.Diagnostics) {
	tags := utils.Map(d.Tags, func(value string) tfattr.Value {
		return types.StringValue(value)
	})

	return types.ListValue(types.StringType, tags)
}

func (d Dashboard) LayoutToTerraform() (types.String, error) {
	b, err := json.Marshal(d.Layout)
	if err != nil {
		return types.StringValue(""), err
	}
	return types.StringValue(string(b)), nil
}

func (d Dashboard) WidgetsToTerraform() (types.String, error) {
	b, err := json.MarshalIndent(d.Widgets, "", "  ")
	if err != nil {
		return types.StringValue(""), err
	}
	return types.StringValue(string(b)), nil
}

func (d *Dashboard) SetVariables(tfVariables types.String) error {
	variables, err := structure.ExpandJsonFromString(tfVariables.ValueString())
	if err != nil {
		return err
	}
	d.Variables = variables
	return nil
}

func (d *Dashboard) SetPanelMap(tfPanelMap types.String) error {
	if tfPanelMap.ValueString() == "" {
		d.PanelMap = make(map[string]interface{})
		return nil
	}
	panelMap, err := structure.ExpandJsonFromString(tfPanelMap.ValueString())
	if err != nil {
		return err
	}
	d.PanelMap = panelMap
	return nil
}

func (d *Dashboard) SetTags(tfTags types.List) {
	tags := utils.Map(tfTags.Elements(), func(value tfattr.Value) string {
		return strings.Trim(value.String(), "\"")
	})
	d.Tags = tags
}

func (d *Dashboard) SetLayout(tfLayout types.String) error {
	var layout []map[string]interface{}
	err := json.Unmarshal([]byte(tfLayout.ValueString()), &layout)
	if err != nil {
		return err
	}
	d.Layout = layout
	return nil
}

func (d *Dashboard) SetWidgets(tfWidgets types.String) error {
	var widgets []map[string]interface{}

	// Check if the input is already a JSON string that needs parsing
	widgetsStr := tfWidgets.ValueString()
	if widgetsStr != "" {
		// Try to parse as JSON first
		if err := json.Unmarshal([]byte(widgetsStr), &widgets); err != nil {
			// If it's not valid JSON, return the error
			return fmt.Errorf("failed to parse widgets JSON: %w", err)
		}
	}

	d.Widgets = widgets
	return nil
}

func (d *Dashboard) SetSourceIfEmpty(hostURL string) {
	d.Source = utils.WithDefault(d.Source, hostURL+"/dashboard")
}
