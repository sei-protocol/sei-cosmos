package accesscontrol

// Alias for Map of MessageIndex -> AccessOperation -> Channel
type MessageAccessOpsChannelMapping = map[int]AccessOpsChannelMapping

// Alias for Map of AccessOperation -> Channel
type AccessOpsChannelMapping = map[AccessOperation][]chan interface{}


func WaitForAllSignals(messageIndexToAccessOpsChannelMapping MessageAccessOpsChannelMapping) {
	println("WaitForallSignals:: Waiting for signals =========")
	for _, accessOpsToChannelsMap  := range messageIndexToAccessOpsChannelMapping {
		for _, channels := range accessOpsToChannelsMap {
			for i, channel := range channels {
				println("WaitForallSignals:: Waiting", i)
				<-channel
				println("WaitForallSignals:: Got Signal!", i)
			}
		}
	}
	println("WaitForallSignals:: Recieved all signals=========")
}

func SendAllSignals(messageIndexToAccessOpsChannelMapping MessageAccessOpsChannelMapping) {
	println("SendAllSignal:: Preparing to send=========")
	for _, accessOpsToChannelsMap  := range messageIndexToAccessOpsChannelMapping {
		for _, channels := range accessOpsToChannelsMap {
			for i, channel := range channels {
				println("SendAllSignal:: Sending", i)
				channel <- struct{}{}
				println("SendAllSignal:: Sent", i)
			}
		}
	}
	println("SendAllSignal:: Sent all Signals=========")
}
