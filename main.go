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

	"github.com/cszczepaniak/go-pedantry/config"
	"github.com/cszczepaniak/go-pedantry/patch"
	"golang.org/x/tools/go/ast/astutil"
)

type nopWriteCloser struct {
	w io.Writer
}

func (n nopWriteCloser) Write(p []byte) (int, error) {
	return n.w.Write(p)
}

func (n nopWriteCloser) Close() error {
	return nil
}

func nopCloser(w io.Writer) io.WriteCloser {
	return nopWriteCloser{
		w: w,
	}
}

func main() {
	var cfg config.Config

	flag.BoolVar(&cfg.Write, `w`, false, `If true, rewrite files. Otherwise, print to stdout.`)
	flag.BoolVar(&cfg.List, `l`, false, `If true, list the files that would be changed to stdout. If -w is not set, this overrides printing the contents of changed files to stdout.`)
	flag.StringVar(&cfg.Input, `input`, ``, `The input file or directory to consider.`)
	flag.StringVar(&cfg.Patch, `patch`, ``, `A git patch file to consider. If both input and patch are set, the program panics. Use "-" for stdin.`)
	flag.Parse()

	err := run(cfg, os.Stdout)
	if err != nil {
		panic(err)
	}
}

func run(cfg config.Config, stdout io.Writer) error {
	if cfg.Patch != `` && cfg.Input != `` {
		return errors.New(`patch and input cannot both be provided`)
	}

	getWriter := func(filename string) (io.WriteCloser, error) {
		if cfg.Write {
			return os.OpenFile(filename, os.O_TRUNC|os.O_WRONLY, os.ModePerm)
		} else if cfg.List {
			// If we're listing, don't write formatted files to stdout.
			return nopCloser(io.Discard), nil
		}
		return nopCloser(stdout), nil
	}

	listSink := io.Discard
	if cfg.List {
		listSink = stdout
	}

	if cfg.Patch != `` {
		return handlePatch(cfg.Patch, getWriter, listSink)
	}

	st, err := os.Stat(cfg.Input)
	if err != nil {
		panic(err)
	}

	if st.IsDir() {
		return filepath.WalkDir(cfg.Input, func(path string, d fs.DirEntry, errr error) error {
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, `.go`) {
				return nil
			}

			formatted, err := formatFile(path, allNodes, listSink)
			if err != nil {
				return err
			}

			return writeFile(path, formatted, getWriter)
		})
	} else {
		formatted, err := formatFile(cfg.Input, allNodes, listSink)
		if err != nil {
			return err
		}

		return writeFile(cfg.Input, formatted, getWriter)
	}
}

func handlePatch(patchArg string, getWriter func(string) (io.WriteCloser, error), listSink io.Writer) error {
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
		formatted, err := formatFile(
			file,
			func(fset *token.FileSet) nodeFilter {
				return patchNodeFilter{
					p:    parsed,
					fset: fset,
				}
			},
			listSink,
		)
		if err != nil {
			return err
		}

		err = writeFile(file, formatted, getWriter)
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

func allNodes(*token.FileSet) nodeFilter {
	return allNodesFilter{}
}

type patchNodeFilter struct {
	p    *patch.Patch
	fset *token.FileSet
}

func (pnf patchNodeFilter) formatNode(n ast.Node) bool {
	pos := pnf.fset.Position(n.Pos())
	return pnf.p.IsLineTouched(pos.Filename, pos.Line)
}

func writeFile(file, s string, getWriter func(string) (io.WriteCloser, error)) error {
	w, err := getWriter(file)
	if err != nil {
		return err
	}

	defer w.Close()
	_, err = io.WriteString(w, s)
	return err
}

func formatFile(
	filename string,
	getNodeFilter func(fset *token.FileSet) nodeFilter,
	listSink io.Writer,
) (_ string, err error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filename, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return ``, err
	}

	originalFset := token.NewFileSet()
	_, err = parser.ParseFile(originalFset, filename, nil, parser.ParseComments|parser.SkipObjectResolution)
	if err != nil {
		return ``, err
	}

	nf := getNodeFilter(originalFset)

	wroteFile := false

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
				wroteFile = putFunctionCallArgsOnSeparateLines(tf, tn, &fsetLiner{
					fset: originalFset,
				}) || wroteFile

				sel, ok := tn.Fun.(*ast.SelectorExpr)
				if ok {
					_, childIsCall := sel.X.(*ast.CallExpr)
					if childIsCall {
						addNewline(tf, sel.Sel.NamePos)
					}
					return true
				}
			case *ast.FuncDecl:
				wroteFile = putFunctionDeclArgsOnSeparateLines(tf, tn, &fsetLiner{
					fset: originalFset,
				}) || wroteFile
			default:
				return true
			}

			return true
		}, nil,
	)

	newFileContents := &strings.Builder{}
	err = format.Node(newFileContents, fset, newF)
	if err != nil {
		return ``, err
	}

	if wroteFile {
		fmt.Fprintln(listSink, filename)
	}

	return newFileContents.String(), nil
}

type liner interface {
	line(pos token.Pos) int
}

type fsetLiner struct {
	fset *token.FileSet
}

func (l *fsetLiner) line(pos token.Pos) int {
	tf := l.fset.File(pos)
	return tf.Line(pos)
}

func putFunctionCallArgsOnSeparateLines(f *token.File, call *ast.CallExpr, l liner) bool {
	elems := call.Args

	if sourceLengthOfList(elems) <= 50 {
		return false
	}

	ret := false
	prevLn := l.line(call.Lparen)
	for i := 0; i < len(elems); i++ {
		el := elems[i]

		if l := l.line(el.Pos()); l > prevLn {
			prevLn = l
			continue
		}

		if i == len(elems)-1 {
			addNewline(f, el.End())
			ret = true
		}
		addNewline(f, el.Pos())
		ret = true
	}
	return ret
}

func putFunctionDeclArgsOnSeparateLines(f *token.File, decl *ast.FuncDecl, l liner) bool {
	params := decl.Type.Params.List

	if sourceLengthOfList(params) <= 50 {
		return false
	}

	ret := false
	prevLn := l.line(decl.Type.Params.Opening)
	for i := 0; i < len(params); i++ {
		el := params[i]

		if l := l.line(el.Pos()); l > prevLn {
			prevLn = l
			continue
		}

		addNewline(f, el.Pos())
		ret = true
	}

	if l.line(decl.Type.Params.Closing) == prevLn {
		decl.Type.Params.Closing += 1
		addNewline(f, decl.Type.Params.Closing)
		ret = true
	}

	return ret
}

func sourceLengthOfList[T ast.Node](items []T) int {
	if len(items) == 0 {
		return 0
	}
	return int(items[len(items)-1].End() - items[0].Pos())
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
