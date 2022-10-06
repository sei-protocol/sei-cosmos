package accesscontrol

import (
	"fmt"

	"github.com/k0kubun/pp"
)

// Alias for Map of MessageIndex -> AccessOperation -> Channel
type MessageAccessOpsChannelMapping = map[int]AccessOpsChannelMapping

// Alias for Map of AccessOperation -> Channel
type AccessOpsChannelMapping = map[AccessOperation][]chan interface{}


func WaitForAllSignals(txIndex int, messageIndexToAccessOpsChannelMapping MessageAccessOpsChannelMapping) {
	println(fmt.Printf("%d=========WaitForallSignals:: Waiting for signals =========", txIndex))
	pp.Println(messageIndexToAccessOpsChannelMapping)
	for _, accessOpsToChannelsMap  := range messageIndexToAccessOpsChannelMapping {
		for _, channels := range accessOpsToChannelsMap {
			for _, channel := range channels {
				println(fmt.Printf("%d==WaitForallSignals:: Waiting", txIndex))
				<-channel
				println(fmt.Printf("%d==WaitForallSignals:: Got Signal", txIndex))
			}
		}
	}
	println("WaitForallSignals:: Recieved all signals=========")
}

func SendAllSignals(txIndex int, messageIndexToAccessOpsChannelMapping MessageAccessOpsChannelMapping) {
	println(fmt.Printf("%d=========SendAllSignals:: Sending Signals =========", txIndex))
	pp.Println(messageIndexToAccessOpsChannelMapping)
	for _, accessOpsToChannelsMap  := range messageIndexToAccessOpsChannelMapping {
		for _, channels := range accessOpsToChannelsMap {
			for _, channel := range channels {
				println(fmt.Printf("%d==SendAllSignals:: Sending Signal", txIndex))
				channel <- struct{}{}
				println(fmt.Printf("%d==SendAllSignals:: Sent Signal", txIndex))
			}
		}
	}
	println(fmt.Printf("%d==SendAllSignals:: Sent All Signals", txIndex))
}
