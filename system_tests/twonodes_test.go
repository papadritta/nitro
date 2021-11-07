//
// Copyright 2021, Offchain Labs, Inc. All rights reserved.
//

package arbtest

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/offchainlabs/arbstate/arbnode"
)

func Create2ndNode(t *testing.T, first *arbnode.Node, l1client arbnode.L1Interface) *arbnode.Node {
	stack, err := arbnode.CreateStack()
	if err != nil {
		t.Fatal(err)
	}
	backend, err := arbnode.CreateArbBackend(stack, l2Genesys)
	if err != nil {
		t.Fatal(err)
	}
	node, err := arbnode.CreateNode(l1client, first.DeployInfo, backend, nil, true)
	if err != nil {
		t.Fatal(err)
	}
	node.Start(context.Background())
	return node
}

func TestTwoNodes(t *testing.T) {
	background := context.Background()
	l2backend, l2info := CreateTestL2(t)
	l1info, node1, _, l1stack := CreateTestNodeOnL1(t, l2backend, true)

	l1rpcClientB, err := l1stack.Attach()
	if err != nil {
		t.Fatal(err)
	}
	l1clientB := ethclient.NewClient(l1rpcClientB)
	nodeB := Create2ndNode(t, node1, l1clientB)
	l2clientB := ClientForArbBackend(t, nodeB.Backend)

	l2info.GenerateAccount("User2")

	tx := l2info.PrepareTx("Owner", "User2", 30000, big.NewInt(1e12), nil)

	ctx := context.Background()

	err = l2info.Client.SendTransaction(ctx, tx)
	if err != nil {
		t.Fatal(err)
	}

	_, err = arbnode.EnsureTxSucceeded(l2info.Client, tx)
	if err != nil {
		t.Fatal(err)
	}

	// give the inbox reader a bit of time to pick up the delayed message
	time.Sleep(time.Millisecond * 100)

	// sending l1 messages creates l1 blocks.. make enough to get that delayed inbox message in
	for i := 0; i < 30; i++ {
		SendWaitTestTransactions(t, l1info.Client, []*types.Transaction{
			l1info.PrepareTx("faucet", "User", 30000, big.NewInt(1e12), nil),
		})
	}

	_, err = arbnode.WaitForTx(l2clientB, tx.Hash(), time.Second*5)
	if err != nil {
		t.Fatal(err)
	}
	l2balance, err := l2clientB.BalanceAt(background, l2info.GetAddress("User2"), nil)
	if err != nil {
		t.Fatal(err)
	}
	if l2balance.Cmp(big.NewInt(1e12)) != 0 {
		t.Fatal("Unexpected balance:", l2balance)
	}
}
