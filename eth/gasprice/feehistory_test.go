// (c) 2019-2020, Ava Labs, Inc.
//
// This file is a derived work, based on the go-ethereum library whose original
// notices appear below.
//
// It is distributed under a license compatible with the licensing terms of the
// original code from which it is derived.
//
// Much love to the original authors for their work.
// **********
// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package gasprice

import (
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ava-labs/subnet-evm/core"
	"github.com/ava-labs/subnet-evm/core/types"

	"github.com/ava-labs/subnet-evm/params"
	"github.com/ava-labs/subnet-evm/rpc"
	"github.com/ethereum/go-ethereum/common"
)

func TestFeeHistory(t *testing.T) {
	cases := []struct {
		pending             bool
		maxHeader, maxBlock int
		count               int
		last                rpc.BlockNumber
		percent             []float64
		expFirst            uint64
		expCount            int
		expErr              error
	}{
		{false, 1000, 1000, 10, 30, nil, 21, 10, nil},
		{false, 1000, 1000, 10, 30, []float64{0, 10}, 21, 10, nil},
		{false, 1000, 1000, 10, 30, []float64{20, 10}, 0, 0, errInvalidPercentile},
		{false, 1000, 1000, 1000000000, 30, nil, 0, 31, nil},
		{false, 1000, 1000, 1000000000, rpc.LatestBlockNumber, nil, 0, 33, nil},
		{false, 1000, 1000, 10, 40, nil, 0, 0, errRequestBeyondHead},
		{true, 1000, 1000, 10, 40, nil, 0, 0, errRequestBeyondHead},
		{false, 20, 2, 100, rpc.LatestBlockNumber, nil, 13, 20, nil},
		{false, 20, 2, 100, rpc.LatestBlockNumber, []float64{0, 10}, 31, 2, nil},
		{false, 20, 2, 100, 32, []float64{0, 10}, 31, 2, nil},
		{false, 1000, 1000, 1, rpc.PendingBlockNumber, nil, 0, 0, nil},
		{false, 1000, 1000, 2, rpc.PendingBlockNumber, nil, 32, 1, nil},
		{true, 1000, 1000, 2, rpc.PendingBlockNumber, nil, 32, 1, nil},
		{true, 1000, 1000, 2, rpc.PendingBlockNumber, []float64{0, 10}, 32, 1, nil},
	}
	for i, c := range cases {
		config := Config{
			MaxHeaderHistory: c.maxHeader,
			MaxBlockHistory:  c.maxBlock,
		}
		tip := big.NewInt(1 * params.GWei)
		backend := newTestBackendFakerEngine(t, params.TestChainConfig, 32, func(i int, b *core.BlockGen) {
			signer := types.LatestSigner(params.TestChainConfig)

			b.SetCoinbase(common.Address{1})

			baseFee := b.BaseFee()
			feeCap := new(big.Int).Add(baseFee, tip)

			var tx *types.Transaction
			txdata := &types.DynamicFeeTx{
				ChainID:   params.TestChainConfig.ChainID,
				Nonce:     b.TxNonce(addr),
				To:        &common.Address{},
				Gas:       params.TxGas,
				GasFeeCap: feeCap,
				GasTipCap: tip,
				Data:      []byte{},
			}
			tx = types.NewTx(txdata)
			tx, err := types.SignTx(tx, signer, key)
			if err != nil {
				t.Fatalf("failed to create tx: %v", err)
			}
			b.AddTx(tx)
		})
		oracle := NewOracle(backend, config)

		first, reward, baseFee, ratio, err := oracle.FeeHistory(context.Background(), c.count, c.last, c.percent)

		expReward := c.expCount
		if len(c.percent) == 0 {
			expReward = 0
		}
		expBaseFee := c.expCount

		if first.Uint64() != c.expFirst {
			t.Fatalf("Test case %d: first block mismatch, want %d, got %d", i, c.expFirst, first)
		}
		if len(reward) != expReward {
			t.Fatalf("Test case %d: reward array length mismatch, want %d, got %d", i, expReward, len(reward))
		}
		if len(baseFee) != expBaseFee {
			t.Fatalf("Test case %d: baseFee array length mismatch, want %d, got %d", i, expBaseFee, len(baseFee))
		}
		if len(ratio) != c.expCount {
			t.Fatalf("Test case %d: gasUsedRatio array length mismatch, want %d, got %d", i, c.expCount, len(ratio))
		}
		if err != c.expErr && !errors.Is(err, c.expErr) {
			t.Fatalf("Test case %d: error mismatch, want %v, got %v", i, c.expErr, err)
		}
	}
}
