package main // import "newtechlab.wtf/anxso"

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
)

func (t TXO) isTransaction(owners []uint32) bool {
	if len(t.from) <= 0 {
		return false // This is money creation, not a transactions
	}
	if owners[t.id] == owners[t.from[0]] {
		return false // This was sent from me to me, so not a transaction between banks
	}
	return true
}

func (t TXO) equal(t2 TXO) bool {
	if t.id != t2.id {
		return false
	}
	if t.value != t2.value {
		return false
	}
	if len(t.from) != len(t2.from) {
		return false
	}
	for i := range t.from {
		if t.from[i] != t2.from[i] {
			return false
		}
	}
	return true
}

// Returns true if bank is directly involved in the transaction
func (t TXO) isInvolved(bank uint32, owners []uint32) bool {
	if owners[t.id] == bank {
		return true
	}
	return len(t.from) > 0 && owners[t.from[0]] == bank
}

func main() {
	noEach := 1
	noits := 10000000
	noSnip := []int{
		1e6,
		9e5, 8e5, 7e5, 6e5, 5e5, 4e5, 3e5, 2e5, 1e5,
		9e4, 8e4, 7e4, 6e4, 5e4, 4e4, 3e4, 2e4, 1e4,
		9e3, 8e3, 7e3, 6e3, 5e3, 4e3, 3e3, 2e3, 1e3,
		9e2, 8e2, 7e2, 6e2, 5e2, 4e2, 3e2, 2e2, 1e2,
		9e1, 8e1, 7e1, 6e1, 5e1, 4e1, 3e1, 2e1, 1e1,
		9e0, 8e0, 7e0, 6e0, 5e0, 4e0, 3e0, 2e0, 1e0,
	}
	results := make([][]float64, 0, noEach*len(noSnip))
	banks := []float64{1.7, 1.3, 1, 1, 0.4, 0.1}
	done := 0
	for _, snip := range noSnip {
		for i := 0; i < noEach; i++ {
			log.Printf("doing %5d/%5d", done, cap(results))
			done++

			stats := run(i, banks, noits, snip, UniformSampler{low: 100, high: 10000})
			A := stats.bstat[0]
			B := stats.bstat[1]
			C := stats.bstat[2]
			D := stats.bstat[4]
			bothA := float64(A.noExtraTotal) / float64(stats.noTrans-A.noTransInvolved)
			bothB := float64(B.noExtraTotal) / float64(stats.noTrans-B.noTransInvolved)
			bothC := float64(C.noExtraTotal) / float64(stats.noTrans-C.noTransInvolved)
			bothD := float64(D.noExtraTotal) / float64(stats.noTrans-D.noTransInvolved)
			sizeA := float64(A.noTransInvolved) / float64(stats.noTrans)
			sizeB := float64(B.noTransInvolved) / float64(stats.noTrans)
			sizeC := float64(C.noTransInvolved) / float64(stats.noTrans)
			sizeD := float64(D.noTransInvolved) / float64(stats.noTrans)
			results = append(results, []float64{
				float64(len(results)),
				float64(snip),
				float64(i),
				sizeA,
				bothA,
				sizeB,
				bothB,
				sizeC,
				bothC,
				sizeD,
				bothD,
			})
		}
	}

	// Print out the results as a table
	fmt.Println(`id	snip	seed	A-size	A-both	B-size	B-both	C-size	C-both	D-size	D-both`)
	for _, it := range results {
		tprint(it)
	}
}

func tprint(its []float64) {
	for i := 0; i < len(its)-1; i++ {
		fmt.Print(its[i], `	`)
	}
	fmt.Println(its[len(its)-1])
}

func run(seed int, banks []float64, noits int, maxDepth int, l Logic) stats {

	r := rand.New(rand.NewSource(int64(seed)))

	// Generate the transactions we will operate on
	gen := NewGenerator(banks, noits, r, maxDepth, l)
	txos, owners := gen.Generate(false)

	// Build the statistics that we will be printing
	stats := stats{
		noTxo: len(txos),
		bstat: make([]bstat, len(banks)),
	}
	for _, t := range txos {
		if t.isTransaction(owners) {
			stats.noTrans++
		}
	}

	// For each of the banks run the analysis engine to learn how much
	// it could gather about the total history, being provided only the
	// information that bank would have seen. Swapping id between bank 0
	// and the current bank such that the analysis engine always sees itself
	// as bank 0.
	for bi, b := range gen.banks {
		mif := &myinfo{
			txos:   txos,
			owners: owners,
			owner:  b.id,
		}
		incoming := make([]TXO, 0, 100)
		outgoing := make([]TXO, 0, 100)
		identities := make(map[uint32]uint32, 100)
		for _, t := range txos {
			if owners[t.id] == b.id {
				incoming = append(incoming, t)
				identities[t.id] = owners[t.id]
				for _, tin := range t.from {
					identities[tin] = owners[tin]
				}
			}
			if len(t.from) > 0 && owners[t.from[0]] == b.id {
				// assuming all from are from the same owner
				identities[t.id] = owners[t.id]
				outgoing = append(outgoing, t)
			}
		}

		txoSeen, identified := NaiveAnalysis(incoming, outgoing, identities, mif)

		// Analyse and make sure the results are correct from the algo, update the statistics that
		// we should be printing to the user.
		bstat := bstat{
			noTxoSeen:         len(txoSeen),
			noTransSeen:       0,
			noTransInvolved:   0,
			noTransOtherSeen:  0,
			noExtraIdentified: len(identified) - len(identities), // TODO: This must be checked so the results are correct!
			noExtraSender:     0,
			noExtraReceiver:   0,
			noExtraTotal:      0,
		}
		for k, v := range identified {
			// Check that all of them are correct, we do not allow incorrect guesses
			if owners[k] != v {
				panic("got back incorrect identified")
			}
		}
		for _, txo := range txoSeen {
			// only interested in real transactions for the stats
			if txo.isInvolved(b.id, owners) {
				continue
			}
			if !txo.isTransaction(owners) {
				continue
			}
			co, ok := identified[txo.id]
			dr, ds := false, false
			if ok {
				if co == owners[txo.id] {
					bstat.noExtraReceiver++
					dr = true
				} else {
					panic("got wrong guess for rec")
				}
			}
			co, ok = identified[txo.from[0]]
			if ok {
				if co == owners[txo.from[0]] {
					bstat.noExtraSender++
					ds = true
				} else {
					panic("got wrong guess for sender")
				}
			}
			if dr && ds {
				bstat.noExtraTotal++
			}
		}
		for _, txo := range txoSeen {
			if !txo.equal(txos[txo.id]) {
				panic("returned a transaction that does not exist")
			}
			if txo.isTransaction(owners) {
				bstat.noTransSeen++
				// Check if we were involved or not:
				if txo.isInvolved(b.id, owners) {
					bstat.noTransInvolved++
				} else {
					bstat.noTransOtherSeen++
				}
			}
		}
		stats.bstat[bi] = bstat
	}

	//fmt.Println(stats)
	//	PrintTransList(txos, owners)
	return stats
}

