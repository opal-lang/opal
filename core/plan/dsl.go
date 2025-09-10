package plan

import "time"

// PlanElement represents a buildable element in an execution plan
type PlanElement interface {
	Build() ExecutionStep
}

// PlanBuilder provides the main interface for building execution plans
type PlanBuilder struct {
	elements []PlanElement
}

// CommandElement represents a command to be executed
type CommandElement struct {
	command     string
	description string
	metadata    map[string]string
}

// DecoratorElement represents a decorator that wraps other elements
type DecoratorElement struct {
	name          string
	decoratorType string
	parameters    map[string]interface{}
	description   string
	timing        *TimingInfo
	children      []PlanElement
	imports       []string
}

// ChildElement represents a collection of nested elements
type ChildElement struct {
	description string
	elements    []PlanElement
}

// ConditionalElement represents conditional execution logic
type ConditionalElement struct {
	variable       string
	currentValue   string
	selectedBranch string
	reason         string
	branches       []BranchInfo
	children       []PlanElement
}

// ParallelElement represents concurrent execution of multiple elements
type ParallelElement struct {
	description      string
	concurrency      int
	failOnFirstError bool
	children         []PlanElement
}

// SequenceElement represents sequential execution of multiple elements
type SequenceElement struct {
	description string
	children    []PlanElement
}

// NewPlan creates a new plan builder
func NewPlan() *PlanBuilder {
	return &PlanBuilder{
		elements: make([]PlanElement, 0),
	}
}

// Add adds an element to the plan
func (pb *PlanBuilder) Add(element PlanElement) *PlanBuilder {
	pb.elements = append(pb.elements, element)
	return pb
}

// Build builds the complete execution plan
func (pb *PlanBuilder) Build() *ExecutionPlan {
	plan := NewExecutionPlan()
	for _, element := range pb.elements {
		plan.AddStep(element.Build())
	}
	return plan
}

// Command creates a new command element
func Command(cmd string) *CommandElement {
	return &CommandElement{
		command:  cmd,
		metadata: make(map[string]string),
	}
}

// WithDescription adds a description to the command
func (ce *CommandElement) WithDescription(desc string) *CommandElement {
	ce.description = desc
	return ce
}

// WithMetadata adds metadata to the command
func (ce *CommandElement) WithMetadata(key, value string) *CommandElement {
	ce.metadata[key] = value
	return ce
}

// Build converts the command element to an execution step
func (ce *CommandElement) Build() ExecutionStep {
	description := ce.description
	if description == "" {
		description = "Execute: " + ce.command
	}

	return ExecutionStep{
		Type:        StepShell,
		Description: description,
		Command:     ce.command,
		Metadata:    ce.metadata,
	}
}

// Decorator creates a new decorator element
func Decorator(name string) *DecoratorElement {
	return &DecoratorElement{
		name:          name,
		decoratorType: "block", // default
		parameters:    make(map[string]interface{}),
		children:      make([]PlanElement, 0),
		imports:       make([]string, 0),
	}
}

// WithType sets the decorator type (function, block, pattern)
func (de *DecoratorElement) WithType(decoratorType string) *DecoratorElement {
	de.decoratorType = decoratorType
	return de
}

// WithParameter adds a parameter to the decorator
func (de *DecoratorElement) WithParameter(key string, value interface{}) *DecoratorElement {
	de.parameters[key] = value
	return de
}

// WithDescription sets the decorator description
func (de *DecoratorElement) WithDescription(desc string) *DecoratorElement {
	de.description = desc
	return de
}

// WithTimeout adds timeout timing information
func (de *DecoratorElement) WithTimeout(timeout time.Duration) *DecoratorElement {
	if de.timing == nil {
		de.timing = &TimingInfo{}
	}
	de.timing.Timeout = &timeout
	return de
}

// WithRetry adds retry timing information
func (de *DecoratorElement) WithRetry(attempts int, delay time.Duration) *DecoratorElement {
	if de.timing == nil {
		de.timing = &TimingInfo{}
	}
	de.timing.RetryAttempts = attempts
	de.timing.RetryDelay = &delay
	return de
}

// WithConcurrency adds concurrency timing information
func (de *DecoratorElement) WithConcurrency(limit int) *DecoratorElement {
	if de.timing == nil {
		de.timing = &TimingInfo{}
	}
	de.timing.ConcurrencyLimit = limit
	return de
}

// AddImport adds an import requirement
func (de *DecoratorElement) AddImport(importPath string) *DecoratorElement {
	de.imports = append(de.imports, importPath)
	return de
}

// WithChildren sets the child elements
func (de *DecoratorElement) WithChildren(children ...PlanElement) *DecoratorElement {
	de.children = children
	return de
}

// AddChild adds a child element
func (de *DecoratorElement) AddChild(child PlanElement) *DecoratorElement {
	de.children = append(de.children, child)
	return de
}

// Build converts the decorator element to an execution step
func (de *DecoratorElement) Build() ExecutionStep {
	// All decorators use StepDecorator now - plugin friendly!
	stepType := StepDecorator

	description := de.description
	if description == "" {
		description = "Apply @" + de.name + " decorator"
	}

	// Build children
	children := make([]ExecutionStep, len(de.children))
	for i, child := range de.children {
		children[i] = child.Build()
	}

	return ExecutionStep{
		Type:        stepType,
		Description: description,
		Decorator: &DecoratorInfo{
			Name:       de.name,
			Type:       de.decoratorType,
			Parameters: de.parameters,
			Imports:    de.imports,
		},
		Timing:   de.timing,
		Children: children,
	}
}

