package testdata

type packageMimic struct{}

func (packageMimic) NewBuilder()

var somepackage = packageMimic{}

type Builder struct{}

func NewBuilder() *Builder {
	return &Builder{}
}

func (b *Builder) AIsForAardvark() *Builder {
	return b
}

func (b *Builder) BIsForBatman(x, y int) *Builder {
	return b
}

func (b *Builder) CIsForCatwoman() *Builder {
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
		AIsForAardvark().
		BIsForBatman(3, 4).
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
		CIsForCatwoman().
		AIsForAardvark()

	b = NewBuilder().
		AIsForAardvark().
		BIsForBatman(3, 4).
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
		CIsForCatwoman().
		AIsForAardvark()
}
