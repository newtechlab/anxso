package main

func analysTransactions(trans []TXO, known []uint32) (guesses []uint32, err error) {
	return
}

type AnalysisInfo interface {
	// GetTXO requires a proof that you should be able to see this transaction, providing
	// a chain that leads back to a transaction in which you were involved.
	GetTXO(id uint32, chain []uint32) (TXO, error)
}

func NaiveAnalysis(incoming, outgoing []TXO, identities map[uint32]uint32, ai AnalysisInfo) (txoSeen []TXO, identified map[uint32]uint32) {
	// First gather all the transactions that I can get based on tracing the ones I have been involved
	// in backwards and store them
	transs := append(incoming, outgoing...)
	kTrans := make(map[uint32]TXO)
	var rec func(uint32, []uint32)
	rec = func(tid uint32, chain []uint32) {
		if _, ok := kTrans[tid]; ok {
			return
		}
		// get this one using the chain, and recurse down, building the chain
		t, e := ai.GetTXO(tid, chain)
		if e != nil {
			panic(e.Error())
		}
		kTrans[t.id] = t

		if t.from != nil {
			for _, tt := range t.from {
				rec(tt, append(chain, tid))
			}
		}
	}
	for _, t := range transs {
		rec(t.id, []uint32{})
	}
	txoSeen = make([]TXO, 0, len(kTrans))
	for _, v := range kTrans {
		txoSeen = append(txoSeen, v)
	}

	// And now try to find out which transactions belong to which bank
	identified = make(map[uint32]uint32)
	for k, v := range identities {
		identified[k] = v
	}
	// One by one move out the items from kTrans if we can identify the playes, keep this going
	// untile we have been through one complete round without adding any one.
	for nod := 1; nod > 0; {
		nod = 0
	Outer:
		for id, t := range kTrans {
			// If we know any one of the from we know all of them, so update identified
			for _, fid := range t.from {
				if iden, ok := identified[fid]; ok {
					// // so update all the from, remove it and continue
					for _, ffid := range t.from {
						identified[ffid] = iden
					}
					delete(kTrans, id)
					nod++
					continue Outer
				}
			}
		}
		// TODO: CAN WE DO MORE HERE, ANY OTHER DETERMINSITIC LOGIC WE CAN USE TO CORRELATE MORE? ANYTHING IF WE DO PROBABILISTIC?
	}

	return txoSeen, identified
}
