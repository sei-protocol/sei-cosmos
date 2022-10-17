package accesscontrol

import (
	"log"

	"github.com/k0kubun/pp"
)

// Alias for Map of MessageIndex -> AccessOperation -> Channel
type MessageAccessOpsChannelMapping = map[int]AccessOpsChannelMapping

// Alias for Map of AccessOperation -> Channel
type AccessOpsChannelMapping = map[AccessOperation][]chan interface{}

func WaitForAllSignalsForTx(messageIndexToAccessOpsChannelMapping MessageAccessOpsChannelMapping) {
	log.Printf("Waiting for unblocker")
	pp.Printf("Waiting MessageIndexToAccessOp: %s", messageIndexToAccessOpsChannelMapping)
	for _, accessOpsToChannelsMap  := range messageIndexToAccessOpsChannelMapping {
		for _, channels := range accessOpsToChannelsMap {
			for _, channel := range channels {
				<-channel
			}
		}
	}
}

func SendAllSignalsForTx(messageIndexToAccessOpsChannelMapping MessageAccessOpsChannelMapping) {
	log.Printf("Waiting for signalling")
	pp.Printf("Signalling MessageIndexToAccessOp: %s", messageIndexToAccessOpsChannelMapping)
	for _, accessOpsToChannelsMap  := range messageIndexToAccessOpsChannelMapping {
		for _, channels := range accessOpsToChannelsMap {
			for _, channel := range channels {
				channel <- struct{}{}
			}
		}
	}
}

