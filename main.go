package main

import (
	"errors"
	"flag"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/cszczepaniak/go-pedantry/patch"
	"golang.org/x/tools/go/ast/astutil"
)

type stdoutWriteCloser struct {
	stdout *os.File
}

func newStdoutWriteCloser() *stdoutWriteCloser {
	return &stdoutWriteCloser{
		stdout: os.Stdout,
	}
}

func (s *stdoutWriteCloser) Write(p []byte) (int, error) {
	return s.stdout.Write(p)
}

func (s *stdoutWriteCloser) Close() error {
	return nil
}

func main() {
	var (
		write    bool
		patchArg string
		input    string
	)

	flag.BoolVar(&write, `w`, false, `If true, rewrite files. Otherwise, print to stdout.`)
	flag.StringVar(&input, `input`, ``, `The input file or directory to consider.`)
	flag.StringVar(&patchArg, `patch`, ``, `A git patch file to consider. If both input and patch are set, the program panics. Use "-" for stdin.`)
	flag.Parse()

	if patchArg != `` && input != `` {
		panic(`patch and input cannot both be provided`)
	}

	getWriter := func(filename string) (io.WriteCloser, error) {
		if write {
			return os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		}
		return newStdoutWriteCloser(), nil
	}

	if patchArg != `` {
		err := handlePatch(patchArg, getWriter)
		if err != nil {
			panic(err)
		}
		return
	}

	st, err := os.Stat(input)
	if err != nil {
		panic(err)
	}

	if st.IsDir() {
		err = filepath.WalkDir(input, func(path string, d fs.DirEntry, err error) error {
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, `.go`) {
				return nil
			}

			return writeFile(path, func(*token.FileSet) nodeFilter {
				return allNodesFilter{}
			}, getWriter)
		})
		if err != nil {
			panic(err)
		}
	} else {
		err = writeFile(input, func(*token.FileSet) nodeFilter {
			return allNodesFilter{}
		}, getWriter)
		if err != nil {
			panic(err)
		}
	}
}

func handlePatch(patchArg string, getWriter func(string) (io.WriteCloser, error)) error {
	var p io.Reader
	if patchArg == `-` {
		p = os.Stdin
	} else {
		patchFile, err := os.Open(patchArg)
		if err != nil {
			panic(err)
		}
		defer patchFile.Close()
		p = patchFile
	}

	parsed, err := patch.Parse(p)
	if err != nil {
		return err
	}

	for _, file := range parsed.ChangedFiles() {
		err := writeFile(file, func(fset *token.FileSet) nodeFilter {
			return patchNodeFilter{
				p:    parsed,
				fset: fset,
			}
		}, getWriter)
		if err != nil {
			return err
		}
	}

	return nil
}

type nodeFilter interface {
	formatNode(n ast.Node) bool
}

type allNodesFilter struct{}

func (allNodesFilter) formatNode(ast.Node) bool {
	return true
}

type patchNodeFilter struct {
	p    *patch.Patch
	fset *token.FileSet
}

func (pnf patchNodeFilter) formatNode(n ast.Node) bool {
	pos := pnf.fset.Position(n.Pos())
	return pnf.p.IsLineTouched(pos.Filename, pos.Line)
}

func writeFile(
	filename string,
	getNodeFilter func(fset *token.FileSet) nodeFilter,
	getWriter func(filename string) (io.WriteCloser, error),
) (err error) {
	w, err := getWriter(filename)
	if err != nil {
		return err
	}
	defer func() {
		closeErr := w.Close()
		if closeErr != nil {
			if err != nil {
				err = errors.Join(err, closeErr)
			} else {
				err = closeErr
			}
		}
	}()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return err
	}

	originalFset := token.NewFileSet()
	_, err = parser.ParseFile(originalFset, filename, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return err
	}

	nf := getNodeFilter(originalFset)

	newF := astutil.Apply(f,
		func(c *astutil.Cursor) bool {
			if c.Node() == nil {
				return true
			}

			if !nf.formatNode(c.Node()) {
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

	format.Node(w, fset, newF)
	return nil
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
