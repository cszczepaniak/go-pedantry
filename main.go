package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"

	
	"os"
	"strings"

	"golang.org/x/tools/go/ast/astutil"
)

func main() {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, ``, []byte(sampleCode2), parser.SpuriousErrors)
	if err != nil {
		panic(err)
	}

	dumpAST(fset, f)

	newF := astutil.Apply(f,
		func(c *astutil.Cursor) bool {
			if c.Node() == nil {
				return true
			}

			tf := fset.File(c.Node().Pos())

			switch tn := c.Node().(type) {
			case *ast.CallExpr:
				putFunctionCallArgsOnSeparateLines(tf, tn)

				sel, ok := tn.Fun.(*ast.SelectorExpr)
				if ok && hasChildCallSelector(c.Node()) {
					chCall, childIsCall := sel.X.(*ast.CallExpr)
					if childIsCall && sourceLengthOfList(chCall.Args) <= 50 {
						addNewline(tf, sel.Sel.NamePos)
					}
					return true
				}
			case *ast.FuncDecl:
				putFunctionDeclArgsOnSeparateLines(tf, tn)
			}

			return true
		}, nil,
	)

	format.Node(os.Stdout, fset, newF)
}

func dumpAST(fset *token.FileSet, f *ast.File) {
	depth := 0
	astutil.Apply(f, func(c *astutil.Cursor) bool {
		if c.Node() == nil {
			return false
		}
		padding := ``
		if depth > 0 {
			padding = strings.Repeat("\t", depth)
		}

		p := fset.Position(c.Node().Pos())

		fmt.Printf("%s%T %#v [%s] {%v:%v}\n", padding, c.Node(), c.Node(), c.Name(), p.Line, p.Offset)
		depth++
		return true
	}, func(c *astutil.Cursor) bool {
		depth--
		return true
	})
}

func putFunctionCallArgsOnSeparateLines(f *token.File, call *ast.CallExpr) {
	elems := call.Args

	if sourceLengthOfList(elems) <= 50 {
		return
	}

	for i := len(elems) - 1; i >= 0; i-- {
		el := elems[i]
		if i == len(elems)-1 {
			addNewline(f, el.End())
		}
		addNewline(f, el.Pos())
	}
}

func putFunctionDeclArgsOnSeparateLines(f *token.File, decl *ast.FuncDecl) {
	params := decl.Type.Params.List

	if sourceLengthOfList(params) <= 50 {
		return
	}

	for i := len(params) - 1; i >= 0; i-- {
		el := params[i]
		addNewline(f, el.Pos())
	}

	decl.Type.Params.Closing += 1
	addNewline(f, decl.Type.Params.Closing)
}

func sourceLengthOfList[T ast.Node](items []T) int {
	if len(items) == 0 {
		return 0
	}
	return int(items[len(items)-1].End() - items[0].Pos())
}

func isCallSelector(n ast.Node) (*ast.CallExpr, *ast.SelectorExpr, bool) {
	call, ok := n.(*ast.CallExpr)
	if !ok {
		return nil, nil, false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return nil, nil, false
	}

	return call, sel, true
}

func hasChildCallSelector(n ast.Node) bool {
	_, sel, ok := isCallSelector(n)
	if !ok {
		return false
	}

	_, _, ok = isCallSelector(sel.X)
	return ok
}

func addNewline(f *token.File, at token.Pos) {
	offset := f.Offset(at)

	insertBefore := -1
	currLines := f.Lines()

	for i, cur := range currLines {
		if offset == cur {
			// This newline already exists; do nothing. Duplicate
			// newlines can't exist.
			return
		}
		if offset < cur {
			insertBefore = i
			break
		}
	}

	lines := make([]int, 0, len(currLines)+1)
	if insertBefore == -1 {
		lines = append(lines, currLines...)
		lines = append(lines, offset)
	} else {
		lines = append(lines, currLines[:insertBefore]...)
		lines = append(lines, offset)
		lines = append(lines, currLines[insertBefore:]...)
	}

	if !f.SetLines(lines) {
		panic(fmt.Sprintf("could not set lines to %v", lines))
	}
}

const sampleCode = `package something

func a(aaa int, bbb bool, ccc bool, ddd string, eee int, fff bool, ggg bool, hhh int) {}

func something() {
	another.DoSomething(11111, 111, 111, 11, 11, 111111, 1111, 1111, 1111, 1111, 2).A(true).B("hey").C(abc)
}`

const sampleCode2 = `package something

func a(aaa int, bbb bool, ccc bool, ddd string, eee int, fff bool, ggg bool, hhh int) {}
`

const desiredCode = `package something

func something() {
	another.
		DoSomething(1, 2).
		A(true)
}`
