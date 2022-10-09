package rules

import (
	"io"
	"io/fs"
)

type Engine struct{}

type Input struct{}
type Output struct{}

func (e Engine) LoadString(_ string) error {
	panic("not yet implemented")
}

func (e Engine) LoadReader(_ io.Reader) error {
	panic("not yet implemented")
}

func (e Engine) LoadFS(_ fs.FS) error {
	panic("not yet implemented")
}

func (e Engine) Execute(_ Input) (Output, error) {
	panic("not yet implemented")
}
