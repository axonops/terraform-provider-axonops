package main

// schema_quality_test.go checks every provider, resource and data source
// attribute for a useful description and for correct Sensitive marking.
// Any finding fails the test, so a new attribute without a proper description
// or Sensitive flag cannot be merged.

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	pschema "github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	rschema "github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
)

// attrInfo is the subset of an attribute the quality checks need. All
// framework attribute types, for resources, data sources and the provider,
// satisfy it.
type attrInfo interface {
	GetDescription() string
	GetMarkdownDescription() string
	IsSensitive() bool
	IsRequired() bool
	IsOptional() bool
	IsComputed() bool
}

type attrNode struct {
	path string
	attr attrInfo
}

type finding struct {
	entity  string
	path    string
	problem string
}

func (f finding) String() string {
	return fmt.Sprintf("%s: %s: %s", f.entity, f.path, f.problem)
}

var (
	replaceWordsRe   = regexp.MustCompile(`(?i)replac|recreat|destroy|forces? (a )?new|new resource`)
	credentialNameRe = regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|api_?key|private_?key|_key$|^key$|webhook|credential|routing_?key|(^|_)auth($|_token|_key|_secret))`)
)

// notCredentials lists attributes whose name looks like a credential but is
// not secret. Each entry needs a reason.
var notCredentials = map[string]string{
	"token_type": "selects the authorisation scheme, not a secret",
}

// flattenResourceAttrs returns every attribute, including nested ones, with a
// dotted path.
func flattenResourceAttrs(prefix string, attrs map[string]rschema.Attribute) []attrNode {
	var out []attrNode
	for name, a := range attrs {
		p := join(prefix, name)
		out = append(out, attrNode{p, a})
		switch n := a.(type) {
		case rschema.SingleNestedAttribute:
			out = append(out, flattenResourceAttrs(p, n.Attributes)...)
		case rschema.ListNestedAttribute:
			out = append(out, flattenResourceAttrs(p, n.NestedObject.Attributes)...)
		case rschema.SetNestedAttribute:
			out = append(out, flattenResourceAttrs(p, n.NestedObject.Attributes)...)
		case rschema.MapNestedAttribute:
			out = append(out, flattenResourceAttrs(p, n.NestedObject.Attributes)...)
		}
	}
	return out
}

func flattenDataSourceAttrs(prefix string, attrs map[string]dsschema.Attribute) []attrNode {
	var out []attrNode
	for name, a := range attrs {
		p := join(prefix, name)
		out = append(out, attrNode{p, a})
		switch n := a.(type) {
		case dsschema.SingleNestedAttribute:
			out = append(out, flattenDataSourceAttrs(p, n.Attributes)...)
		case dsschema.ListNestedAttribute:
			out = append(out, flattenDataSourceAttrs(p, n.NestedObject.Attributes)...)
		case dsschema.SetNestedAttribute:
			out = append(out, flattenDataSourceAttrs(p, n.NestedObject.Attributes)...)
		case dsschema.MapNestedAttribute:
			out = append(out, flattenDataSourceAttrs(p, n.NestedObject.Attributes)...)
		}
	}
	return out
}

func flattenProviderAttrs(prefix string, attrs map[string]pschema.Attribute) []attrNode {
	var out []attrNode
	for name, a := range attrs {
		out = append(out, attrNode{join(prefix, name), a})
	}
	return out
}

func join(prefix, name string) string {
	if prefix == "" {
		return name
	}
	return prefix + "." + name
}

// entitySchema is one resource, data source or the provider with its
// flattened attributes. replaceNodes records which attributes carry a
// plan modifier that forces replacement.
type entitySchema struct {
	name         string
	nodes        []attrNode
	forceReplace map[string]bool
}

func collectEntities(t *testing.T) []entitySchema {
	t.Helper()
	ctx := context.Background()
	p := New()()
	var out []entitySchema

	var presp provider.SchemaResponse
	p.Schema(ctx, provider.SchemaRequest{}, &presp)
	out = append(out, entitySchema{
		name:  "provider",
		nodes: flattenProviderAttrs("", presp.Schema.Attributes),
	})

	for _, f := range p.Resources(ctx) {
		r := f()
		var m resource.MetadataResponse
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "axonops"}, &m)
		var s resource.SchemaResponse
		r.Schema(ctx, resource.SchemaRequest{}, &s)
		e := entitySchema{
			name:         "resource " + m.TypeName,
			nodes:        flattenResourceAttrs("", s.Schema.Attributes),
			forceReplace: map[string]bool{},
		}
		for _, n := range e.nodes {
			if forcesReplacement(ctx, n.attr) {
				e.forceReplace[n.path] = true
			}
		}
		out = append(out, e)
	}

	for _, f := range p.DataSources(ctx) {
		d := f()
		var m datasource.MetadataResponse
		d.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "axonops"}, &m)
		var s datasource.SchemaResponse
		d.Schema(ctx, datasource.SchemaRequest{}, &s)
		out = append(out, entitySchema{
			name:  "data source " + m.TypeName,
			nodes: flattenDataSourceAttrs("", s.Schema.Attributes),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].name < out[j].name })
	return out
}

// forcesReplacement inspects an attribute's plan modifiers by description,
// because the framework exposes them through type-specific methods.
func forcesReplacement(ctx context.Context, a any) bool {
	v := reflect.ValueOf(a)
	for i := 0; i < v.NumMethod(); i++ {
		if !strings.HasSuffix(v.Type().Method(i).Name, "PlanModifiers") {
			continue
		}
		res := v.Method(i).Call(nil)
		if len(res) != 1 || res[0].Kind() != reflect.Slice {
			continue
		}
		for j := 0; j < res[0].Len(); j++ {
			d, ok := res[0].Index(j).Interface().(interface{ Description(context.Context) string })
			if ok && strings.Contains(strings.ToLower(d.Description(ctx)), "recreate") {
				return true
			}
		}
	}
	return false
}

func description(a attrInfo) string {
	if d := a.GetMarkdownDescription(); d != "" {
		return d
	}
	return a.GetDescription()
}

var nonAlnumRe = regexp.MustCompile(`[^a-z0-9]+`)

func normalise(s string) string {
	return strings.Trim(nonAlnumRe.ReplaceAllString(strings.ToLower(s), " "), " ")
}

// repeatsName reports whether the description only restates the attribute
// name, optionally with a leading article or "the".
func repeatsName(path, desc string) bool {
	name := normalise(path[strings.LastIndex(path, ".")+1:])
	d := normalise(desc)
	for _, lead := range []string{"the ", "a ", "an "} {
		d = strings.TrimPrefix(d, lead)
	}
	return d == name
}

func checkDescriptions(e entitySchema) []finding {
	var out []finding
	for _, n := range e.nodes {
		desc := strings.TrimSpace(description(n.attr))
		switch {
		case desc == "":
			out = append(out, finding{e.name, n.path, "empty description"})
		case repeatsName(n.path, desc):
			out = append(out, finding{e.name, n.path, "description only repeats the attribute name"})
		case e.forceReplace[n.path] && !replaceWordsRe.MatchString(desc):
			out = append(out, finding{e.name, n.path, "forces replacement but the description does not say so"})
		}
	}
	return out
}

func checkSensitive(e entitySchema) []finding {
	var out []finding
	for _, n := range e.nodes {
		leaf := n.path[strings.LastIndex(n.path, ".")+1:]
		if _, ok := notCredentials[leaf]; ok {
			continue
		}
		if credentialNameRe.MatchString(leaf) && !n.attr.IsSensitive() {
			out = append(out, finding{e.name, n.path, "credential-like name but not Sensitive"})
		}
	}
	return out
}

func report(t *testing.T, findings []finding) {
	t.Helper()
	if len(findings) == 0 {
		return
	}
	var b strings.Builder
	for _, f := range findings {
		b.WriteString(f.String() + "\n")
	}
	t.Errorf("%d findings:\n%s", len(findings), b.String())
}

func TestSchemaDescriptions(t *testing.T) {
	var all []finding
	for _, e := range collectEntities(t) {
		all = append(all, checkDescriptions(e)...)
	}
	report(t, all)
}

func TestSchemaSensitiveCredentials(t *testing.T) {
	var all []finding
	for _, e := range collectEntities(t) {
		all = append(all, checkSensitive(e)...)
	}
	report(t, all)
}

func replaceModifiers() []planmodifier.String {
	return []planmodifier.String{stringplanmodifier.RequiresReplace()}
}

// fixtureEntity builds a deliberately bad resource schema so the checks can
// be shown to detect each failure type.
func fixtureEntity() entitySchema {
	ctx := context.Background()
	attrs := map[string]rschema.Attribute{
		"empty": rschema.StringAttribute{Optional: true},
		"cluster_name": rschema.StringAttribute{
			Required:    true,
			Description: "The cluster name.",
		},
		"silent_replace": rschema.StringAttribute{
			Required:      true,
			Description:   "Name of the cluster the topic belongs to.",
			PlanModifiers: replaceModifiers(),
		},
		"api_key": rschema.StringAttribute{
			Required:    true,
			Description: "Key used to authenticate against the API.",
		},
		"good_replace": rschema.StringAttribute{
			Required:      true,
			Description:   "Topic name. Changing it forces the topic to be replaced.",
			PlanModifiers: replaceModifiers(),
		},
		"safe_key": rschema.StringAttribute{
			Required:    true,
			Sensitive:   true,
			Description: "Signing key for outbound requests.",
		},
	}
	e := entitySchema{
		name:         "resource fixture",
		nodes:        flattenResourceAttrs("", attrs),
		forceReplace: map[string]bool{},
	}
	for _, n := range e.nodes {
		if forcesReplacement(ctx, n.attr) {
			e.forceReplace[n.path] = true
		}
	}
	return e
}

func problemsByPath(fs []finding) map[string]string {
	m := map[string]string{}
	for _, f := range fs {
		m[f.path] = f.problem
	}
	return m
}

func TestSchemaQualityDetectsBadAttributes(t *testing.T) {
	e := fixtureEntity()

	desc := problemsByPath(checkDescriptions(e))
	for path, want := range map[string]string{
		"empty":          "empty description",
		"cluster_name":   "repeats the attribute name",
		"silent_replace": "forces replacement",
	} {
		if got := desc[path]; !strings.Contains(got, want) {
			t.Errorf("description check for %q: got %q, want it to contain %q", path, got, want)
		}
	}
	for _, ok := range []string{"good_replace", "api_key", "safe_key"} {
		if got, bad := desc[ok]; bad {
			t.Errorf("description check flagged %q unexpectedly: %s", ok, got)
		}
	}

	sens := problemsByPath(checkSensitive(e))
	if _, ok := sens["api_key"]; !ok {
		t.Errorf("sensitive check did not flag api_key")
	}
	if _, ok := sens["safe_key"]; ok {
		t.Errorf("sensitive check flagged safe_key although it is Sensitive")
	}
}
