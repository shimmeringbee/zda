package zda

import (
	"context"
	"github.com/shimmeringbee/logwrap"
	"github.com/shimmeringbee/zda/implcaps/factory"
	"github.com/shimmeringbee/zigbee"
)

func (z *ZDA) providerLoad() {
	ctx, end := z.logger.Segment(z.ctx, "Loading persistence.")
	defer end()

	for _, i := range z.nodeListFromPersistence() {
		z.providerLoadNode(ctx, i)
	}
}

func (z *ZDA) providerLoadNode(pctx context.Context, i zigbee.IEEEAddress) {
	ctx, end := z.logger.Segment(pctx, "Loading node data.", logwrap.Datum("node", i.String()))
	defer end()

	n, _ := z.createNode(i)
	for _, d := range z.deviceListFromPersistence(i) {
		z.providerLoadDevice(ctx, n, d)
	}
}

func (z *ZDA) providerLoadDevice(pctx context.Context, n *node, i IEEEAddressWithSubIdentifier) {
	ctx, end := z.logger.Segment(pctx, "Loading device data.", logwrap.Datum("device", i.String()))
	defer end()

	d := z.createSpecificDevice(n, i.SubIdentifier)

	capSection := z.sectionForDevice(i).Section("Device")

	for _, cName := range capSection.SectionKeys() {
		cctx, cend := z.logger.Segment(ctx, "Loading capability data.", logwrap.Datum("capability", cName))

		cSection := capSection.Section(cName)

		if capImpl, ok := cSection.String("Implementation"); ok {
			if capI := factory.Create(capImpl, z.zdaInterface); capI == nil {
				z.logger.LogError(cctx, "Could not find capability implementation.", logwrap.Datum("Implementation", capImpl))
				continue
			} else {
				z.logger.LogInfo(cctx, "Constructed capability implementation.", logwrap.Datum("Implementation", capImpl))
				capI.Init(d, cSection.Section("Data"))
				attached, err := capI.Load(cctx)

				if err != nil {
					z.logger.LogError(cctx, "Error while loading from persistence.", logwrap.Err(err), logwrap.Datum("Implementation", capImpl))
				}

				if attached {
					z.attachCapabilityToDevice(d, capI)
					z.logger.LogInfo(cctx, "Attached capability from persistence.", logwrap.Datum("Implementation", capImpl))
				} else {
					z.logger.LogWarn(cctx, "Rejected capability attach from persistence.", logwrap.Datum("Implementation", capImpl))
				}
			}
		}

		cend()
	}
}
