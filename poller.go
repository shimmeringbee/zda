package zda

import (
	"context"
	"math/rand"
	"sync"
	"time"
)

const pollerBacklog = 200
const pollerWorkers = 4
const workerMaximumJobDuration = 15 * time.Second

type zdaPoller struct {
	nodeTable nodeTable

	pollerWork chan pollerWork
	pollerStop chan bool

	randLock *sync.Mutex
	rand     *rand.Rand
}

type pollerWork struct {
	identifier IEEEAddressWithSubIdentifier
	interval   time.Duration
	fn         func(context.Context, *internalDevice) bool
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

func (p *zdaPoller) Add(identifier IEEEAddressWithSubIdentifier, interval time.Duration, fn func(context.Context, *internalDevice) bool) {
	p.randLock.Lock()
	initialWait := time.Duration(float64(interval) * p.rand.Float64())
	p.randLock.Unlock()

	time.AfterFunc(initialWait, func() {
		p.pollerWork <- pollerWork{
			identifier: identifier,
			interval:   interval,
			fn:         fn,
		}
	})
}

func (p *zdaPoller) worker() {
	for {
		select {
		case work := <-p.pollerWork:
			iNode := p.nodeTable.getDevice(work.identifier)

			if iNode != nil {
				ctx, cancel := context.WithTimeout(context.Background(), workerMaximumJobDuration)

				if work.fn(ctx, iNode) {
					time.AfterFunc(work.interval, func() {
						p.pollerWork <- work
					})
				}

				cancel()
			}
		case <-p.pollerStop:
			return
		}
	}
}
