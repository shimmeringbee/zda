package zda

import (
	"context"
	"math/rand"
	"time"
)

const pollerBacklog = 200
const pollerWorkers = 4
const workerMaximumJobDuration = 15 * time.Second

type zdaPoller struct {
	nodeStore nodeStore

	pollerWork chan pollerWork
	pollerStop chan bool

	rand *rand.Rand
}

type pollerWork struct {
	node     *internalNode
	interval time.Duration
	fn       func(context.Context, *internalNode)
}

func (p *zdaPoller) Start() {
	p.pollerStop = make(chan bool, pollerWorkers)
	p.pollerWork = make(chan pollerWork, pollerBacklog)

	for i := 0; i < pollerWorkers; i++ {
		go p.worker()
	}

	p.rand = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func (p *zdaPoller) Stop() {
	for i := 0; i < pollerWorkers; i++ {
		p.pollerStop <- true
	}
}

func (p *zdaPoller) AddNode(node *internalNode, interval time.Duration, fn func(context.Context, *internalNode)) {
	initialWait := time.Duration(float64(interval) * p.rand.Float64())

	time.AfterFunc(initialWait, func() {
		p.pollerWork <- pollerWork{
			node:     node,
			interval: interval,
			fn:       fn,
		}
	})
}

func (p *zdaPoller) worker() {
	for {
		select {
		case work := <-p.pollerWork:
			_, found := p.nodeStore.getNode(work.node.ieeeAddress)

			if found {
				ctx, cancel := context.WithTimeout(context.Background(), workerMaximumJobDuration)
				work.fn(ctx, work.node)

				time.AfterFunc(work.interval, func() {
					p.pollerWork <- work
				})

				cancel()
			}
		case <-p.pollerStop:
			return
		}
	}
}
