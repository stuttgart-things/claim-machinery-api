package api

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"math/rand/v2"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gorilla/mux"
	"github.com/stuttgart-things/claim-machinery-api/internal/claimtemplate"
	"github.com/stuttgart-things/claim-machinery-api/internal/version"
)

//go:embed ui/templates/*.html ui/static/*
var uiFiles embed.FS

// uiStatic serves ui/static/ as static/ (htmx, stylesheet).
var uiStatic = func() fs.FS {
	sub, err := fs.Sub(uiFiles, "ui")
	if err != nil {
		panic(err)
	}
	return sub
}()

var uiTemplates = template.Must(template.New("").Funcs(template.FuncMap{
	"join": strings.Join,
}).ParseFS(uiFiles, "ui/templates/*.html"))

// randomValue is the form value of the "Random" option of an enum with
// allowRandom; the server picks one of the enum values for it, like the TUI.
const randomValue = "__random__"

// uiPage is what every UI page gets: the prefix for its URLs and the footer.
type uiPage struct {
	Title    string
	Base     string
	Version  string
	LoadedAt string
	Count    int
}

type uiIndexData struct {
	uiPage
	Templates []*claimtemplate.ClaimTemplate
	Tags      []string
}

type uiTemplateData struct {
	uiPage
	Template *claimtemplate.ClaimTemplate
	Fields   []uiField
	Hidden   int
}

// uiField is one form input, derived from a template parameter.
type uiField struct {
	claimtemplate.Parameter
	// Widget is "select", "multiselect", "boolean" or "text".
	Widget string
	// Value is the default as the form shows it; Selected marks the
	// defaults of a multiselect.
	Value    string
	Selected map[string]bool
	// FromFunction names the profile function that fills an empty field.
	FromFunction string
}

type uiResultData struct {
	Base     string
	Name     string
	Rendered string
	Error    string
}

func (s *Server) page(title string) uiPage {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return uiPage{
		Title:    title,
		Base:     s.basePath,
		Version:  version.Version,
		LoadedAt: s.loadedAt.Format(time.RFC3339),
		Count:    len(s.templates),
	}
}

func (s *Server) renderUI(w http.ResponseWriter, name string, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := uiTemplates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("[ui] rendering %s: %v", name, err)
	}
}

// uiIndex lists the templates, filterable by text and tag in the browser.
func (s *Server) uiIndex(w http.ResponseWriter, r *http.Request) {
	data := uiIndexData{uiPage: s.page("Templates")}
	tags := map[string]bool{}
	for _, t := range s.getTemplateMap() {
		data.Templates = append(data.Templates, t)
		for _, tag := range t.Metadata.Tags {
			tags[tag] = true
		}
	}
	sort.Slice(data.Templates, func(i, j int) bool {
		return data.Templates[i].Metadata.Name < data.Templates[j].Metadata.Name
	})
	for tag := range tags {
		data.Tags = append(data.Tags, tag)
	}
	sort.Strings(data.Tags)
	s.renderUI(w, "index.html", data)
}

// uiTemplate shows the order form of one template, generated from its
// parameters. Hidden parameters keep their defaults and are not shown.
func (s *Server) uiTemplate(w http.ResponseWriter, r *http.Request) {
	tmpl, ok := s.getTemplateMap()[mux.Vars(r)["name"]]
	if !ok {
		http.NotFound(w, r)
		return
	}
	data := uiTemplateData{uiPage: s.page(tmpl.Metadata.Name), Template: tmpl}
	for _, p := range tmpl.Spec.Parameters {
		if p.Hidden {
			data.Hidden++
			continue
		}
		data.Fields = append(data.Fields, newUIField(p))
	}
	s.renderUI(w, "template.html", data)
}

