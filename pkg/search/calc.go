package search

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	"github.com/knetic/govaluate"
)

func searchCalcMode(expr string) []Result {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return []Result{
			{
				Name:    "Calculator",
				GUI:     false,
				Type:    "calc",
				Source:  "internal",
				Comment: "Usage: :calc 1+2*3, supports sin(), cos(), tan(), sinh(), cosh(), sqrt(), factorial(), log(), ln(), pi, e, min(), max(), abs(), round(), etc.",
				Command: "",
			},
		}
	}

	// Allow digits, letters, operators, parentheses, dots, commas
	allowed := regexp.MustCompile(`^[0-9+\-*/%^().!,\sA-Za-z]+$`)
	if !allowed.MatchString(expr) {
		expr = regexp.MustCompile(`[^0-9+\-*/%^().!,\sA-Za-z]`).ReplaceAllString(expr, "")
		if expr == "" {
			return []Result{
				{
					Name:    "Expression contained invalid characters",
					GUI:     false,
					Type:    "calc",
					Source:  "internal",
					Comment: "Invalid expression",
					Command: "",
				},
			}
		}
	}

	// Custom math functions
	functions := map[string]govaluate.ExpressionFunction{
		"sqrt": func(args ...any) (any, error) { return math.Sqrt(toFloat64(args[0])), nil },
		"cbrt": func(args ...any) (any, error) { return math.Cbrt(toFloat64(args[0])), nil },
		"pow": func(args ...any) (any, error) {
			return math.Pow(toFloat64(args[0]), toFloat64(args[1])), nil
		},
		"log":   func(args ...any) (any, error) { return math.Log10(toFloat64(args[0])), nil },
		"ln":    func(args ...any) (any, error) { return math.Log(toFloat64(args[0])), nil },
		"sin":   func(args ...any) (any, error) { return math.Sin(toFloat64(args[0])), nil },
		"cos":   func(args ...any) (any, error) { return math.Cos(toFloat64(args[0])), nil },
		"tan":   func(args ...any) (any, error) { return math.Tan(toFloat64(args[0])), nil },
		"sinh":  func(args ...any) (any, error) { return math.Sinh(toFloat64(args[0])), nil },
		"cosh":  func(args ...any) (any, error) { return math.Cosh(toFloat64(args[0])), nil },
		"tanh":  func(args ...any) (any, error) { return math.Tanh(toFloat64(args[0])), nil },
		"abs":   func(args ...any) (any, error) { return math.Abs(toFloat64(args[0])), nil },
		"ceil":  func(args ...any) (any, error) { return math.Ceil(toFloat64(args[0])), nil },
		"floor": func(args ...any) (any, error) { return math.Floor(toFloat64(args[0])), nil },
		"round": func(args ...any) (any, error) { return math.Round(toFloat64(args[0])), nil },
		"factorial": func(args ...any) (any, error) {
			n := int(toFloat64(args[0]))
			if n < 0 {
				return nil, fmt.Errorf("factorial of negative number")
			}
			return float64(factorial(n)), nil
		},
		"min": func(args ...any) (any, error) {
			a, b := toFloat64(args[0]), toFloat64(args[1])
			if a < b {
				return a, nil
			}
			return b, nil
		},
		"max": func(args ...any) (any, error) {
			a, b := toFloat64(args[0]), toFloat64(args[1])
			if a > b {
				return a, nil
			}
			return b, nil
		},
	}

	// Replace constants
	expr = strings.ReplaceAll(expr, "pi", fmt.Sprintf("%v", math.Pi))
	expr = strings.ReplaceAll(expr, "e", fmt.Sprintf("%v", math.E))

	expression, err := govaluate.NewEvaluableExpressionWithFunctions(expr, functions)
	if err != nil {
		return []Result{
			{
				Name:    "Error parsing expression: " + err.Error(),
				GUI:     false,
				Type:    "calc",
				Source:  "internal",
				Comment: expr,
				Command: "",
			},
		}
	}

	result, err := expression.Evaluate(nil)
	if err != nil {
		return []Result{
			{
				Name:    "Error evaluating expression: " + err.Error(),
				GUI:     false,
				Type:    "calc",
				Source:  "internal",
				Comment: expr,
				Command: "",
			},
		}
	}

	return []Result{
		{
			Name:    fmt.Sprintf("%v", result),
			GUI:     false,
			Type:    "calc",
			Source:  "internal",
			Command: "wl-copy " + fmt.Sprintf("%v", result),
			Comment: fmt.Sprintf("%v", expr),
		},
	}
}

func toFloat64(arg any) float64 {
	switch v := arg.(type) {
	case float64:
		return v
	case int:
		return float64(v)
	case int64:
		return float64(v)
	default:
		return 0
	}
}

func factorial(n int) int {
	if n == 0 {
		return 1
	}
	res := 1
	for i := 2; i <= n; i++ {
		res *= i
	}
	return res
}
