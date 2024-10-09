package broadcaster

import (
	"net/http"

	"github.com/vechain/thor/v2/chain"
	"github.com/vechain/thor/v2/thor"
)

// MessageHandler is a function that takes a block and returns a message to be broadcasted.
type MessageHandler func(repo *chain.Repository, block *chain.ExtendedBlock) (interface{}, error)

// Broadcaster is used to send common messages to all clients on every new block.
type Broadcaster struct {
	repo    *chain.Repository
	handler MessageHandler
	hub     *Hub
}

func New(repo *chain.Repository, handler MessageHandler) *Broadcaster {
	b := &Broadcaster{
		repo:    repo,
		handler: handler,
		hub:     NewHub(),
	}

	go b.hub.run()
	go b.start()

	return b
}

// Add adds a new client to the broadcaster. If the client has requested a historical position, it will send all blocks
// from that position to the client, then add it to the broadcaster.
// TODO: this needs extensive testing, what if a new block is added while sending the historic blocks?
func (br *Broadcaster) Add(w http.ResponseWriter, r *http.Request, pos thor.Bytes32) error {
	client := NewClient(br.hub, w, r)
	if pos != br.repo.NewBestChain().HeadID() {
		if err := br.sendHistoric(client, pos); err != nil {
			return err
		}
	}
	client.Start()
	return nil
}

// start starts the broadcaster. It listens for new blocks, creates the message and broadcasts it.
func (br *Broadcaster) start() {
	for {
		<-br.repo.NewTicker().C()
		best, err := br.repo.GetBlock(br.repo.BestBlockSummary().Header.ID())
		if err != nil {
			continue
		}
		msg, err := br.handler(br.repo, &chain.ExtendedBlock{Block: best, Obsolete: false})
		if err != nil {
			continue
		}
		br.hub.Broadcast(msg)
	}
}

// sendHistoric loops through all blocks from the given position and sends them to the client.
func (br *Broadcaster) sendHistoric(client *Client, pos thor.Bytes32) error {
	reader := br.repo.NewBlockReader(pos)
	for {
		blocks, err := reader.Read()
		if err != nil {
			return err
		}
		if len(blocks) == 0 {
			break
		}
		for _, block := range blocks {
			msg, err := br.handler(br.repo, block)
			if err != nil {
				return err
			}
			if err := client.WriteJSON(msg); err != nil {
				return err
			}
		}
	}
	return nil
}
