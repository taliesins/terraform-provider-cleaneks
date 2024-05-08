package provider

import (
	"fmt"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// AttributeValueToString will attempt to execute the appropriate AttributeStringerFunc from the ones registered.
func AttributeValueToString(v attr.Value) string {
	if s, ok := v.(types.String); ok {
		return s.ValueString()
	}
	return v.String()
}

// ValueToListType ensures we have a types.List literal
func ValueToListType(v attr.Value) types.List {
	if vb, ok := v.(types.List); ok {
		return vb
	} else if vb, ok := v.(*types.List); ok {
		return *vb
	} else {
		panic(fmt.Sprintf("cannot pass type %T to conv.ValueToListType", v))
	}
}

func StringListToStrings(v attr.Value) []string {
	vt := ValueToListType(v)
	out := make([]string, len(vt.Elements()))
	for i, ve := range vt.Elements() {
		out[i] = AttributeValueToString(ve)
	}
	return out
}