type stats struct {
	noTxo   int // total number of txos simulated
	noTrans int // total number of transactions Between Banks carried out

	bstat []bstat // the statistics for each individual bank
}

func (bs stats) String() string {
	s := ""
	s += fmt.Sprintln("Number of txos: ", bs.noTxo)
	s += fmt.Sprintln("Number of transactions: ", bs.noTrans)
	for i, b := range bs.bstat {
		s += `
`
		s += fmt.Sprintln("Bank " + string('A'+rune(i)))

		s += b.String(bs)
	}
	return s
}

type bstat struct {
	noTxoSeen         int // Total number of txos this bank has seen
	noTransSeen       int // Total number of transactions seen by this bank
	noTransInvolved   int // Total number of transactions this bank have been involved int
	noTransOtherSeen  int // Total number of transactions seen in which not involved at all
	noExtraIdentified int // Number of additional identities found
	noExtraSender     int
	noExtraReceiver   int
	noExtraTotal      int
}

func (bs bstat) String(stats stats) string {
	txoNotInvolved := stats.noTrans - bs.noTransInvolved
	s := ""
	s += fmt.Sprintf("txos seen: %.2f%% (%v)", 100*float64(bs.noTxoSeen)/float64(stats.noTxo), bs.noTxoSeen)
	s += fmt.Sprintln("")
	s += fmt.Sprintf("transactions seen: %.2f%% (%v)", 100*float64(bs.noTransSeen)/float64(stats.noTrans), bs.noTransSeen)
	s += fmt.Sprintln("")
	s += fmt.Sprintf("transactions involved in: %.2f%% (%v)", 100*float64(bs.noTransInvolved)/float64(stats.noTrans), bs.noTransInvolved)
	s += fmt.Sprintln("")
	s += fmt.Sprintf("transactions not involved in: %.2f%% (%v)", 100*float64(txoNotInvolved)/float64(stats.noTrans), txoNotInvolved)
	s += fmt.Sprintln("")
	s += fmt.Sprintf("transactions not involved in but seen: %.2f%% (%v)", 100*float64(bs.noTransOtherSeen)/float64(stats.noTrans), bs.noTransOtherSeen)
	s += fmt.Sprintln("")
	s += fmt.Sprintf("extra identities identified: %v", bs.noExtraIdentified)
	s += fmt.Sprintln("")
	s += fmt.Sprintf("uninvolved transactions identified sender: %.2f%% (%v)", 100*float64(bs.noExtraSender)/float64(txoNotInvolved), bs.noExtraSender)
	s += fmt.Sprintln("")
	s += fmt.Sprintf("uninvolved transactions identified reciever: %.2f%% (%v)", 100*float64(bs.noExtraReceiver)/float64(txoNotInvolved), bs.noExtraReceiver)
	s += fmt.Sprintln("")
	s += fmt.Sprintf("uninvolved transaction identified both: %.2f%% (%v)", 100*float64(bs.noExtraTotal)/float64(txoNotInvolved), bs.noExtraTotal)
	s += fmt.Sprintln("")
	return s
}

type myinfo struct {
	txos   []TXO
	owners []uint32
	owner  uint32
}

func (mi *myinfo) GetTXO(id uint32, chain []uint32) (TXO, error) {
	// follow the chain down, ensuring that we in each step are either
	// a sender or a reciever, once we have checked the chain, also check
	// the last item.
	chain = append(chain, id)
	lid := -1
	for _, id := range chain {
		// ensure that we are either owner of this transaction, or that
		// the "beforeitem" in the chain depends on this such that we could
		// get it.
		if mi.owners[id] == mi.owner {
			lid = int(id)
			continue
		}
		t := mi.txos[id]
		if len(t.from) > 0 && mi.owners[t.from[0]] == mi.owner {
			lid = int(id)
			continue
		}
		if lid > 0 {
			if contains(id, mi.txos[lid].from) {
				lid = int(id)
				continue
			}
		}
		return TXO{}, errors.New("requested a transaction you should not have requested")
	}
	return mi.txos[id], nil
}

func contains(id uint32, ids []uint32) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}
