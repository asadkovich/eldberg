package blockchain

import (
	"github.com/AlexeySadkovich/eldberg/internal/blockchain/crypto"
)

type MerkleTree struct {
	transactions []*Transaction
	length       int
}

func NewMerkleTree(transactions []*Transaction) *MerkleTree {
	return &MerkleTree{transactions, len(transactions)}
}

func (mt *MerkleTree) AddTransaction(tx *Transaction) {
	mt.transactions = append(mt.transactions, tx)
}

func (mt *MerkleTree) CalculateRoot() []byte {
	var subTree [][]byte

	if mt.length%2 != 0 {
		lastTx := mt.transactions[mt.length-1]
		mt.transactions = append(mt.transactions, lastTx)
		mt.length = len(mt.transactions)
	}

	for i := 0; i < mt.length; i += 2 {
		tx1 := mt.transactions[i]
		tx2 := mt.transactions[i+1]

		txsBytes := append(tx1.Bytes(), tx2.Bytes()...)

		hashBytes := crypto.Hash(txsBytes)

		subTree = append(subTree, hashBytes)
	}

	if len(subTree) == 1 {
		return subTree[0]
	}

	root := mt.calculate(subTree)

	return root
}

func (mt *MerkleTree) calculate(tree [][]byte) []byte {
	var subTree [][]byte

	for i := 0; i < len(tree)-1; i += 2 {
		nodesBytes := append(tree[i], tree[i+1]...)

		hashBytes := crypto.Hash(nodesBytes)

		subTree = append(subTree, hashBytes)
	}

	if len(subTree) == 1 {
		return subTree[0]
	}

	return mt.calculate(subTree)
}