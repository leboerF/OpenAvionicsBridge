package main

func lookupFMSMessageOffset(messages map[uint32]uint64, selector uint32) (uint64, bool) {
	if selector == 0 || len(messages) == 0 {
		return 0, false
	}
	off, ok := messages[selector]
	return off, ok
}
