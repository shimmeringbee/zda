package zda

import (
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/logwrap/impl/golog"
	"log"
)

func (g *gateway) WithGoLogger(parentLogger *log.Logger) {
	g.WithLogWrapLogger(logwrap.New(golog.Wrap(parentLogger)))
}

func (g *gateway) WithLogWrapLogger(lw logwrap.Logger) {
	g.logger = lw
}
