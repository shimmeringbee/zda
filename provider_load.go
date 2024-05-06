package zda

import (
	"context"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zda/implcaps/factory"
	"github.com/shimmeringbee/zigbee"
)

func (g *gateway) providerLoad() {
	ctx, end := g.logger.Segment(g.ctx, "Loading persistence.")
	defer end()

	for _, i := range g.nodeListFromPersistence() {
		g.providerLoadNode(ctx, i)
	}
}

func (g *gateway) providerLoadNode(pctx context.Context, i zigbee.IEEEAddress) {
	ctx, end := g.logger.Segment(pctx, "Loading node data.", logwrap.Datum("node", i.String()))
	defer end()

	n, _ := g.createNode(i)
	for _, d := range g.deviceListFromPersistence(i) {
		g.providerLoadDevice(ctx, n, d)
	}
}

func (g *gateway) providerLoadDevice(pctx context.Context, n *node, i IEEEAddressWithSubIdentifier) {
	ctx, end := g.logger.Segment(pctx, "Loading device data.", logwrap.Datum("device", i.String()))
	defer end()

	d := g.createSpecificDevice(n, i.SubIdentifier)

	capSection := g.sectionForDevice(i).Section("capability")

	for _, cName := range capSection.Keys() {
		cSection := capSection.Section(cName)

		if capImpl, ok := cSection.String("implementation"); ok {
			if capI := factory.Create(capImpl, g.zdaInterface); capI == nil {
				g.logger.LogError(ctx, "Could not find capability implementation.", logwrap.Datum("implementation", capImpl))
				continue
			} else {
				g.logger.LogInfo(ctx, "Constructed capability implementation.", logwrap.Datum("implementation", capImpl))
				capI.Init(d, cSection.Section("data"))
				attached, err := capI.Load(ctx)

				if err != nil {
					g.logger.LogError(ctx, "Error while loading from persistence.", logwrap.Err(err), logwrap.Datum("implementation", capImpl))
				}

				if attached {
					g.attachCapabilityToDevice(d, capI)
					g.logger.LogInfo(ctx, "Attached capability from persistence.", logwrap.Datum("implementation", capImpl))
				} else {
					g.logger.LogWarn(ctx, "Rejected capability attach from persistence.", logwrap.Datum("implementation", capImpl))
				}
			}
		}
	}
}
