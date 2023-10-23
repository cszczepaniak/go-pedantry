package patch

import (
	"io"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

type Patch struct {
	fileToPatch map[string]*FilePatch
}

func newPatch() *Patch {
	return &Patch{
		fileToPatch: make(map[string]*FilePatch),
	}
}

func (p *Patch) newForFile(file string) *FilePatch {
	fp := newFilePatch()
	p.fileToPatch[file] = fp
	return fp
}

func (p *Patch) ChangedFiles() []string {
	res := make([]string, 0, len(p.fileToPatch))
	for f := range p.fileToPatch {
		res = append(res, f)
	}
	return res
}

func (p *Patch) IsLineTouched(filename string, ln int) bool {
	fp, ok := p.fileToPatch[filename]
	if !ok {
		return false
	}
	return fp.isLineTouched(ln)
}

type FilePatch struct {
	touchedLines map[int]struct{}
}

func newFilePatch() *FilePatch {
	return &FilePatch{
		touchedLines: make(map[int]struct{}),
	}
}

func (fp *FilePatch) insertTouchedLine(l int) {
	fp.touchedLines[l] = struct{}{}
}

func (fp *FilePatch) isLineTouched(l int) bool {
	_, ok := fp.touchedLines[l]
	return ok

}

func Parse(r io.Reader) (*Patch, error) {
	files, _, err := gitdiff.Parse(r)
	if err != nil {
		return nil, err
	}

	p := newPatch()

	for _, f := range files {
		fp := p.newForFile(f.NewName)

		for _, frag := range f.TextFragments {
			lineInNewFile := frag.NewPosition - 1
			for _, l := range frag.Lines {
				if l.Op != gitdiff.OpDelete {
					lineInNewFile++
				}
				if l.Op == gitdiff.OpAdd {
					fp.insertTouchedLine(int(lineInNewFile))
				}
			}
		}
	}

	return p, nil
}
