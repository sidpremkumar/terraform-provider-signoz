package model

import (
	"encoding/json"
	"fmt"

	"github.com/SigNoz/terraform-provider-signoz/signoz/internal/utils"
	tfattr "github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
	Widgets                 interface{}              `json:"widgets"`
}

func (d Dashboard) PanelMapToTerraform() (types.String, error) {
	if d.PanelMap == nil {
		return types.StringNull(), nil
	}

	// Use json.Marshal for consistent formatting instead of structure.FlattenJsonToString
	panelMapJSON, err := json.Marshal(d.PanelMap)
	if err != nil {
		return types.StringNull(), err
	}

	return types.StringValue(string(panelMapJSON)), nil
}

func (d Dashboard) VariablesToTerraform() (types.String, error) {
	// Use json.Marshal for consistent formatting instead of structure.FlattenJsonToString
	variablesJSON, err := json.Marshal(d.Variables)
	if err != nil {
		return types.StringValue("{}"), err
	}

	return types.StringValue(string(variablesJSON)), nil
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
	if d.Widgets == nil {
		return types.StringValue("[]"), nil
	}

	// Use json.Marshal directly without additional normalization
	// This ensures consistency with what was sent during creation
	b, err := json.Marshal(d.Widgets)
	if err != nil {
		return types.StringValue("[]"), err
	}

	return types.StringValue(string(b)), nil
}

func (d *Dashboard) SetVariables(tfVariables types.String) error {
	variablesStr := tfVariables.ValueString()
	if variablesStr == "" {
		d.Variables = make(map[string]interface{})
		return nil
	}

	// Use json.Unmarshal for consistent parsing instead of structure.ExpandJsonFromString
	var variables map[string]interface{}
	if err := json.Unmarshal([]byte(variablesStr), &variables); err != nil {
		return err
	}
	d.Variables = variables
	return nil
}

func (d *Dashboard) SetPanelMap(tfPanelMap types.String) error {
	panelMapStr := tfPanelMap.ValueString()
	if panelMapStr == "" {
		d.PanelMap = make(map[string]interface{})
		return nil
	}

	// Use json.Unmarshal for consistent parsing instead of structure.ExpandJsonFromString
	var panelMap map[string]interface{}
	if err := json.Unmarshal([]byte(panelMapStr), &panelMap); err != nil {
		return err
	}
	d.PanelMap = panelMap
	return nil
}

func (d *Dashboard) SetTags(tfTags types.List) {
	tags := utils.Map(tfTags.Elements(), func(value tfattr.Value) string {
		// Use ValueString() directly instead of trimming quotes
		// This ensures consistency with how the data is stored and retrieved
		return value.(types.String).ValueString()
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
	widgetsStr := tfWidgets.ValueString()
	if widgetsStr == "" {
		d.Widgets = []map[string]interface{}{}
		return nil
	}

	// Parse the JSON string into a slice of maps
	var widgets []map[string]interface{}
	if err := json.Unmarshal([]byte(widgetsStr), &widgets); err != nil {
		return fmt.Errorf("failed to parse widgets JSON: %w", err)
	}

	d.Widgets = widgets
	return nil
}

func (d *Dashboard) SetSourceIfEmpty(hostURL string) {
	d.Source = utils.WithDefault(d.Source, hostURL+"/dashboard")
}
