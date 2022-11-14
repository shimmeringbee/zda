package rules

import (
	"encoding/json"
	"fmt"
	"github.com/antonmedv/expr"
	"github.com/antonmedv/expr/vm"
	"github.com/shimmeringbee/zcl"
	"github.com/shimmeringbee/zigbee"
	"io"
	"io/fs"
	"sort"
	"strings"
)

type Engine struct {
	RuleSets map[string]RuleSet
	Rules    []CompiledRule
}

type CapabilityValues map[string]string

type CompiledCapabilityValues map[string]*vm.Program

type Capabilities struct {
	Add    map[string]CapabilityValues
	Remove map[string]CapabilityValues
}

type CompiledCapabilities struct {
	Add    map[string]CompiledCapabilityValues
	Remove map[string]CompiledCapabilityValues
}

type Actions struct {
	Capabilities Capabilities
}

type CompiledActions struct {
	Capabilities CompiledCapabilities
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
	Actions     CompiledActions
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
	Fn       Fn
}

type Fn struct{}

func (f Fn) Endpoint(e int) zigbee.Endpoint {
	return zigbee.Endpoint(e)
}

func (f Fn) ClusterID(c int) zigbee.ClusterID {
	return zigbee.ClusterID(c)
}

func (f Fn) AttributeID(a int) zcl.AttributeID {
	return zcl.AttributeID(a)
}

type Output struct {
	Capabilities map[string]map[string]interface{}
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
	var ruleSets []string

	for k := range e.RuleSets {
		alreadyLoaded[k] = false
		ruleSets = append(ruleSets, k)
	}

	/* Sort ruleset names before processing, this is primarily for ensuring tests pass predictably. */
	sort.Strings(ruleSets)
	for _, k := range ruleSets {
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
			ca, err := compileActions(rule.Actions)
			if err != nil {
				return nil, fmt.Errorf("%s: %w", rule.Description, err)
			}

			compiledRules = append(compiledRules, CompiledRule{
				Description: rule.Description,
				Filter:      cf,
				Actions:     ca,
				Children:    childCompiledRules,
			})
		}
	}

	return compiledRules, nil
}

func compileActions(a Actions) (CompiledActions, error) {
	addCapabilities, err := compileActionParameters(a.Capabilities.Add)
	if err != nil {
		return CompiledActions{}, fmt.Errorf("add capability: %w", err)
	}

	removeCapabilities, err := compileActionParameters(a.Capabilities.Remove)
	if err != nil {
		return CompiledActions{}, fmt.Errorf("remove capability: %w", err)
	}

	return CompiledActions{
		Capabilities: CompiledCapabilities{
			Add:    addCapabilities,
			Remove: removeCapabilities,
		},
	}, nil
}

func compileActionParameters(values map[string]CapabilityValues) (map[string]CompiledCapabilityValues, error) {
	ret := make(map[string]CompiledCapabilityValues)

	for k, val := range values {
		ccv, err := compileCapabilityValue(val)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", k, err)
		}
		ret[k] = ccv
	}

	return ret, nil
}

func compileCapabilityValue(c CapabilityValues) (CompiledCapabilityValues, error) {
	ret := make(CompiledCapabilityValues)

	for k, v := range c {
		ca, err := expr.Compile(v, expr.Env(Input{}))
		if err != nil {
			return nil, fmt.Errorf("attribute '%s' compilation: %w", k, err)
		}

		ret[k] = ca
	}

	return ret, nil
}

func (e *Engine) Execute(i Input) (Output, error) {
	o := Output{
		Capabilities: map[string]map[string]interface{}{},
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
		values := make(map[string]interface{})

		for valueName, valueProgram := range v {
			out, err := expr.Run(valueProgram, i)
			if err != nil {
				return fmt.Errorf("rule %s: value %s: errored: %w", valueName, r.Description, err)
			}
			values[valueName] = out
		}

		o.Capabilities[k] = values
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
