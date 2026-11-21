package standards

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

type Variant struct {
	Name        string
	TalentLevel *int
	Options     string
}

type Standards struct {
	variants []Variant
	err      error
}

type fileConfig struct {
	Variants []struct {
		Name    string         `yaml:"name"`
		Options map[string]any `yaml:"options"`
	} `yaml:"optimizer_variants"`
}

var optionName = regexp.MustCompile(`^[a-z_]+$`)
var decimal = regexp.MustCompile(`^[0-9]+(?:\.[0-9]+)?$`)

func Load(path string) *Standards {
	raw, err := os.ReadFile(path)
	if err != nil {
		return &Standards{err: fmt.Errorf("read balancer settings %s: %w", path, err)}
	}
	var cfg fileConfig
	decoder := yaml.NewDecoder(bytes.NewReader(raw))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return &Standards{err: fmt.Errorf("parse balancer settings %s: %w", path, err)}
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			err = errors.New("multiple YAML documents are not supported")
		}
		return &Standards{err: fmt.Errorf("parse balancer settings %s: %w", path, err)}
	}
	if len(cfg.Variants) == 0 {
		return &Standards{variants: []Variant{{Name: "kqms"}}}
	}
	result := &Standards{}
	seen := make(map[string]bool)
	for _, entry := range cfg.Variants {
		name := strings.TrimSpace(entry.Name)
		if name == "" || seen[name] {
			return &Standards{err: fmt.Errorf("balancer settings %s: empty or duplicate variant name %q", path, name)}
		}
		seen[name] = true
		variant := Variant{Name: name}
		keys := make([]string, 0, len(entry.Options))
		for key := range entry.Options {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		var options []string
		for _, key := range keys {
			if !optionName.MatchString(key) {
				return &Standards{err: fmt.Errorf("variant %q: invalid optimizer option name %q", name, key)}
			}
			value, err := number(entry.Options[key])
			if err != nil {
				return &Standards{err: fmt.Errorf("variant %q option %q: %w", name, key, err)}
			}
			if key == "talent_level" {
				level, err := strconv.Atoi(value)
				if err != nil || level < 1 || level > 10 {
					return &Standards{err: fmt.Errorf("variant %q: talent_level must be an integer in [1..10]", name)}
				}
				variant.TalentLevel = &level
				continue
			}
			switch key {
			case "total_liquid_substats", "indiv_liquid_cap", "fixed_substats_count", "fine_tune":
				if _, err := strconv.Atoi(value); err != nil {
					return &Standards{err: fmt.Errorf("variant %q option %q must be a non-negative integer", name, key)}
				}
			case "show_substat_scalars", "verbose":
				if value != "0" && value != "1" {
					return &Standards{err: fmt.Errorf("variant %q option %q must be 0 or 1", name, key)}
				}
			}
			options = append(options, key+"="+value)
		}
		variant.Options = strings.Join(options, ";")
		result.variants = append(result.variants, variant)
	}
	return result
}

func number(raw any) (string, error) {
	var value string
	switch n := raw.(type) {
	case int:
		value = strconv.Itoa(n)
	case int64:
		value = strconv.FormatInt(n, 10)
	case uint64:
		value = strconv.FormatUint(n, 10)
	case float64:
		if math.IsInf(n, 0) || math.IsNaN(n) {
			return "", errors.New("must be a finite non-negative number")
		}
		value = strconv.FormatFloat(n, 'f', -1, 64)
	case string:
		value = strings.TrimSpace(n)
	default:
		return "", errors.New("must be a non-negative number")
	}
	if !decimal.MatchString(value) {
		return "", fmt.Errorf("invalid number %q", value)
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsInf(n, 0) {
		return "", fmt.Errorf("invalid number %q", value)
	}
	return value, nil
}

func (s *Standards) Names() ([]string, error) {
	if s.err != nil {
		return nil, s.err
	}
	names := make([]string, len(s.variants))
	for i, variant := range s.variants {
		names[i] = variant.Name
	}
	return names, nil
}

func (s *Standards) Find(name string) (Variant, error) {
	if s.err != nil {
		return Variant{}, s.err
	}
	for _, variant := range s.variants {
		if variant.Name == name {
			return variant, nil
		}
	}
	return Variant{}, fmt.Errorf("unknown balancer standard %q", name)
}
