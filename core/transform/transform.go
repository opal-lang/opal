package transform

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/aledsdavies/devcmd/core/ast"
	"github.com/aledsdavies/devcmd/core/decorators"
	"github.com/aledsdavies/devcmd/core/ir"
)

// ================================================================================================
// AST TO IR TRANSFORMATION
// ================================================================================================

// TransformCommand converts an AST CommandDecl to IR ir.Node
func TransformCommand(cmd *ast.CommandDecl) (ir.Node, error) {
	return transformCommandBody(&cmd.Body)
}

// transformCommandBody converts a CommandBody to IR ir.Node
func transformCommandBody(body *ast.CommandBody) (ir.Node, error) {
	// Handle single content case - might be ir.Pattern or ir.Wrapper node
	if len(body.Content) == 1 {
		return transformCommandContent(body.Content[0])
	}

	// Multiple content items - combine into ir.CommandSeq
	var steps []ir.CommandStep

	for _, content := range body.Content {
		node, err := transformCommandContent(content)
		if err != nil {
			return nil, fmt.Errorf("error transforming command content: %w", err)
		}

		// Convert ir.Node to ir.CommandSeq and add steps
		switch n := node.(type) {
		case ir.CommandSeq:
			steps = append(steps, n.Steps...)
		case ir.Wrapper:
			// Block decorators can't be mixed with other content in command body
			return nil, fmt.Errorf("block decorators must be the only content in a command")
		case ir.Pattern:
			// ir.Pattern decorators can't be mixed with other content in command body
			return nil, fmt.Errorf("pattern decorators must be the only content in a command")
		default:
			return nil, fmt.Errorf("unknown node type: %T", node)
		}
	}

	return ir.CommandSeq{Steps: steps}, nil
}

// transformCommandContent converts CommandContent to ir.Node (properly handling ir.Pattern nodes)
func transformCommandContent(content ast.CommandContent) (ir.Node, error) {
	switch c := content.(type) {
	case *ast.ShellContent:
		step, err := transformShellContent(c)
		if err != nil {
			return nil, err
		}
		if step == nil {
			return ir.CommandSeq{Steps: []ir.CommandStep{}}, nil
		}
		return ir.CommandSeq{Steps: []ir.CommandStep{*step}}, nil
	case *ast.ShellChain:
		step, err := transformShellChain(c)
		if err != nil {
			return nil, err
		}
		if step == nil {
			return ir.CommandSeq{Steps: []ir.CommandStep{}}, nil
		}
		return ir.CommandSeq{Steps: []ir.CommandStep{*step}}, nil
	case *ast.BlockDecorator:
		return transformBlockDecoratorToWrapper(c)
	case *ast.PatternDecorator:
		return transformPatternDecoratorToPattern(c)
	case *ast.ActionDecorator:
		step, err := transformActionDecorator(c)
		if err != nil {
			return nil, err
		}
		if step == nil {
			return ir.CommandSeq{Steps: []ir.CommandStep{}}, nil
		}
		return ir.CommandSeq{Steps: []ir.CommandStep{*step}}, nil
	default:
		return nil, fmt.Errorf("unsupported command content type: %T", content)
	}
}

