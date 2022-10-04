package accesscontrol

func (a *AccessOperation) GetResourceID(msg interface{}) string {
	// If the Msg implements GetMsgResourceIdentifier, then use that to retrieve the identifier, or else
	// use the Identifier template as the resourceId for this operation
	if message, ok := msg.(interface{GetMsgResourceIdentifier(a AccessOperation) string}); ok {
		return message.GetMsgResourceIdentifier(*a)
	}
	return a.GetIdentifierTemplate()
}
