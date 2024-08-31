package llm

import "fmt"

type ChunkReader struct {
	chunkChan chan string
	doneChan  chan struct{}
}

func NewChunkReader() *ChunkReader {
	return &ChunkReader{}
}

func (c *ChunkReader) Enable() {
	c.chunkChan = make(chan string)
	c.doneChan = make(chan struct{})

	go func() {
		defer close(c.doneChan)
		for chunk := range c.chunkChan {
			fmt.Print(chunk)
		}
		fmt.Println()
	}()
}

func (c *ChunkReader) Wait() {
	if c.doneChan != nil {
		<-c.doneChan
	}
}

func (c *ChunkReader) Chan() chan<- string {
	return c.chunkChan
}
