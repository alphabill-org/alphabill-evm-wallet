package orchestration

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/alphabill-org/alphabill-go-base/txsystem/orchestration"
	"github.com/alphabill-org/alphabill-go-base/types"
	"github.com/alphabill-org/alphabill-go-base/util"

	clitypes "github.com/alphabill-org/alphabill-wallet/cli/alphabill/cmd/types"
	cliaccount "github.com/alphabill-org/alphabill-wallet/cli/alphabill/cmd/util/account"
	"github.com/alphabill-org/alphabill-wallet/cli/alphabill/cmd/wallet/args"
	"github.com/alphabill-org/alphabill-wallet/client"
	"github.com/alphabill-org/alphabill-wallet/wallet/orchestration/txbuilder"
)

const (
	cmdFlagPartitionID = "partition-id"
	cmdFlagShardID     = "shard-id"
	cmdFlagVarFilePath = "var-file"

	txTimeoutBlockCount = 10
)

func NewCmd(config *clitypes.WalletConfig) *cobra.Command {
	orchestrationConfig := &clitypes.OrchestrationConfig{WalletConfig: config}
	cmd := &cobra.Command{
		Use:   "orchestration",
		Short: "tools to manage orchestration partition",
	}
	cmd.AddCommand(addVarCmd(orchestrationConfig))
	cmd.PersistentFlags().StringVarP(&orchestrationConfig.RpcUrl, args.RpcUrl, "r", args.DefaultOrchestrationRpcUrl, "rpc node url")
	cmd.PersistentFlags().Uint64VarP(&orchestrationConfig.Key, args.KeyCmdName, "k", 1, "account number of the proof-of-authority key")
	return cmd
}

func addVarCmd(orchestrationConfig *clitypes.OrchestrationConfig) *cobra.Command {
	config := &clitypes.AddVarCmdConfig{OrchestrationConfig: orchestrationConfig}
	var shardID string
	cmd := &cobra.Command{
		Use:   "add-var",
		Short: "adds validator assignment record",
		RunE: func(cmd *cobra.Command, args []string) error {
			if shardID != "" {
				if err := config.ShardID.UnmarshalText([]byte(shardID)); err != nil {
					return fmt.Errorf("parsing %q value: %w", cmdFlagShardID, err)
				}

			}
			return execAddVarCmd(cmd, config)
		},
	}
	cmd.Flags().Uint32Var(&config.PartitionID, cmdFlagPartitionID, 0, "partition identifier of the managed partition")
	_ = cmd.MarkFlagRequired(cmdFlagPartitionID)

	cmd.Flags().StringVar(&shardID, cmdFlagShardID, "", "shard id (nil if only one shard) of the managed shard")

	cmd.Flags().StringVar(&config.VarFilePath, cmdFlagVarFilePath, "", "path to validator assignment record json file")
	_ = cmd.MarkFlagRequired(cmdFlagVarFilePath)
	args.AddMaxFeeFlag(cmd, cmd.Flags())

	return cmd
}

func execAddVarCmd(cmd *cobra.Command, config *clitypes.AddVarCmdConfig) error {
	// load account manager (it is expected that accounts.db exists in wallet home dir)
	walletConfig := config.OrchestrationConfig.WalletConfig
	am, err := cliaccount.LoadExistingAccountManager(walletConfig)
	if err != nil {
		return fmt.Errorf("failed to load account manager: %w", err)
	}
	ac, err := am.GetAccountKey(config.OrchestrationConfig.Key - 1)
	if err != nil {
		return fmt.Errorf("failed to load account key: %w", err)
	}

	// create rpc client
	rpcUrl := args.BuildRpcUrl(config.OrchestrationConfig.RpcUrl)
	orcClient, err := client.NewOrchestrationPartitionClient(cmd.Context(), rpcUrl)
	if err != nil {
		return fmt.Errorf("failed to create rpc client: %w", err)
	}

	// load var file
	varFile, err := util.ReadJsonFile(config.VarFilePath, &orchestration.ValidatorAssignmentRecord{})
	if err != nil {
		return fmt.Errorf("failed to load var file: %w", err)
	}

	// create 'addVar' tx
	pdr, err := orcClient.PartitionDescription(cmd.Context())
	if err != nil {
		return fmt.Errorf("loading Partition Description: %w", err)
	}
	unitID, err := pdr.ComposeUnitID(config.ShardID, orchestration.VarUnitType, orchestration.PrndSh(types.PartitionID(config.PartitionID), config.ShardID))
	if err != nil {
		return fmt.Errorf("creating unit id: %w", err)
	}

	roundInfo, err := orcClient.GetRoundInfo(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to fetch round info: %w", err)
	}
	timeout := roundInfo.RoundNumber + txTimeoutBlockCount
	maxFee, err := args.ParseMaxFeeFlag(cmd)
	if err != nil {
		return err
	}
	txo, err := txbuilder.NewAddVarTx(*varFile, pdr.NetworkID, pdr.PartitionID, unitID, timeout, maxFee, ac)
	if err != nil {
		return fmt.Errorf("failed to create 'addVar' tx: %w", err)
	}

	// send 'addVar' tx
	_, err = orcClient.ConfirmTransaction(cmd.Context(), txo, walletConfig.Base.Logger)
	if err != nil {
		return fmt.Errorf("failed to send tx: %w", err)
	}

	walletConfig.Base.ConsoleWriter.Println("Validator Assignment Record added successfully.")
	return nil
}
