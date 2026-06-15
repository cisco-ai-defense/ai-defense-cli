package spec

import (
	_ "embed"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed config.spec.yaml
var embeddedSpec []byte

// Schema is the parsed CLI config contract (from config.spec.yaml).
type Schema struct {
	Meta         metaDef          `yaml:"meta"`
	Core         coreDef          `yaml:"core"`
	Integrations []integrationDef `yaml:"integrations"`
}

type metaDef struct {
	SchemaVersion string `yaml:"schema_version"`
	Name          string `yaml:"name"`
	Description   string `yaml:"description"`
}

type coreDef struct {
	ID       string     `yaml:"id"`
	Label    string     `yaml:"label"`
	TomlPath []string   `yaml:"toml_path"`
	Fields   []fieldDef `yaml:"fields"`
}

type fieldOptionDef struct {
	Label string `yaml:"label"`
	Value string `yaml:"value"`
}

type integrationDef struct {
	ID          string      `yaml:"id"`
	Label       string      `yaml:"label"`
	Description string      `yaml:"description"`
	TomlPath    []string    `yaml:"toml_path"`
	Optional    bool        `yaml:"optional"`
	MissingHint string      `yaml:"missing_hint"`
	NextSteps   []string    `yaml:"next_steps"`
	Validator   string      `yaml:"validator"`
	Fields      []fieldDef  `yaml:"fields"`
	Presets     []presetDef `yaml:"presets"`
	Validate    []ruleDef   `yaml:"validate"`
}

type fieldDef struct {
	Key         string           `yaml:"key"`
	Title       string           `yaml:"title"`
	Description string           `yaml:"description"`
	Kind        string           `yaml:"kind"`
	Required    bool             `yaml:"required"`
	Env         string           `yaml:"env"`
	Flag        string           `yaml:"flag"`
	Options     []fieldOptionDef `yaml:"options"`
	AllowCustom bool             `yaml:"allow_custom"`
}

type presetDef struct {
	ID     string            `yaml:"id"`
	Label  string            `yaml:"label"`
	Set    map[string]string `yaml:"set"`
	Prompt []string          `yaml:"prompt"`
	Omit   []string          `yaml:"omit"`
}

type ruleDef struct {
	Type    string `yaml:"type"`
	Field   string `yaml:"field"`
	Message string `yaml:"message"`
}

var (
	// Loaded at init from config.spec.yaml
	schema           Schema
	integrationsByID map[string]IntegrationSpec
	integrationsList []IntegrationSpec
)

func init() {
	if err := loadEmbedded(embeddedSpec); err != nil {
		panic("config spec: " + err.Error())
	}
}

// LoadSchema parses a config.spec.yaml document.
func LoadSchema(raw []byte) (Schema, error) {
	var s Schema
	if err := yaml.Unmarshal(raw, &s); err != nil {
		return Schema{}, fmt.Errorf("parse config.spec.yaml: %w", err)
	}
	if s.Meta.SchemaVersion == "" {
		return Schema{}, fmt.Errorf("config.spec.yaml: meta.schema_version is required")
	}
	return s, nil
}

func loadEmbedded(raw []byte) error {
	s, err := LoadSchema(raw)
	if err != nil {
		return err
	}
	schema = s

	integrationsList, err = buildIntegrations(s.Integrations)
	if err != nil {
		return err
	}
	integrationsByID = make(map[string]IntegrationSpec, len(integrationsList))
	for _, ig := range integrationsList {
		integrationsByID[ig.ID] = ig
	}
	return nil
}

func buildIntegrations(defs []integrationDef) ([]IntegrationSpec, error) {
	out := make([]IntegrationSpec, 0, len(defs))
	for _, d := range defs {
		ig, err := d.toIntegrationSpec()
		if err != nil {
			return nil, fmt.Errorf("integration %q: %w", d.ID, err)
		}
		out = append(out, ig)
	}
	return out, nil
}

func (d integrationDef) toIntegrationSpec() (IntegrationSpec, error) {
	if d.ID == "" {
		return IntegrationSpec{}, fmt.Errorf("id is required")
	}
	if len(d.TomlPath) == 0 {
		return IntegrationSpec{}, fmt.Errorf("toml_path is required")
	}

	fields := make([]FieldSpec, len(d.Fields))
	for i, f := range d.Fields {
		fields[i] = f.toFieldSpec()
	}

	presets := make([]WizardPreset, len(d.Presets))
	for i, p := range d.Presets {
		presets[i] = WizardPreset{
			ID:          p.ID,
			Label:       p.Label,
			SetFields:   p.Set,
			RequireKeys: p.Prompt,
			OmitKeys:    p.Omit,
		}
	}

	validate, err := buildValidator(d)
	if err != nil {
		return IntegrationSpec{}, err
	}

	return IntegrationSpec{
		ID:          d.ID,
		Label:       d.Label,
		Description: d.Description,
		Section: SectionSpec{
			TomlPath: d.TomlPath,
			Fields:   fields,
		},
		Presets:     presets,
		MissingHint: d.MissingHint,
		NextSteps:   d.NextSteps,
		Validate:    validate,
	}, nil
}

func (f fieldDef) toFieldSpec() FieldSpec {
	kind := FieldKindString
	if strings.EqualFold(f.Kind, "secret") {
		kind = FieldKindSecret
	}
	opts := make([]FieldOption, len(f.Options))
	for i, o := range f.Options {
		opts[i] = FieldOption{Label: o.Label, Value: o.Value}
	}
	return FieldSpec{
		Key:         f.Key,
		EnvVar:      f.Env,
		Flag:        f.Flag,
		Title:       f.Title,
		Description: f.Description,
		Kind:        kind,
		Required:    f.Required,
		Options:     opts,
		AllowCustom: f.AllowCustom,
	}
}

func buildValidator(d integrationDef) (func(map[string]string) error, error) {
	if d.Validator != "" {
		fn, ok := namedValidators[d.Validator]
		if !ok {
			return nil, fmt.Errorf("unknown validator %q", d.Validator)
		}
		return fn, nil
	}
	if len(d.Validate) > 0 {
		return declarativeValidator(d.Validate), nil
	}
	return nil, nil
}

func declarativeValidator(rules []ruleDef) func(map[string]string) error {
	return func(values map[string]string) error {
		for _, r := range rules {
			switch r.Type {
			case "non_empty":
				if strings.TrimSpace(values[r.Field]) == "" {
					msg := r.Message
					if msg == "" {
						msg = fmt.Sprintf("%s is required", r.Field)
					}
					return fmt.Errorf("%s", msg)
				}
			default:
				return fmt.Errorf("unknown validate rule type %q", r.Type)
			}
		}
		return nil
	}
}

// Integrations returns optional capability specs (from config.spec.yaml).
func Integrations() []IntegrationSpec {
	return integrationsList
}

// ByID returns an integration spec by id.
func ByID(id string) (IntegrationSpec, bool) {
	ig, ok := integrationsByID[id]
	return ig, ok
}

// MustByID returns an integration spec or panics if the id is absent from config.spec.yaml.
// Use from command packages that bind to a known integration id at compile time.
func MustByID(id string) IntegrationSpec {
	ig, ok := ByID(id)
	if !ok {
		panic("config spec: integration " + id + " not defined in config.spec.yaml")
	}
	return ig
}

// CoreFields returns parsed core field specs (api_key, base_url, …).
func CoreFields() []FieldSpec {
	fields := make([]FieldSpec, len(schema.Core.Fields))
	for i, f := range schema.Core.Fields {
		fields[i] = f.toFieldSpec()
	}
	return fields
}

// CoreSchema returns the parsed core section.
func CoreSchema() coreDef {
	return schema.Core
}

// EmbeddedSpecRaw returns the embedded config.spec.yaml bytes (for docs/tests).
func EmbeddedSpecRaw() []byte {
	return embeddedSpec
}
