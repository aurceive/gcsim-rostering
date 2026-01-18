package charalias

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"strings"
)

type Resolver struct {
	aliasToCanonical map[string]string
	engineRoot       string
}

func LoadFromEngineRoot(engineRoot string) (*Resolver, error) {
	engineRoot = strings.TrimSpace(engineRoot)
	if engineRoot == "" {
		return nil, errors.New("engineRoot is empty")
	}

	path := filepath.Join(engineRoot, "pkg", "shortcut", "characters.go")
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse engine shortcut file %s: %w", path, err)
	}

	aliases, err := extractCharNameToKeyMap(f)
	if err != nil {
		return nil, fmt.Errorf("extract character aliases from %s: %w", path, err)
	}
	if len(aliases) == 0 {
		return nil, fmt.Errorf("no aliases found in %s", path)
	}
	for _, canon := range aliases {
		if canon == "" {
			continue
		}
		aliases[canon] = canon
	}

	return &Resolver{aliasToCanonical: aliases, engineRoot: engineRoot}, nil
}

func (r *Resolver) Canonicalize(raw string) (string, bool) {
	if r == nil {
		return "", false
	}
	k := strings.ToLower(strings.TrimSpace(raw))
	if k == "" {
		return "", false
	}
	v, ok := r.aliasToCanonical[k]
	return v, ok
}

func (r *Resolver) EngineRoot() string {
	if r == nil {
		return ""
	}
	return r.engineRoot
}

func extractCharNameToKeyMap(f *ast.File) (map[string]string, error) {
	out := map[string]string{}

	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		for _, spec := range gd.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for i, name := range vs.Names {
				if name == nil || name.Name != "CharNameToKey" {
					continue
				}
				if i >= len(vs.Values) {
					return nil, fmt.Errorf("unexpected AST: CharNameToKey has no value")
				}
				lit, ok := vs.Values[i].(*ast.CompositeLit)
				if !ok {
					return nil, fmt.Errorf("unexpected AST: CharNameToKey is not a composite literal")
				}
				for _, elt := range lit.Elts {
					kv, ok := elt.(*ast.KeyValueExpr)
					if !ok {
						continue
					}
					kLit, ok := kv.Key.(*ast.BasicLit)
					if !ok || kLit.Kind != token.STRING {
						continue
					}
					kRaw, err := strconv.Unquote(kLit.Value)
					if err != nil {
						return nil, fmt.Errorf("unquote alias key %q: %w", kLit.Value, err)
					}
					alias := strings.ToLower(strings.TrimSpace(kRaw))
					if alias == "" {
						continue
					}

					canonical, ok := selectorToCanonicalKey(kv.Value)
					if !ok {
						return nil, fmt.Errorf("unsupported alias value expression for %q", alias)
					}

					if existing, exists := out[alias]; exists && existing != canonical {
						return nil, fmt.Errorf("alias %q maps to multiple canonicals: %q and %q", alias, existing, canonical)
					}
					out[alias] = canonical
				}
				return out, nil
			}
		}
	}

	return nil, fmt.Errorf("CharNameToKey not found")
}

func selectorToCanonicalKey(expr ast.Expr) (string, bool) {
	// Expected form: keys.Xingqiu
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok || sel.Sel == nil {
		return "", false
	}
	name := strings.TrimSpace(sel.Sel.Name)
	if name == "" {
		return "", false
	}
	return strings.ToLower(name), true
}
