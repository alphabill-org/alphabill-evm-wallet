package evm

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"github.com/alphabill-org/alphabill-evm/txsystem/evm"
	"github.com/alphabill-org/alphabill-go-base/types"
	"github.com/stretchr/testify/require"

	sdktypes "github.com/alphabill-org/alphabill-wallet/client/types"
	test "github.com/alphabill-org/alphabill-wallet/internal/testutils"
	"github.com/alphabill-org/alphabill-wallet/wallet/account"
	evmclient "github.com/alphabill-org/alphabill-wallet/wallet/evm/client"
)

const (
	testMnemonic = "dinosaur simple verify deliver bless ridge monkey design venue six problem lucky"
)

func bigIntFromString(t *testing.T, value string) *big.Int {
	t.Helper()
	i, b := new(big.Int).SetString(value, 10)
	require.True(t, b)
	return i
}

type evmClientMock struct {
	SimulateErr error
	noFcb       bool
	gasPrice    string
}

func newClientMock() *evmClientMock {
	return &evmClientMock{}
}

func (e *evmClientMock) GetRoundNumber(ctx context.Context) (*evmclient.RoundNumber, error) {
	if e.SimulateErr != nil {
		return nil, e.SimulateErr
	}
	return &evmclient.RoundNumber{RoundNumber: 3, LastIndexedRoundNumber: 3}, nil
}

func (e *evmClientMock) PostTransaction(ctx context.Context, tx *types.TransactionOrder) error {
	if e.SimulateErr != nil {
		return e.SimulateErr
	}
	return nil
}

func (e *evmClientMock) GetTxProof(ctx context.Context, unitID types.UnitID, txHash sdktypes.TxHash) (*types.TxRecordProof, error) {
	if e.SimulateErr != nil {
		return nil, e.SimulateErr
	}
	details := evm.ProcessingDetails{
		ErrorDetails: "some error string",
	}
	encoded, _ := types.Cbor.Marshal(details)
	txBytes, _ := (&types.TransactionOrder{}).MarshalCBOR()
	return &types.TxRecordProof{
		TxRecord: &types.TransactionRecord{
			TransactionOrder: txBytes,
			ServerMetadata: &types.ServerMetadata{
				ActualFee:         1,
				TargetUnits:       []types.UnitID{test.RandomBytes(20)},
				SuccessIndicator:  0, // execution failed
				ProcessingDetails: encoded,
			},
		},
		TxProof: &types.TxProof{},
	}, nil
}

func (e *evmClientMock) Call(ctx context.Context, callAttr *evm.CallEVMRequest) (*evm.ProcessingDetails, error) {
	if e.SimulateErr != nil {
		return nil, e.SimulateErr
	}
	return &evm.ProcessingDetails{
		ErrorDetails: "actual execution failed",
	}, nil
}

func (e *evmClientMock) GetTransactionCount(ctx context.Context, ethAddr []byte) (uint64, error) {
	if e.SimulateErr != nil {
		return 0, e.SimulateErr
	}
	return uint64(1), nil
}

func (e *evmClientMock) GetBalance(ctx context.Context, ethAddr []byte) (string, uint64, error) {
	if e.SimulateErr != nil {
		return "", 0, e.SimulateErr
	}
	if e.noFcb {
		return "", 0, evmclient.ErrNotFound
	}
	return "100000", 0, nil
}

func (e *evmClientMock) GetFeeCreditBill(ctx context.Context, unitID types.UnitID) (*evmclient.Bill, error) {
	if e.SimulateErr != nil {
		return nil, e.SimulateErr
	}
	if e.noFcb {
		return nil, nil
	}
	return &evmclient.Bill{
		Id:    unitID,
		Value: 100 * 1e8,
	}, nil
}

func (e *evmClientMock) GetGasPrice(ctx context.Context) (string, error) {
	if e.SimulateErr != nil {
		return "", e.SimulateErr
	}
	if e.gasPrice != "" {
		return e.gasPrice, nil
	}
	return "100", nil
}

func createTestWallet(t *testing.T) (*Wallet, *evmClientMock) {
	dir := t.TempDir()
	am, err := account.NewManager(dir, "", true)
	require.NoError(t, err)
	clientMock := newClientMock()
	return &Wallet{
		partitionID: evm.DefaultPartitionID,
		am:          am,
		restCli:     clientMock,
	}, clientMock
}

