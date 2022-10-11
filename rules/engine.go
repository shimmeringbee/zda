package rules

import (
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"io"
	"io/fs"
	"strings"
)

type Engine struct {
	RuleSets map[string]RuleSet
	Rules    []CompiledRule
}

type Capabilities struct {
	Add    map[string]interface{}
	Remove map[string]interface{}
}

type Actions struct {
	Capabilities Capabilities
}

type Rule struct {
	Description string
	Filter      string
	Actions     Actions
	Children    []Rule
}

type CompiledRule struct {
	Description string
	Filter      *vm.Program
	Actions     Actions
	Children    []CompiledRule
}

type RuleSet struct {
	Name      string
	DependsOn []string
	Rules     []Rule
}

type InputProductData struct {
	Name         string
	Manufacturer string
	Version      string
	Serial       string
}

type InputNode struct {
	ManufacturerCode uint16
	Type             string
}

type InputEndpoint struct {
	ID          uint8
	ProfileID   uint16
	DeviceID    uint16
	InClusters  []uint16
	OutClusters []uint16
}

type Input struct {
	Product  InputProductData
	Node     InputNode
	Endpoint InputEndpoint
}

type Output struct {
	Capabilities map[string]interface{}
}

func (e *Engine) LoadString(_ string) error {
	panic("not yet implemented")
}

func (e *Engine) LoadReader(_ io.Reader) error {
	panic("not yet implemented")
}

func (e *Engine) LoadFS(_ fs.FS) error {
	panic("not yet implemented")
}

func (e *Engine) CompileRules() error {
	alreadyLoaded := map[string]bool{}

	for k := range e.RuleSets {
		alreadyLoaded[k] = false
	}

	for k := range e.RuleSets {
		if !alreadyLoaded[k] {
			if err := e.compileRuleSet(alreadyLoaded, []string{}, k); err != nil {
				return err
			}
		}
	}

	return nil
}

func (e *Engine) compileRuleSet(alreadyLoaded map[string]bool, trail []string, name string) error {
	rs, ok := e.RuleSets[name]
	if !ok {
		return fmt.Errorf("ruleset missing dependency: %s->%s", strings.Join(trail, "->"), name)
	}

	trail = append(trail, rs.Name)

	for _, k := range rs.DependsOn {
		for _, t := range trail {
			if k == t {
				return fmt.Errorf("ruleset circular dependency: %s->%s", strings.Join(trail, "->"), k)
			}
		}

		if !alreadyLoaded[k] {
			if err := e.compileRuleSet(alreadyLoaded, trail, k); err != nil {
				return err
			}
		}
	}

	if cr, err := compileRules(rs.Rules); err != nil {
		return fmt.Errorf("ruleset compilation: %s: %w", strings.Join(trail, "->"), err)
	} else {
		e.Rules = append(e.Rules, cr...)
	}

	alreadyLoaded[name] = true

	return nil
}

func compileRules(rules []Rule) ([]CompiledRule, error) {
	var compiledRules []CompiledRule

	for _, rule := range rules {
		cf, err := expr.Compile(rule.Filter, expr.Env(Input{}))
		if err != nil {
			return nil, fmt.Errorf("filter compilation: %w", err)
		}

		if childCompiledRules, err := compileRules(rule.Children); err != nil {
			return nil, fmt.Errorf("%s: %w", rule.Description, err)
		} else {
			compiledRules = append(compiledRules, CompiledRule{
				Description: rule.Description,
				Filter:      cf,
				Actions:     rule.Actions,
				Children:    childCompiledRules,
			})
		}
	}

	return compiledRules, nil
}

func (e *Engine) Execute(_ Input) (Output, error) {
	panic("not yet implemented")
}
