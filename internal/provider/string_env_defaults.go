package provider

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EnvDefaultString returns a static string value default handler.
//
// Use EnvDefaultString if a static default value for a string should be set.
func EnvDefaultString(envName string, defaultVal string) defaults.String {
	return envDefaultStringDefault{
		envName:    envName,
		defaultVal: defaultVal,
	}
}

// envDefaultDefault is static value default handler that
// sets a value on a string attribute.
type envDefaultStringDefault struct {
	envName    string
	defaultVal string
}

// Description returns a human-readable description of the default value handler.
func (d envDefaultStringDefault) Description(_ context.Context) string {
	return fmt.Sprintf("value defaults to value of an environment variable called %s, if environment variable does not exist then it defaults to %s", d.envName, d.defaultVal)
}

// MarkdownDescription returns a markdown description of the default value handler.
func (d envDefaultStringDefault) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("value defaults to value of an environment variable called `%s`, if environment variable does not exist then it defaults to `%s`", d.envName, d.defaultVal)
}

// DefaultString implements the static default value logic.
func (d envDefaultStringDefault) DefaultString(_ context.Context, req defaults.StringRequest, resp *defaults.StringResponse) {
	value := d.defaultVal
	if d.envName != "" {
		envValue, ok := os.LookupEnv(d.envName)
		if ok {
			value = envValue
		}
	}
	resp.PlanValue = types.StringValue(value)
}
