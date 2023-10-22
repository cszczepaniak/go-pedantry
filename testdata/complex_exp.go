package testdata

type packageMimic struct{}

func (packageMimic) NewBuilder()

var somepackage = packageMimic{}

type Builder struct{}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) A() *Builder {
	return b
}

func (b *Builder) B(x, y int) *Builder {
	return b
}

func (b *Builder) C() *Builder {
	return b
}

func (b *Builder) LotsOfArgs(
	string,
	string,
	string,
	string,
	string,
	string,
	string,
	string,
	string,
) *Builder {
	return b
}

func something() {
	b := somepackage.NewBuilder().
		A().
		B(3, 4).
		LotsOfArgs(
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
		).
		C().
		A()

	b = NewBuilder().
		A().
		B(3, 4).
		LotsOfArgs(
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
			`abcdef`,
		).
		C().
		A()
}
