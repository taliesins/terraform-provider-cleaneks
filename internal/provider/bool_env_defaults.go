package provider

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EnvDefaultBool returns a static string value default handler.
//
// Use EnvDefaultBool if a static default value for a string should be set.
func EnvDefaultBool(envName string, defaultVal bool) defaults.Bool {
	return envDefaultBoolDefault{
		envName:    envName,
		defaultVal: defaultVal,
	}
}

// envDefaultDefault is static value default handler that
// sets a value on a string attribute.
type envDefaultBoolDefault struct {
	envName    string
	defaultVal bool
}

// Description returns a human-readable description of the default value handler.
func (d envDefaultBoolDefault) Description(_ context.Context) string {
	return fmt.Sprintf("value defaults to value of an environment variable called %s, if environment variable does not exist or can't be converted to boolean then it defaults to %t", d.envName, d.defaultVal)
}

// MarkdownDescription returns a markdown description of the default value handler.
func (d envDefaultBoolDefault) MarkdownDescription(_ context.Context) string {
	return fmt.Sprintf("value defaults to value of an environment variable called `%s`, if environment variable does not exist or can't be converted to boolean then it defaults to `%t`", d.envName, d.defaultVal)
}

// DefaultBool implements the static default value logic.
func (d envDefaultBoolDefault) DefaultBool(_ context.Context, req defaults.BoolRequest, resp *defaults.BoolResponse) {
	value := d.defaultVal
	if d.envName != "" {
		envValue, ok := os.LookupEnv(d.envName)
		if ok {
			boolValue, err := strconv.ParseBool(envValue)
			if err == nil {
				value = boolValue
			}
		}
	}
	resp.PlanValue = types.BoolValue(value)
}
