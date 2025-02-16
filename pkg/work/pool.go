package work

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

type Executor interface {
	Execute() error
	OnError(error)
}

type Pool struct {
	numWorkers int
	tasks      chan Executor
	start      sync.Once
	stop       sync.Once
	quit       chan struct{}
}

func NewPool(numWorkers int, taskChannelSize int) (*Pool, error) {
	if numWorkers <= 0 {
		return nil, errors.New("numWorkers must be greater than 0")
	}
	if taskChannelSize < 0 {
		return nil, errors.New("taskChannelSize must be greater than or equal to 0")
	}
	return &Pool{
		numWorkers: numWorkers,
		tasks:      make(chan Executor, taskChannelSize),
		start:      sync.Once{},
		stop:       sync.Once{},
		quit:       make(chan struct{}),
	}, nil
}

func (p *Pool) Start(ctx context.Context) {
	p.start.Do(func() {
		p.startWorker(ctx)
	})
}

func (p *Pool) Stop() {
	p.stop.Do(func() {
		close(p.quit)
	})
}

func (p *Pool) AddTask(t Executor) {
	select {
	case p.tasks <- t:
	case <-p.quit:
	}
}

func (p *Pool) startWorker(ctx context.Context) {
	for i := 0; i < p.numWorkers; i++ {
		go func(workerNum int) {
			fmt.Printf("worker number %d started\n", workerNum)
			for {
				select {
				case task, ok := <-p.tasks:
					if !ok {
						return
					}
					if err := task.Execute(); err != nil {
						task.OnError(err)
					}
				case <-ctx.Done():
					return
				case <-p.quit:
					return
				}
			}
		}(i)
	}
}
