package accesscontrol

// Alias for Map of MessageIndex -> AccessOperation -> Channel
type MessageAccessOpsChannelMapping = map[int]AccessOpsChannelMapping

// Alias for Map of AccessOperation -> Channel
type AccessOpsChannelMapping = map[AccessOperation][]chan interface{}

func WaitForAllSignals(accessOpsToChannelsMap AccessOpsChannelMapping) {
	println("WaitForallSignals:: Waiting")
	for _, channels := range accessOpsToChannelsMap {
		for _, channel := range channels {
			<-channel
		}
	}
	println("WaitForallSignals:: Recieved all signals")
}

func SendAllSignals(accessOpsToChannelsMap AccessOpsChannelMapping) {
	println("SendAllSignal:: Preparing to send")
	for _, channels := range accessOpsToChannelsMap {
		for _, channel := range channels {
			channel <- struct{}{}
		}
	}
	println("SendAllSignal:: Sent")
}
