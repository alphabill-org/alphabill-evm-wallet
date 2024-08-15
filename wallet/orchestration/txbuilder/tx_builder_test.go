package txbuilder

import (
	"testing"

	"github.com/alphabill-org/alphabill-go-base/txsystem/orchestration"
	"github.com/stretchr/testify/require"

	test "github.com/alphabill-org/alphabill-wallet/internal/testutils"
	"github.com/alphabill-org/alphabill-wallet/wallet/account"
)

const testMnemonic = "dinosaur simple verify deliver bless ridge monkey design venue six problem lucky"

var (
	accountKey, _ = account.NewKeys(testMnemonic)
)

func TestNewAddVarTx_OK(t *testing.T) {
	timeout := uint64(10)
	systemID := orchestration.DefaultSystemID
	unitID := orchestration.NewVarID(nil, test.RandomBytes(32))
	_var := orchestration.ValidatorAssignmentRecord{}

	tx, err := NewAddVarTx(_var, systemID, unitID, timeout, accountKey.AccountKey, 3)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.EqualValues(t, systemID, tx.SystemID())
	require.EqualValues(t, unitID, tx.UnitID())
	require.EqualValues(t, timeout, tx.Timeout())
	require.EqualValues(t, 3, tx.GetClientMaxTxFee())
	require.Nil(t, tx.GetClientFeeCreditRecordID())
	require.Nil(t, tx.FeeProof)
	require.NotNil(t, tx.OwnerProof)

	require.EqualValues(t, orchestration.PayloadTypeAddVAR, tx.PayloadType())
	var attr *orchestration.AddVarAttributes
	err = tx.UnmarshalAttributes(&attr)
	require.NoError(t, err)
	require.Equal(t, _var, attr.Var)
}

func TestNewAddVarTxUnsigned_OK(t *testing.T) {
	timeout := uint64(10)
	systemID := orchestration.DefaultSystemID
	unitID := orchestration.NewVarID(nil, test.RandomBytes(32))
	_var := orchestration.ValidatorAssignmentRecord{}

	tx, err := NewAddVarTx(_var, systemID, unitID, timeout, nil, 5)
	require.NoError(t, err)
	require.NotNil(t, tx)
	require.EqualValues(t, systemID, tx.SystemID())
	require.EqualValues(t, unitID, tx.UnitID())
	require.EqualValues(t, timeout, tx.Timeout())
	require.EqualValues(t, 5, tx.GetClientMaxTxFee())
	require.Nil(t, tx.GetClientFeeCreditRecordID())
	require.Nil(t, tx.FeeProof)
	require.Nil(t, tx.OwnerProof)

	require.EqualValues(t, orchestration.PayloadTypeAddVAR, tx.PayloadType())
	var attr *orchestration.AddVarAttributes
	err = tx.UnmarshalAttributes(&attr)
	require.NoError(t, err)
	require.Equal(t, _var, attr.Var)
}
