package rules

import (
	"github.com/antonmedv/expr"
	"github.com/stretchr/testify/assert"
	"testing"
)

func Test_compileRule(t *testing.T) {
	t.Run("returns an error if the filter compilation fails", func(t *testing.T) {
		r := Rule{
			Filter: "INVALID UNPARSABLE FILTER",
		}

		crs, err := compileRules([]Rule{r})
		assert.Error(t, err)
		assert.Nil(t, crs)
		assert.Contains(t, err.Error(), "filter compilation:")
	})

	t.Run("returns a compiled rule", func(t *testing.T) {
		r := Rule{
			Description: "On Off ZCL",
			Filter:      "0x0006 in Endpoint.InClusters",
			Actions: Actions{
				Capabilities: Capabilities{
					Add: map[string]interface{}{
						"ZclOnOff": nil,
					},
				},
			},
		}

		cr, err := compileRules([]Rule{r})
		assert.NoError(t, err)

		assert.Equal(t, r.Description, cr[0].Description)
		assert.NotNil(t, cr[0].Filter)
		assert.Equal(t, r.Actions, cr[0].Actions)
		assert.Nil(t, r.Children)
	})
}

func TestEngine_CompileRules(t *testing.T) {
	t.Run("raises an error if a depended on ruleset is not loaded", func(t *testing.T) {
		e := Engine{
			RuleSets: map[string]RuleSet{
				"one": {
					Name:      "one",
					DependsOn: []string{"two"},
				},
			},
		}

		err := e.CompileRules()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ruleset missing dependency: one->two")
	})

	t.Run("raises an error if there is a circular dependency", func(t *testing.T) {
		e := Engine{
			RuleSets: map[string]RuleSet{
				"one": {
					Name:      "one",
					DependsOn: []string{"two"},
				},
				"two": {
					Name:      "two",
					DependsOn: []string{"one"},
				},
			},
		}

		err := e.CompileRules()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ruleset circular dependency: one->two->one")
	})

	t.Run("raises an error if a rule fails to compile", func(t *testing.T) {
		e := Engine{
			RuleSets: map[string]RuleSet{
				"one": {
					Name: "one",
					Rules: []Rule{
						{
							Description: "this rule",
							Filter:      "INVALID UNPARSABLE FILTER",
						},
					},
				},
			},
		}

		err := e.CompileRules()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ruleset compilation: one: filter compilation:")
	})

	t.Run("successfully compiles nested rules and resolves execution order", func(t *testing.T) {
		e := Engine{
			RuleSets: map[string]RuleSet{
				"one": {
					Name:      "one",
					DependsOn: []string{"two"},
					Rules: []Rule{
						{
							Description: "one",
							Filter:      "1 == 1",
						},
						{
							Description: "two",
							Filter:      "1 == 1",
							Children: []Rule{
								{
									Description: "two-one",
									Filter:      "1 == 1",
								},
							},
						},
					},
				},
				"two": {
					Name: "two",
					Rules: []Rule{
						{
							Description: "three",
							Filter:      "1 == 1",
						},
					},
				},
			},
		}

		vm, _ := expr.Compile("1 == 1", expr.Env(Input{}))

		expectedRules := []CompiledRule{
			{
				Description: "three",
				Filter:      vm,
			},
			{
				Description: "one",
				Filter:      vm,
			},
			{
				Description: "two",
				Filter:      vm,
				Children: []CompiledRule{
					{
						Description: "two-one",
						Filter:      vm,
					},
				},
			},
		}

		assert.NoError(t, e.CompileRules())
		assert.Equal(t, expectedRules, e.Rules)
	})
}

func TestEngine_Execute(t *testing.T) {
	t.Run("executes all rules that match, including any descendants", func(t *testing.T) {
		i := Input{
			Product: InputProductData{Manufacturer: "manufacturer"},
		}

		match, err := expr.Compile("'manufacturer' == Product.Manufacturer", expr.Env(Input{}))
		assert.NoError(t, err)
		nomatch, err := expr.Compile("'other manufacturer' == Product.Manufacturer", expr.Env(Input{}))
		assert.NoError(t, err)

		e := Engine{
			Rules: []CompiledRule{
				{
					Filter: nomatch,
					Actions: Actions{
						Capabilities: Capabilities{
							Add: map[string]interface{}{"one": nil},
						},
					},
				},
				{
					Filter: match,
					Actions: Actions{
						Capabilities: Capabilities{
							Add: map[string]interface{}{"two": nil},
						},
					},
					Children: []CompiledRule{
						{
							Filter: match,
							Actions: Actions{
								Capabilities: Capabilities{
									Add: map[string]interface{}{"three": nil},
								},
							},
							Children: []CompiledRule{
								{
									Filter: match,
									Actions: Actions{
										Capabilities: Capabilities{
											Add: map[string]interface{}{"four": nil},
										},
									},
								},
							},
						},
					},
				},
				{
					Filter: match,
					Actions: Actions{
						Capabilities: Capabilities{
							Remove: map[string]interface{}{"three": nil},
						},
					},
				},
			},
		}

		o, err := e.Execute(i)
		assert.NoError(t, err)

		assert.NotContains(t, o.Capabilities, "one")
		assert.Contains(t, o.Capabilities, "two")
		assert.NotContains(t, o.Capabilities, "three")
		assert.Contains(t, o.Capabilities, "four")
	})
}