// Child creates a new child element container
func Child() *ChildElement {
	return &ChildElement{
		elements: make([]PlanElement, 0),
	}
}

// WithDescription sets the child container description
func (ce *ChildElement) WithDescription(desc string) *ChildElement {
	ce.description = desc
	return ce
}

// Add adds an element to the child container
func (ce *ChildElement) Add(element PlanElement) *ChildElement {
	ce.elements = append(ce.elements, element)
	return ce
}

// Build converts the child element to an execution step
func (ce *ChildElement) Build() ExecutionStep {
	description := ce.description
	if description == "" {
		description = "Execute nested commands"
	}

	// Build all child elements
	children := make([]ExecutionStep, len(ce.elements))
	for i, element := range ce.elements {
		children[i] = element.Build()
	}

	return ExecutionStep{
		Type:        StepSequence,
		Description: description,
		Children:    children,
	}
}

// Conditional creates a new conditional element
func Conditional(variable, currentValue, selectedBranch string) *ConditionalElement {
	return &ConditionalElement{
		variable:       variable,
		currentValue:   currentValue,
		selectedBranch: selectedBranch,
		branches:       make([]BranchInfo, 0),
		children:       make([]PlanElement, 0),
	}
}

// WithReason sets the evaluation reason
func (ce *ConditionalElement) WithReason(reason string) *ConditionalElement {
	ce.reason = reason
	return ce
}

// AddBranch adds a branch to the conditional
func (ce *ConditionalElement) AddBranch(pattern, description string, willExecute bool) *ConditionalElement {
	ce.branches = append(ce.branches, BranchInfo{
		Pattern:     pattern,
		Description: description,
		WillExecute: willExecute,
	})
	return ce
}

// WithChildren sets the child elements for the selected branch
func (ce *ConditionalElement) WithChildren(children ...PlanElement) *ConditionalElement {
	ce.children = children
	return ce
}

// Build converts the conditional element to an execution step
func (ce *ConditionalElement) Build() ExecutionStep {
	// Build children
	children := make([]ExecutionStep, len(ce.children))
	for i, child := range ce.children {
		children[i] = child.Build()
	}

	return ExecutionStep{
		Type:        StepDecorator,
		Description: "Conditional execution based on " + ce.variable,
		Condition: &ConditionInfo{
			Variable: ce.variable,
			Branches: ce.branches,
			Evaluation: ConditionResult{
				CurrentValue:   ce.currentValue,
				SelectedBranch: ce.selectedBranch,
				Reason:         ce.reason,
			},
		},
		Children: children,
	}
}

// Parallel creates a new parallel element
func Parallel(concurrency int) *ParallelElement {
	return &ParallelElement{
		concurrency: concurrency,
		children:    make([]PlanElement, 0),
	}
}

// WithDescription sets the parallel element description
func (pe *ParallelElement) WithDescription(desc string) *ParallelElement {
	pe.description = desc
	return pe
}

// WithFailOnFirstError sets the fail-fast behavior
func (pe *ParallelElement) WithFailOnFirstError(fail bool) *ParallelElement {
	pe.failOnFirstError = fail
	return pe
}

// WithChildren sets the child elements to execute in parallel
func (pe *ParallelElement) WithChildren(children ...PlanElement) *ParallelElement {
	pe.children = children
	return pe
}

// AddChild adds a child element to execute in parallel
func (pe *ParallelElement) AddChild(child PlanElement) *ParallelElement {
	pe.children = append(pe.children, child)
	return pe
}

// Build converts the parallel element to an execution step
func (pe *ParallelElement) Build() ExecutionStep {
	description := pe.description
	if description == "" {
		description = "Execute in parallel"
	}

	// Build children
	children := make([]ExecutionStep, len(pe.children))
	for i, child := range pe.children {
		children[i] = child.Build()
	}

	return ExecutionStep{
		Type:        StepDecorator,
		Description: description,
		Timing: &TimingInfo{
			ConcurrencyLimit: pe.concurrency,
		},
		Children: children,
	}
}

// Sequence creates a new sequence element
func Sequence() *SequenceElement {
	return &SequenceElement{
		children: make([]PlanElement, 0),
	}
}

// WithDescription sets the sequence element description
func (se *SequenceElement) WithDescription(desc string) *SequenceElement {
	se.description = desc
	return se
}

// WithChildren sets the child elements to execute in sequence
func (se *SequenceElement) WithChildren(children ...PlanElement) *SequenceElement {
	se.children = children
	return se
}

// AddChild adds a child element to execute in sequence
func (se *SequenceElement) AddChild(child PlanElement) *SequenceElement {
	se.children = append(se.children, child)
	return se
}

// Build converts the sequence element to an execution step
func (se *SequenceElement) Build() ExecutionStep {
	description := se.description
	if description == "" {
		description = "Execute in sequence"
	}

	// Build children
	children := make([]ExecutionStep, len(se.children))
	for i, child := range se.children {
		children[i] = child.Build()
	}

	return ExecutionStep{
		Type:        StepSequence,
		Description: description,
		Children:    children,
	}
}
