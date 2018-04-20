package main

import (
	"fmt"
	"math/rand"
	"sort"
)

type Generator struct {
	*rand.Rand

	TXOs   []TXO
	owners []uint32

	maxLength int // The maximum length of a sequence before snipping it

	banks []*Bank
	probs []float64

	no int
	l  Logic
}

type Bank struct {
	id  uint32
	myt []TXO // My existing TXOs that I could use
	g   *Generator
}

type TXO struct {
	id    uint32
	value uint32
	depth uint32
	from  []uint32
}

type Logic interface {
	TransactionSize(*rand.Rand) uint32
}

func NewGenerator(sizes []float64, no int, r *rand.Rand, maxLength int, l Logic) *Generator {
	g := &Generator{
		l:         l,
		TXOs:      make([]TXO, 0, no),
		owners:    make([]uint32, no),
		Rand:      r,
		probs:     make([]float64, len(sizes)),
		banks:     make([]*Bank, len(sizes)),
		maxLength: maxLength,
	}

	// Build the banks, and the array we use to sample transactions from
	sum := 0.0
	for i := range g.banks {
		g.banks[i] = NewBank(uint32(i), g)
		sum += sizes[i]
	}
	for i := range g.probs {
		if i == 0 {
			g.probs[i] = sizes[i] / sum
		} else {
			g.probs[i] = g.probs[i-1] + sizes[i]/sum
		}
	}
	return g
}

// Get a random bank that is not not, to get any random bank simply send not as negative
func (g *Generator) rBank(not int) *Bank {
	for {
		f := g.Float64()
		ind := sort.SearchFloat64s(g.probs, f)
		if ind != not {
			return g.banks[ind]
		}
	}
}

func (g *Generator) Generate(log bool) (transactions []TXO, owners []uint32) {
	ltid := 0
	for len(g.TXOs) < cap(g.TXOs) {
		// Which banks should we operate between
		fBank := g.rBank(-1)
		tBank := g.rBank(int(fBank.id))

		g.createTXO(fBank, tBank)
		if log {
			// log the transaction and the state of the banks at this position
			for j, t := range g.TXOs[ltid:len(g.TXOs)] {
				i := j + ltid
				fmt.Printf(`id: %3d	ow: %v	val: %v	from: %v
`, t.id, string('A'+rune(g.owners[i])), t.value, t.from)
			}
			ltid = len(g.TXOs)
			s := 0
			for _, b := range g.banks {
				sum := 0
				for _, t := range b.myt {
					sum += int(t.value)
				}
				fmt.Println("	", string('A'+rune(b.id)), ":", sum, len(b.myt))
				s += sum
			}
			fmt.Println("	", s)
		}
		_ = 3
	}
	return g.TXOs, g.owners
}

func PrintTransList(transactions []TXO, owners []uint32) {
	for i, t := range transactions {
		fmt.Printf(`id: %3d	ow: %v	val: %v	from: %v
`, t.id, string('A'+rune(owners[i])), t.value, t.from)
	}
}

func (g *Generator) createTXO(fBank, tBank *Bank) {
	amount := uint64(g.l.TransactionSize(g.Rand))

	// Figure out which TXOs we should use, by being carful we here we insert TXOs
	// we can be sure that they are always in sorted order - allowing us to use the
	// simple algorithm of simply using the smallest of the available ones.
	sum := uint64(0)
	id := 0
	maxDepth := uint32(0)
	for i := range fBank.myt {
		sum += uint64(fBank.myt[i].value)
		if fBank.myt[i].depth > maxDepth {
			maxDepth = fBank.myt[i].depth
		}
		if sum >= amount {
			break
		}
		id++
	}

	// If we do not have enough, create more money
	if len(fBank.myt) == 0 || id >= len(fBank.myt) {
		g.owners[len(g.TXOs)] = fBank.id
		g.TXOs = append(g.TXOs, TXO{
			id:    uint32(len(g.TXOs)),
			from:  []uint32{},
			value: uint32(amount * 3),
			depth: 0,
		})
		fBank.myt = append(fBank.myt, g.TXOs[len(g.TXOs)-1])
		sum += amount * 3
	}

	// Create the new TXO(s) and add them, making sure that we have enough space
	// to actually add them
	from := []uint32{}
	for i := 0; i <= id; i++ {
		from = append(from, fBank.myt[i].id)
	}
	fBank.myt = fBank.myt[id+1:]
	if len(g.TXOs) < cap(g.TXOs) {
		// The transaction to the reciever
		g.owners[len(g.TXOs)] = tBank.id
		g.TXOs = append(g.TXOs, TXO{
			id:    uint32(len(g.TXOs)),
			from:  from,
			value: uint32(amount),
			depth: maxDepth + 1,
		})
		// Insert the transaction at the correct place
		ind := sort.Search(len(tBank.myt), func(i int) bool { return uint32(amount) < tBank.myt[i].value })
		tBank.myt = append(tBank.myt, TXO{})
		copy(tBank.myt[ind+1:len(tBank.myt)], tBank.myt[ind:len(tBank.myt)-1])
		tBank.myt[ind] = g.TXOs[len(g.TXOs)-1]

		g.maybeSnipp(tBank, ind)
		//tBank.myt = append(tBank.myt[:ind], g.TXOs[len(g.TXOs)-1])
		//tBank.myt = append(tBank.myt, tBank.myt[ind+1:]...)
	}

	rem := uint32(sum - amount)
	if len(g.TXOs) < cap(g.TXOs) && rem > 0 {
		// The transaction to the sender if needed
		g.owners[len(g.TXOs)] = fBank.id
		g.TXOs = append(g.TXOs, TXO{
			id:    uint32(len(g.TXOs)),
			from:  from,
			value: uint32(rem),
			depth: maxDepth + 1,
		})
		// Insert the transaction at the correct place
		ind := sort.Search(len(fBank.myt), func(i int) bool { return uint32(rem) < fBank.myt[i].value })
		fBank.myt = append(fBank.myt, TXO{})
		copy(fBank.myt[ind+1:len(fBank.myt)], fBank.myt[ind:len(fBank.myt)-1])
		fBank.myt[ind] = g.TXOs[len(g.TXOs)-1]

		g.maybeSnipp(fBank, ind)
		//fBank.myt = append(fBank.myt[:ind], g.TXOs[len(g.TXOs)-1])
		//fBank.myt = append(fBank.myt, fBank.myt[ind+1:]...)
	}

}

// check if the given transaction indec in myt should be snipped, that is exchanged
// with a "central bank" creating a new one that cannot be traced backwards.
func (g *Generator) maybeSnipp(bank *Bank, ind int) {
	// first check the depth of the transaction
	if bank.myt[ind].depth >= uint32(g.maxLength) {
		// snipping involves simply creating a new transaction from the sentral bank
		// to this bank, thus under a new ID  - and replacing it in myt - thus the old one
		// will never be used and no one can use it, instead there will exist a new clean
		// transaction that no-one else can check. Also make sure that there is enough space
		// to actually create one additional transaction

		if len(g.TXOs) >= cap(g.TXOs) {
			return
		}
		g.owners[len(g.TXOs)] = bank.id
		g.TXOs = append(g.TXOs, TXO{
			id:    uint32(len(g.TXOs)),
			from:  []uint32{},
			value: bank.myt[ind].value,
			depth: 0,
		})
		bank.myt[ind] = g.TXOs[len(g.TXOs)-1]
	}
}

func NewBank(id uint32, g *Generator) *Bank {
	b := &Bank{
		id:  id,
		myt: make([]TXO, 0, 100),
		g:   g,
	}
	return b
}