// transformShellContent converts ShellContent to ir.CommandStep with structured content
func transformShellContent(shell *ast.ShellContent) (*ir.CommandStep, error) {
	var elements []ir.ChainElement
	var contentParts []ir.ContentPart

	for _, part := range shell.Parts {
		switch p := part.(type) {
		case *ast.TextPart:
			contentParts = append(contentParts, ir.ContentPart{
				Kind: ir.PartKindLiteral,
				Text: p.Text,
				Span: createSourceSpan(p.Position()),
			})
		case *ast.TextStringPart:
			contentParts = append(contentParts, ir.ContentPart{
				Kind: ir.PartKindLiteral,
				Text: p.Text,
				Span: createSourceSpan(p.Position()),
			})
		case *ast.ValueDecorator:
			// Preserve value decorators as structured content parts
			args, err := transformDecoratorArgs(p.Args)
			if err != nil {
				return nil, fmt.Errorf("error transforming value decorator args: %w", err)
			}

			contentParts = append(contentParts, ir.ContentPart{
				Kind:          ir.PartKindDecorator,
				DecoratorName: p.Name,
				DecoratorArgs: args,
				Span:          createSourceSpan(p.Position()),
			})
		case *ast.ActionDecorator:
			// Action decorators in shell chains become separate chain elements
			args, err := transformDecoratorArgs(p.Args)
			if err != nil {
				return nil, fmt.Errorf("error transforming action decorator args: %w", err)
			}

			actionElement := ir.ChainElement{
				Kind: ir.ElementKindAction,
				Name: p.Name,
				Args: args,
				Span: createSourceSpan(p.Position()),
			}
			elements = append(elements, actionElement)
		default:
			return nil, fmt.Errorf("unsupported shell part type: %T", part)
		}
	}

	// Add shell element with structured content if we have content parts
	if len(contentParts) > 0 {
		shellElement := ir.ChainElement{
			Kind:    ir.ElementKindShell,
			Content: &ir.ElementContent{Parts: contentParts},
			Span:    createSourceSpan(shell.Position()),
		}
		// Insert shell element at the beginning
		elements = append([]ir.ChainElement{shellElement}, elements...)
	}

	// If we have no elements, this might be an empty shell content
	if len(elements) == 0 {
		return nil, nil // Skip empty content
	}

	return &ir.CommandStep{
		Chain: elements,
		Span:  createSourceSpan(shell.Position()),
	}, nil
}

// transformShellChain converts ShellChain to ir.CommandStep with chained elements
func transformShellChain(chain *ast.ShellChain) (*ir.CommandStep, error) {
	var elements []ir.ChainElement

	for i, element := range chain.Elements {
		// Transform the shell content of this element to structured content
		var contentParts []ir.ContentPart

		for _, part := range element.Content.Parts {
			switch p := part.(type) {
			case *ast.TextPart:
				contentParts = append(contentParts, ir.ContentPart{
					Kind: ir.PartKindLiteral,
					Text: p.Text,
					Span: createSourceSpan(p.Position()),
				})
			case *ast.TextStringPart:
				contentParts = append(contentParts, ir.ContentPart{
					Kind: ir.PartKindLiteral,
					Text: p.Text,
					Span: createSourceSpan(p.Position()),
				})
			case *ast.ValueDecorator:
				args, err := transformDecoratorArgs(p.Args)
				if err != nil {
					return nil, fmt.Errorf("error transforming value decorator args in chain element %d: %w", i, err)
				}

				contentParts = append(contentParts, ir.ContentPart{
					Kind:          ir.PartKindDecorator,
					DecoratorName: p.Name,
					DecoratorArgs: args,
					Span:          createSourceSpan(p.Position()),
				})
			default:
				return nil, fmt.Errorf("unsupported shell part in chain element %d: %T", i, part)
			}
		}

		// Create chain element with structured content
		chainElement := ir.ChainElement{
			Kind:    ir.ElementKindShell,
			Content: &ir.ElementContent{Parts: contentParts},
			Span:    createSourceSpan(element.Position()),
		}

		// Set the operator for this element (from the AST Shell ChainElement.Operator)
		if element.Operator != "" {
			switch element.Operator {
			case "&&":
				chainElement.OpNext = ir.ChainOpAnd
			case "||":
				chainElement.OpNext = ir.ChainOpOr
			case "|":
				chainElement.OpNext = ir.ChainOpPipe
			case ">>":
				chainElement.OpNext = ir.ChainOpAppend
				chainElement.Target = element.Target // Set target file for append
			default:
				return nil, fmt.Errorf("unsupported shell operator: %s", element.Operator)
			}
		} else {
			chainElement.OpNext = ir.ChainOpNone // No operator (last element)
		}

		elements = append(elements, chainElement)
	}

	return &ir.CommandStep{
		Chain: elements,
		Span:  createSourceSpan(chain.Position()),
	}, nil
}

