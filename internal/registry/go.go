package registry

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strings"
)

type GoParser struct{}

func (g *GoParser) GetExtensions() []string {
	return []string{".go"}
}

func (g *GoParser) IsHeaderFile(filePath string) bool {
	return false
}

func (g *GoParser) ParseFile(filePath string) ([]Function, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	var functions []Function
	
	// Extract function documentation from comments
	funcDocs := make(map[string]string)
	
	for _, decl := range node.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok && fn.Name != nil {
			if fn.Doc != nil {
				funcDocs[fn.Name.Name] = fn.Doc.Text()
			}
		}
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Name != nil {
				pos := fset.Position(x.Pos())
				end := fset.Position(x.End())
				
				visibility := "private"
				if x.Name.IsExported() {
					visibility = "public"
				}
				
				var params []string
				if x.Type.Params != nil {
					for _, param := range x.Type.Params.List {
						for _, name := range param.Names {
							params = append(params, name.Name)
						}
					}
				}
				
				returnType := parseGoReturnType(x.Type.Results)
				
				isTest := strings.HasPrefix(x.Name.Name, "Test") || 
				         strings.HasPrefix(x.Name.Name, "Benchmark") || 
				         strings.HasPrefix(x.Name.Name, "Example")
				isMain := x.Name.Name == "main"
				
				// Determine if it's a method
				var fullName string
				var receiverType string
				if x.Recv != nil && len(x.Recv.List) > 0 {
					receiverType = extractReceiverType(x.Recv.List[0])
					fullName = receiverType + "." + x.Name.Name
				} else {
					fullName = x.Name.Name
				}
				
				fn := Function{
					Name:       fullName,
					File:       filePath,
					Line:       pos.Line,
					Visibility: visibility,
					ReturnType: returnType,
					Parameters: params,
					Language:   "go",
					Signature:  extractGoSignature(x, fset),
					IsTest:     isTest,
					IsMain:     isMain,
					Size:       end.Line - pos.Line + 1,
					Comments:   funcDocs[x.Name.Name],
					Complexity: calculateGoComplexity(x),
				}
				
				// Add metadata
				fn.Metadata = make(map[string]string)
				if receiverType != "" {
					fn.Metadata["receiver"] = receiverType
					fn.Metadata["method"] = "true"
				}
				if isGenericFunction(x) {
					fn.Metadata["generic"] = "true"
				}
				
				functions = append(functions, fn)
			}
		}
		return true
	})

	return functions, nil
}

func (g *GoParser) FindFunctionCalls(content string) []string {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "", content, 0)
	if err != nil {
		// Fallback to regex if AST parsing fails
		return g.findCallsWithRegex(content)
	}

	var calls []string
	seen := make(map[string]bool)

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			switch fun := x.Fun.(type) {
			case *ast.Ident:
				if !seen[fun.Name] && !isGoBuiltin(fun.Name) {
					calls = append(calls, fun.Name)
					seen[fun.Name] = true
				}
			case *ast.SelectorExpr:
				if sel := fun.Sel; sel != nil {
					if !seen[sel.Name] && !isGoBuiltin(sel.Name) {
						calls = append(calls, sel.Name)
						seen[sel.Name] = true
					}
				}
			}
		}
		return true
	})

	return calls
}

func (g *GoParser) findCallsWithRegex(content string) []string {
	// This is a simplified fallback - the AST method above is preferred
	lines := strings.Split(content, "\n")
	var calls []string
	seen := make(map[string]bool)
	
	for _, line := range lines {
		// Simple regex approach for fallback
		words := strings.Fields(line)
		for i, word := range words {
			if strings.HasSuffix(word, "(") && i > 0 {
				funcName := strings.TrimSuffix(word, "(")
				if !seen[funcName] && !isGoBuiltin(funcName) {
					calls = append(calls, funcName)
					seen[funcName] = true
				}
			}
		}
	}
	
	return calls
}

func parseGoReturnType(results *ast.FieldList) string {
	if results == nil || len(results.List) == 0 {
		return ""
	}
	
	if len(results.List) == 1 {
		return "single"
	}
	
	return "multiple"
}

func extractReceiverType(field *ast.Field) string {
	switch t := field.Type.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	}
	return "unknown"
}

func extractGoSignature(fn *ast.FuncDecl, _ *token.FileSet) string {
	
	// This is a simplified signature extraction
	var sig strings.Builder
	sig.WriteString("func ")
	
	if fn.Recv != nil {
		sig.WriteString("(")
		if len(fn.Recv.List) > 0 {
			sig.WriteString("receiver")
		}
		sig.WriteString(") ")
	}
	
	sig.WriteString(fn.Name.Name)
	sig.WriteString("(")
	
	if fn.Type.Params != nil {
		paramCount := 0
		for _, param := range fn.Type.Params.List {
			if paramCount > 0 {
				sig.WriteString(", ")
			}
			for i, name := range param.Names {
				if i > 0 {
					sig.WriteString(", ")
				}
				sig.WriteString(name.Name)
				paramCount++
			}
		}
	}
	
	sig.WriteString(")")
	
	if fn.Type.Results != nil && len(fn.Type.Results.List) > 0 {
		if len(fn.Type.Results.List) == 1 {
			sig.WriteString(" result")
		} else {
			sig.WriteString(" (results)")
		}
	}
	
	return sig.String()
}

func calculateGoComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // Base complexity
	
	ast.Inspect(fn, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt:
			complexity++
		case *ast.ForStmt:
			complexity++
		case *ast.RangeStmt:
			complexity++
		case *ast.SwitchStmt:
			complexity++
		case *ast.TypeSwitchStmt:
			complexity++
		case *ast.SelectStmt:
			complexity++
		case *ast.CaseClause:
			complexity++
		}
		return true
	})
	
	return complexity
}

func isGenericFunction(fn *ast.FuncDecl) bool {
	if fn.Type.TypeParams != nil && len(fn.Type.TypeParams.List) > 0 {
		return true
	}
	return false
}

func isGoBuiltin(name string) bool {
	builtins := []string{
		"append", "cap", "close", "complex", "copy", "delete", "imag", "len",
		"make", "new", "panic", "print", "println", "real", "recover",
		"bool", "byte", "complex64", "complex128", "error", "float32", "float64",
		"int", "int8", "int16", "int32", "int64", "rune", "string",
		"uint", "uint8", "uint16", "uint32", "uint64", "uintptr",
		"true", "false", "iota", "nil",
	}
	
	for _, builtin := range builtins {
		if name == builtin {
			return true
		}
	}
	
	return false
}