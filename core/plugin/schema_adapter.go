package plugin

import "github.com/builtwithtofu/sigil/core/types"

// DecoratorSchema adapts a plugin capability schema to the legacy parser schema.
func DecoratorSchema(capability Capability) types.DecoratorSchema {
	schema := capability.Schema()
	params := make(map[string]types.ParamSchema)
	order := make([]string, 0, len(schema.Params)+1)
	primary := ""

	if schema.Primary.Name != "" {
		primary = schema.Primary.Name
		params[schema.Primary.Name] = adaptParam(schema.Primary)
		order = append(order, schema.Primary.Name)
	}

	for _, param := range schema.Params {
		params[param.Name] = adaptParam(param)
		order = append(order, param.Name)
	}

	for _, secret := range schema.Secrets {
		if _, exists := params[secret]; exists {
			continue
		}
		params[secret] = types.ParamSchema{Name: secret, Type: types.TypeString}
		order = append(order, secret)
	}

	decoratorSchema := types.DecoratorSchema{
		Path:              capability.Path(),
		Kind:              decoratorKind(capability.Kind()),
		PrimaryParameter:  primary,
		Parameters:        params,
		ParameterOrder:    order,
		BlockRequirement:  types.BlockRequirement(schema.Block),
		SwitchesTransport: capability.Kind() == KindTransport,
	}

	if schema.Returns != "" {
		decoratorSchema.Returns = &types.ReturnSchema{Type: schema.Returns}
	}

	return decoratorSchema
}

func decoratorKind(kind CapabilityKind) types.DecoratorKindString {
	switch kind {
	case KindValue:
		return types.KindValue
	default:
		return types.KindExecution
	}
}

func adaptParam(param Param) types.ParamSchema {
	adapted := types.ParamSchema{
		Name:     param.Name,
		Type:     param.Type,
		Required: param.Required,
		Default:  param.Default,
		Examples: param.Examples,
		Minimum:  param.Minimum,
		Maximum:  param.Maximum,
	}

	if len(param.Enum) > 0 {
		adapted.Enum = make([]any, len(param.Enum))
		for i, value := range param.Enum {
			adapted.Enum[i] = value
		}
	}

	return adapted
}
