package rules

import (
	"encoding/json"
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
	Node     InputNode
	Self     uint8
	Product  map[uint8]InputProductData
	Endpoint map[uint8]InputEndpoint
}

type Output struct {
	Capabilities map[string]interface{}
}

func New() Engine {
	return Engine{
		RuleSets: map[string]RuleSet{},
		Rules:    nil,
	}
}

func (e *Engine) LoadReader(r io.Reader) error {
	rs := RuleSet{}

	if err := json.NewDecoder(r).Decode(&rs); err != nil {
		return err
	}

	e.RuleSets[rs.Name] = rs
	return nil
}

func (e *Engine) LoadFS(lFS fs.FS) error {
	return fs.WalkDir(lFS, ".", func(path string, d fs.DirEntry, err error) error {
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".json") {
			if f, err := lFS.Open(d.Name()); err != nil {
				return err
			} else {
				defer func() {
					_ = f.Close()
				}()

				err = e.LoadReader(f)
			}
		}

		return nil
	})
}

func (e *Engine) CompileRules() error {
	alreadyLoaded := map[string]bool{}

	for k := range e.RuleSets {
		alreadyLoaded[k] = false
	}

	for k := range e.RuleSets {
		if !alreadyLoaded[k] {
			if err := e.compileRuleSet(alreadyLoaded, nil, k); err != nil {
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

func (e *Engine) Execute(i Input) (Output, error) {
	o := Output{
		Capabilities: map[string]interface{}{},
	}

	for _, r := range e.Rules {
		if err := e.executeRule(i, &o, r); err != nil {
			return o, err
		}
	}

	return o, nil
}

func (e *Engine) executeRule(i Input, o *Output, r CompiledRule) error {
	execOut, err := expr.Run(r.Filter, i)
	if err != nil {
		return fmt.Errorf("rule %s: execution error: %w", r.Description, err)
	}

	if match, ok := execOut.(bool); ok {
		if !match {
			return nil
		}
	} else {
		return fmt.Errorf("rule %s: filter returned non boolean", r.Description)
	}

	for k, v := range r.Actions.Capabilities.Add {
		o.Capabilities[k] = v
	}

	for k := range r.Actions.Capabilities.Remove {
		delete(o.Capabilities, k)
	}

	for _, sr := range r.Children {
		if err := e.executeRule(i, o, sr); err != nil {
			return fmt.Errorf("rule %s: child error: %w", r.Description, err)
		}
	}

	return nil
}