func TestConvertBalanceToAlpha(t *testing.T) {
	type args struct {
		wei *big.Int
	}
	tests := []struct {
		name string
		args args
		want uint64
	}{
		{
			name: "1 wei is 0 alpha",
			args: args{wei: bigIntFromString(t, "1")},
			want: 0,
		},
		{
			name: "10^10-1 wei is 0 alpha",
			args: args{wei: bigIntFromString(t, "9999999999")},
			want: 0,
		},
		{
			name: "10^10 wei is 1 alpha",
			args: args{wei: bigIntFromString(t, "10000000000")},
			want: 1,
		},
		{
			name: "10^10+1 wei is 1 alpha",
			args: args{wei: bigIntFromString(t, "10000000001")},
			want: 1,
		},
		{
			name: "(2*10^10)-1 wei is 1 alpha",
			args: args{wei: bigIntFromString(t, "19999999999")},
			want: 1,
		},
		{
			name: "2*10^10 wei is 1 alpha",
			args: args{wei: bigIntFromString(t, "20000000000")},
			want: 2,
		},
		{
			name: "12*10^10 wei is 1 alpha",
			args: args{wei: bigIntFromString(t, "120000000000")},
			want: 12,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ConvertBalanceToAlpha(tt.args.wei); got != tt.want {
				t.Errorf("ConvertBalanceToAlpha() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNew(t *testing.T) {
	dir := t.TempDir()
	am, err := account.NewManager(dir, "", true)
	require.NoError(t, err)
	// no partition id
	w, err := New(0, ":23435", am)
	require.ErrorContains(t, err, "partition id is unassigned")
	require.Nil(t, w)
	// no URL
	w, err = New(1, "", am)
	require.ErrorContains(t, err, "rest url is empty")
	require.Nil(t, w)
	// no account manager
	w, err = New(1, ":23435", nil)
	require.ErrorContains(t, err, "account manager is nil")
	require.Nil(t, w)
	w, err = New(1, ":23435", am)
	require.NoError(t, err)
	require.NotNil(t, w)
	w.Shutdown()
}

func TestWallet_EvmCall(t *testing.T) {
	w, clientMock := createTestWallet(t)
	require.NotNil(t, w)
	require.NotNil(t, clientMock)
	ctx := context.Background()
	attrs := &evm.CallEVMRequest{}
	res, err := w.EvmCall(ctx, 1, attrs)
	require.ErrorContains(t, err, "account key read failed: account does not exist")
	require.Nil(t, res)
	// add account
	require.NoError(t, w.am.CreateKeys(testMnemonic))
	res, err = w.EvmCall(ctx, 1, attrs)
	require.NoError(t, err)
	require.NotNil(t, res)
	// 0 is invalid account index
	res, err = w.EvmCall(ctx, 0, attrs)
	require.ErrorContains(t, err, "invalid account number: 0")
	require.Nil(t, res)
	// simulate error from client
	clientMock.SimulateErr = fmt.Errorf("something bad happened")
	res, err = w.EvmCall(ctx, 1, attrs)
	require.ErrorContains(t, err, "something bad happened")
	require.Nil(t, res)
}

func TestWallet_GetBalance(t *testing.T) {
	w, clientMock := createTestWallet(t)
	require.NotNil(t, w)
	require.NotNil(t, clientMock)
	ctx := context.Background()
	res, err := w.GetBalance(ctx, 1)
	require.ErrorContains(t, err, "account key read failed: account does not exist")
	require.Nil(t, res)
	// add account
	require.NoError(t, w.am.CreateKeys(testMnemonic))
	res, err = w.GetBalance(ctx, 1)
	require.NoError(t, err)
	require.NotNil(t, res)
	// 0 is invalid account index
	res, err = w.GetBalance(ctx, 0)
	require.ErrorContains(t, err, "invalid account number: 0")
	require.Nil(t, res)
	// simulate error from client
	clientMock.SimulateErr = fmt.Errorf("something bad happened")
	res, err = w.GetBalance(ctx, 1)
	require.ErrorContains(t, err, "something bad happened")
	require.Nil(t, res)
}

func TestWallet_SendEvmTx(t *testing.T) {
	w, clientMock := createTestWallet(t)
	require.NotNil(t, w)
	require.NotNil(t, clientMock)
	ctx := context.Background()
	attrs := &evm.TxAttributes{}
	res, err := w.SendEvmTx(ctx, 1, attrs)
	require.ErrorContains(t, err, "account key read failed: account does not exist")
	require.Nil(t, res)
	// add account
	require.NoError(t, w.am.CreateKeys(testMnemonic))
	res, err = w.SendEvmTx(ctx, 1, attrs)
	require.NoError(t, err)
	require.NotNil(t, res)
	// 0 is invalid account index
	res, err = w.SendEvmTx(ctx, 0, attrs)
	require.ErrorContains(t, err, "invalid account number: 0")
	require.Nil(t, res)
	// simulate error from client
	clientMock.SimulateErr = fmt.Errorf("something bad happened")
	res, err = w.SendEvmTx(ctx, 1, attrs)
	require.ErrorContains(t, err, "something bad happened")
	require.Nil(t, res)
	// simulate no fee credit
	clientMock.SimulateErr = nil
	clientMock.noFcb = true
	res, err = w.SendEvmTx(ctx, 1, attrs)
	require.ErrorContains(t, err, "no fee credit in evm wallet")
	require.Nil(t, res)
	// simulate insufficient fee credit
	clientMock.noFcb = false
	clientMock.gasPrice = "100000000000"
	attrs.Gas = 1
	res, err = w.SendEvmTx(ctx, 1, attrs)
	require.ErrorContains(t, err, "insufficient fee credit balance for transaction")
	require.Nil(t, res)
}