func newUIField(p claimtemplate.Parameter) uiField {
	f := uiField{Parameter: p, Widget: "text", Selected: map[string]bool{}}
	if p.ValueFrom != nil {
		f.FromFunction = p.ValueFrom.Function
	}
	switch {
	case p.Multiselect && len(p.Enum) > 0:
		f.Widget = "multiselect"
		if items, ok := p.Default.([]interface{}); ok {
			for _, it := range items {
				f.Selected[fmt.Sprint(it)] = true
			}
		}
		return f
	case len(p.Enum) > 0:
		f.Widget = "select"
	case p.Type == "boolean":
		f.Widget = "boolean"
	}
	switch d := p.Default.(type) {
	case nil:
	case []interface{}:
		parts := make([]string, len(d))
		for i, v := range d {
			parts[i] = fmt.Sprint(v)
		}
		f.Value = strings.Join(parts, ",")
	default:
		f.Value = fmt.Sprint(d)
	}
	return f
}

// uiRender renders the template with the submitted form through the same
// path as POST /api/v1/claim-templates/{name}/order. It answers with an HTML
// fragment for htmx, the error included, so the form stays as it is.
func (s *Server) uiRender(w http.ResponseWriter, r *http.Request) {
	tmpl, ok := s.getTemplateMap()[mux.Vars(r)["name"]]
	if !ok {
		http.NotFound(w, r)
		return
	}
	result := uiResultData{Base: s.basePath, Name: tmpl.Metadata.Name}
	if err := r.ParseForm(); err != nil {
		result.Error = "invalid form: " + err.Error()
		s.renderUI(w, "result.html", result)
		return
	}
	params, err := formParams(tmpl, r.PostForm)
	if err == nil {
		result.Rendered, err = s.order(tmpl, params, strings.TrimSpace(r.PostForm.Get("_author")))
	}
	if err != nil {
		result.Error = err.Error()
	}
	s.renderUI(w, "result.html", result)
}

// formParams turns the submitted form into order parameters with the TUI's
// rules: values stay strings (the KCL schemas expect strings), a multiselect
// is a list, an empty field is left out so the default or the profile
// function fills it, and "Random" picks one of the enum values. Hidden
// parameters are never taken from the form.
func formParams(tmpl *claimtemplate.ClaimTemplate, form map[string][]string) (map[string]interface{}, error) {
	params := map[string]interface{}{}
	var problems []string
	for _, p := range tmpl.Spec.Parameters {
		if p.Hidden {
			continue
		}
		values := form[p.Name]
		if p.Multiselect && len(p.Enum) > 0 {
			var picked []string
			for _, v := range values {
				if contains(p.Enum, v) {
					picked = append(picked, v)
				}
			}
			if len(picked) > 0 {
				params[p.Name] = picked
			}
			continue
		}
		v := ""
		if len(values) > 0 {
			v = strings.TrimSpace(values[0])
		}
		if v == "" {
			if p.Required && p.Default == nil && p.ValueFrom == nil {
				problems = append(problems, fmt.Sprintf("%s is required", label(p)))
			}
			continue
		}
		if v == randomValue && p.AllowRandom && len(p.Enum) > 0 {
			v = p.Enum[rand.IntN(len(p.Enum))]
		}
		if msg := validate(p, v); msg != "" {
			problems = append(problems, msg)
			continue
		}
		params[p.Name] = v
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	return params, nil
}

// validate checks a value against the parameter's enum, pattern and length;
// the browser checks the same, this keeps a hand-made request honest.
func validate(p claimtemplate.Parameter, v string) string {
	if len(p.Enum) > 0 && !contains(p.Enum, v) {
		return fmt.Sprintf("%s must be one of %s", label(p), strings.Join(p.Enum, ", "))
	}
	if p.Type == "boolean" && v != "true" && v != "false" {
		return fmt.Sprintf("%s must be true or false", label(p))
	}
	n := utf8.RuneCountInString(v)
	if p.MinLength != nil && n < *p.MinLength {
		return fmt.Sprintf("%s needs at least %d characters", label(p), *p.MinLength)
	}
	if p.MaxLength != nil && n > *p.MaxLength {
		return fmt.Sprintf("%s allows at most %d characters", label(p), *p.MaxLength)
	}
	if p.Pattern != "" {
		re, err := regexp.Compile(p.Pattern)
		if err == nil && !re.MatchString(v) {
			return fmt.Sprintf("%s does not match %s", label(p), p.Pattern)
		}
	}
	return ""
}

func label(p claimtemplate.Parameter) string {
	if p.Title != "" {
		return p.Title
	}
	return p.Name
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
