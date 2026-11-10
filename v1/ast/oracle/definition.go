// Copyright 2025 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.
package oracle

import (
	"maps"
	"slices"

	"github.com/open-policy-agent/opa/v1/ast"
)

// findDefinition searches a stack of containing nodes for a definition of the
// given target.
// It handles both variables and references and returns nil if no definition is
// found.
func findDefinition(
	t *target,
	stack []ast.Node,
	compiler *ast.Compiler,
	parsed *ast.Module,
) *ast.Location {
	// we only know how to look for vars and refs.
	if !t.isRef && !t.isVar {
		return nil
	}

	matchFn := createMatcher(t)

	// Walk down the stack from innermost to outermost, collecting matches.
	// The matchResult tracks the best match found (ref or variable).
	result := &matchResult{}
	for _, s := range slices.Backward(stack) {
		matchFn(s, compiler, parsed, result)
		// exit if we found a ref (refs always win)
		if !result.isVar && result.loc != nil {
			break
		}
	}

	return result.loc
}

// importAt returns the import whose path contains pos.
func importAt(module *ast.Module, pos int) *ast.Import {
	if module == nil {
		return nil
	}

	for _, imp := range module.Imports {
		if imp.Path == nil || imp.Path.Location == nil {
			continue
		}
		loc := imp.Path.Location
		if pos >= loc.Offset && pos < loc.Offset+len(loc.Text) {
			return imp
		}
	}

	return nil
}

// findImportDefinition returns the package declaration of the package an import
// refers to. If the import refers to a rule instead, the rule is returned.
// When a package is split across modules, the module that sorts first by name
// is used, so the result is stable.
func findImportDefinition(imp *ast.Import, compiler *ast.Compiler) *ast.Location {
	path, ok := imp.Path.Value.(ast.Ref)
	if !ok {
		return nil
	}

	for _, name := range slices.Sorted(maps.Keys(compiler.Modules)) {
		if pkg := compiler.Modules[name].Package; pkg != nil && pkg.Path.Equal(path) {
			return pkg.Location
		}
	}

	if rules := compiler.GetRulesExact(path); len(rules) > 0 {
		return rules[0].Location
	}

	return nil
}
