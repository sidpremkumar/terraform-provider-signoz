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
	Widgets                 interface{}              `json:"widgets"`
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
	if d.Widgets == nil {
		return types.StringValue("[]"), nil
	}

	// First marshal to get the data
	b, err := json.Marshal(d.Widgets)
	if err != nil {
		return types.StringValue(""), err
	}

	// Parse it back to normalize the structure
	var normalized interface{}
	if err := json.Unmarshal(b, &normalized); err != nil {
		return types.StringValue(""), err
	}

	// Marshal with exact formatting to match API
	formatted, err := json.MarshalIndent(normalized, "", "  ")
	if err != nil {
		return types.StringValue(""), err
	}

	return types.StringValue(string(formatted)), nil
}

func (d *Dashboard) SetVariables(tfVariables types.String) error {
	variablesStr := tfVariables.ValueString()
	if variablesStr == "" {
		d.Variables = make(map[string]interface{})
		return nil
	}

	// Debug log the variables string we're trying to parse
	fmt.Printf("DEBUG: SetVariables attempting to parse: %q\n", variablesStr)

	variables, err := structure.ExpandJsonFromString(variablesStr)
	if err != nil {
		fmt.Printf("DEBUG: SetVariables parse error: %v\n", err)
		return fmt.Errorf("failed to parse variables JSON: %w", err)
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
	widgetsStr := tfWidgets.ValueString()
	if widgetsStr == "" {
		d.Widgets = []map[string]interface{}{}
		return nil
	}

	// Try to parse as JSON first
	var widgets interface{}
	if err := json.Unmarshal([]byte(widgetsStr), &widgets); err != nil {
		return fmt.Errorf("failed to parse widgets JSON: %w", err)
	}

	d.Widgets = widgets
	return nil
}

func (d *Dashboard) SetSourceIfEmpty(hostURL string) {
	d.Source = utils.WithDefault(d.Source, hostURL+"/dashboard")
}