// transformBlockDecoratorToWrapper converts BlockDecorator to ir.Wrapper node
func transformBlockDecoratorToWrapper(block *ast.BlockDecorator) (ir.Wrapper, error) {
	// Transform parameters
	params, err := transformParameters(block.Args)
	if err != nil {
		return ir.Wrapper{}, fmt.Errorf("error transforming block decorator parameters: %w", err)
	}

	// Transform inner content to steps
	var steps []ir.CommandStep
	for _, content := range block.Content {
		node, err := transformCommandContent(content)
		if err != nil {
			return ir.Wrapper{}, fmt.Errorf("error transforming block decorator inner content: %w", err)
		}

		// Convert ir.Node to ir.CommandSeq steps
		switch n := node.(type) {
		case ir.CommandSeq:
			steps = append(steps, n.Steps...)
		case ir.Wrapper:
			// Nested block decorators need to be converted to ir.CommandStep with InnerSteps
			wrapperStep := ir.CommandStep{
				Chain: []ir.ChainElement{{
					Kind:       ir.ElementKindBlock,
					Name:       n.Kind,
					Args:       convertParamsToDecoratorParams(n.Params),
					InnerSteps: n.Inner.Steps, // Use ir.CommandSeq.Steps directly
				}},
			}
			steps = append(steps, wrapperStep)
		case ir.Pattern:
			return ir.Wrapper{}, fmt.Errorf("pattern decorators not allowed inside block decorators")
		default:
			return ir.Wrapper{}, fmt.Errorf("unknown node type in block decorator: %T", node)
		}
	}

	// Return actual ir.Wrapper node
	return ir.Wrapper{
		Kind:   block.Name,
		Params: params,
		Inner:  ir.CommandSeq{Steps: steps},
	}, nil
}

// transformPatternDecoratorToPattern converts PatternDecorator to ir.Pattern node
func transformPatternDecoratorToPattern(pattern *ast.PatternDecorator) (ir.Pattern, error) {
	// Transform parameters
	params, err := transformParameters(pattern.Args)
	if err != nil {
		return ir.Pattern{}, fmt.Errorf("error transforming pattern decorator parameters: %w", err)
	}

	// Transform branches
	branches := make(map[string]ir.CommandSeq)
	for _, branch := range pattern.Patterns {
		// Transform pattern content to ir.CommandSeq
		var steps []ir.CommandStep
		for _, content := range branch.Commands {
			node, err := transformCommandContent(content)
			if err != nil {
				return ir.Pattern{}, fmt.Errorf("error transforming pattern branch content: %w", err)
			}

			// Convert ir.Node to ir.CommandSeq steps
			switch n := node.(type) {
			case ir.CommandSeq:
				steps = append(steps, n.Steps...)
			case ir.Wrapper:
				// Block decorators inside pattern branches need to be converted to ir.CommandStep with InnerSteps
				wrapperStep := ir.CommandStep{
					Chain: []ir.ChainElement{{
						Kind:       ir.ElementKindBlock,
						Name:       n.Kind,
						Args:       convertParamsToDecoratorParams(n.Params),
						InnerSteps: n.Inner.Steps, // Use ir.CommandSeq.Steps directly
					}},
				}
				steps = append(steps, wrapperStep)
			case ir.Pattern:
				return ir.Pattern{}, fmt.Errorf("pattern decorators inside pattern branches not yet supported (would require complex nested evaluation)")
			default:
				return ir.Pattern{}, fmt.Errorf("unknown node type in pattern branch: %T", node)
			}
		}

		branchName := formatPatternName(branch.Pattern)
		branches[branchName] = ir.CommandSeq{Steps: steps}
	}

	// Return actual ir.Pattern node (not ir.CommandStep)
	return ir.Pattern{
		Kind:     pattern.Name,
		Params:   params,
		Branches: branches,
	}, nil
}

// transformActionDecorator converts ActionDecorator to action element
func transformActionDecorator(action *ast.ActionDecorator) (*ir.CommandStep, error) {
	// Transform parameters
	params, err := transformParameters(action.Args)
	if err != nil {
		return nil, fmt.Errorf("error transforming action decorator parameters: %w", err)
	}

	// Create action element
	actionElement := ir.ChainElement{
		Kind: ir.ElementKindAction,
		Name: action.Name,
		Args: convertParamsToDecoratorParams(params),
		Span: createSourceSpan(action.Position()),
	}

	return &ir.CommandStep{
		Chain: []ir.ChainElement{actionElement},
		Span:  createSourceSpan(action.Position()),
	}, nil
}

