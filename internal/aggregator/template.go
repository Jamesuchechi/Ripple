package aggregator

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// TemplateContext holds variables provided to template rendering.
type TemplateContext struct {
	ActorName  string
	Count      int
	OtherCount int
	Verb       string
	ObjectID   string
	TargetID   string
	Extra      map[string]interface{}
}

// TemplateEngine parses and renders Liquid-style notification templates.
type TemplateEngine struct{}

func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{}
}

var varRegex = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// Render replaces variables in template string using provided context.
func (e *TemplateEngine) Render(tmplStr string, ctx *TemplateContext) string {
	if tmplStr == "" {
		return ""
	}

	result := varRegex.ReplaceAllStringFunc(tmplStr, func(match string) string {
		submatches := varRegex.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}
		varName := strings.TrimSpace(submatches[1])
		switch varName {
		case "actor_name":
			return ctx.ActorName
		case "count":
			return strconv.Itoa(ctx.Count)
		case "other_count":
			return strconv.Itoa(ctx.OtherCount)
		case "verb":
			return ctx.Verb
		case "object_id":
			return ctx.ObjectID
		case "target_id":
			return ctx.TargetID
		default:
			if ctx.Extra != nil {
				if val, ok := ctx.Extra[varName]; ok {
					return fmt.Sprintf("%v", val)
				}
			}
			return match
		}
	})

	return result
}
