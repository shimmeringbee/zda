package zda

import (
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/golog"
	"log"
)

func (z *ZDA) WithGoLogger(parentLogger *log.Logger) {
	z.WithLogWrapLogger(logwrap.New(golog.Wrap(parentLogger)))
}

func (z *ZDA) WithLogWrapLogger(lw logwrap.Logger) {
	z.logger = lw
}