// ================================================================================================
// PARAMETER TRANSFORMATION
// ================================================================================================

// transformParameters converts AST NamedParameters to a map
func transformParameters(params []ast.NamedParameter) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	for i, param := range params {
		key := param.Name
		if key == "" {
			// Positional parameter - use index as key
			key = fmt.Sprintf("_pos_%d", i)
		}

		value, err := transformExpression(param.Value)
		if err != nil {
			return nil, fmt.Errorf("error transforming parameter value: %w", err)
		}

		result[key] = value
	}

	return result, nil
}

// transformExpression converts AST expressions to Go values
func transformExpression(expr ast.Expression) (interface{}, error) {
	switch e := expr.(type) {
	case *ast.StringLiteral:
		// Process all parts and combine them
		var result strings.Builder
		for _, part := range e.Parts {
			switch p := part.(type) {
			case *ast.TextStringPart:
				result.WriteString(p.Text)
			case *ast.ValueDecorator:
				// For now, just return the decorator as text - we'll expand later
				result.WriteString(p.String())
			}
		}
		return result.String(), nil
	case *ast.NumberLiteral:
		// Try to parse as int first, then float
		if intVal, err := strconv.Atoi(e.Value); err == nil {
			return intVal, nil
		}
		if floatVal, err := strconv.ParseFloat(e.Value, 64); err == nil {
			return floatVal, nil
		}
		return nil, fmt.Errorf("invalid number literal: %s", e.Value)
	case *ast.BooleanLiteral:
		return e.Value, nil
	case *ast.DurationLiteral:
		// Parse duration string
		if duration, err := time.ParseDuration(e.Value); err == nil {
			return duration, nil
		} else {
			return nil, fmt.Errorf("invalid duration: %s", e.Value)
		}
	case *ast.Identifier:
		// For identifiers, return as string for now
		// In a full implementation, this might resolve to variable values
		return e.Name, nil
	default:
		return nil, fmt.Errorf("unsupported expression type: %T", expr)
	}
}

// transformDecoratorArgs directly converts AST NamedParameter to DecoratorParam
func transformDecoratorArgs(astArgs []ast.NamedParameter) ([]decorators.Param, error) {
	params := make([]decorators.Param, 0, len(astArgs))

	for _, arg := range astArgs {
		value, err := transformExpression(arg.Value)
		if err != nil {
			return nil, fmt.Errorf("transforming argument expression: %w", err)
		}

		param := decorators.Param{
			ParamName:  arg.Name,
			ParamValue: value,
		}
		params = append(params, param)
	}

	return params, nil
}

// convertParamsToDecoratorParams converts map parameters to DecoratorParam slice
func convertParamsToDecoratorParams(params map[string]interface{}) []decorators.Param {
	var result []decorators.Param

	// Handle positional parameters first (those with _pos_ prefix)
	for i := 0; ; i++ {
		key := fmt.Sprintf("_pos_%d", i)
		if value, exists := params[key]; exists {
			result = append(result, decorators.Param{
				ParamName:  "", // Empty name for positional
				ParamValue: value,
			})
		} else {
			break
		}
	}

	// Handle named parameters
	for key, value := range params {
		if !strings.HasPrefix(key, "_pos_") {
			result = append(result, decorators.Param{
				ParamName:  key,
				ParamValue: value,
			})
		}
	}

	return result
}

// ================================================================================================
// HELPER FUNCTIONS
// ================================================================================================

// createSourceSpan converts AST Position to IR SourceSpan
func createSourceSpan(pos ast.Position) *ir.SourceSpan {
	return &ir.SourceSpan{
		File:   "", // File path would be set by caller
		Line:   pos.Line,
		Column: pos.Column,
		Length: 0, // Could be calculated from token range
	}
}

// formatPatternName converts pattern to string name (simplified for now)
func formatPatternName(pattern interface{}) string {
	return fmt.Sprintf("%v", pattern)
}
