package e2e

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"github.com/0xPolygon/polygon-edge/bls"
	"github.com/cometbft/cometbft/crypto/tmhash"
	"github.com/cometbft/cometbft/votepool"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	govTypesV1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	stakeTypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/mocachain/moca-go-sdk/e2e/basesuite"
	"github.com/mocachain/moca-go-sdk/types"
	gnfdsdktypes "github.com/mocachain/moca/v2/sdk/types"
	"github.com/stretchr/testify/suite"
)

type ValidatorTestSuite struct {
	basesuite.BaseSuite
}

func (s *ValidatorTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func (s *ValidatorTestSuite) Test_Validator_Operations() {
	newValAccount, _, _ := types.NewAccount("new_validator")
	newRelayerAccount, _, _ := types.NewAccount("new_validator_relayer")
	newChallengerAccount, _, _ := types.NewAccount("new_validator_challenger")
	newValEd25519PubKey := hex.EncodeToString(ed25519.GenPrivKey().PubKey().Bytes())
	newValidatorAddr := newValAccount.GetAddress()
	s.T().Logf("new validator address is %s", newValidatorAddr.String())

	// transfer some funds to the new validator
	validator0Account := s.DefaultAccount
	txHash, err := s.Client.Transfer(s.ClientContext, newValidatorAddr.String(), math.NewIntWithDecimal(1000, gnfdsdktypes.DecimalMOCA), gnfdsdktypes.TxOption{})
	s.Require().NoError(err)

	_, err = s.Client.WaitForTx(s.ClientContext, txHash)
	s.Require().NoError(err)

	// newVal grant gov module account for proposal execution
	s.Client.SetDefaultAccount(newValAccount)
	delegationAmount := math.NewIntWithDecimal(1, 18)

	txHash, err = s.Client.GrantDelegationForValidator(s.ClientContext, delegationAmount, gnfdsdktypes.TxOption{})
	s.Require().NoError(err)
	s.T().Logf("grant auth txHash %s", txHash)

	_, err = s.Client.WaitForTx(s.ClientContext, txHash)
	s.Require().NoError(err)

	description := stakeTypes.Description{Moniker: "test_new_validator"}
	rates := stakeTypes.CommissionRates{
		Rate:          math.LegacyNewDecWithPrec(5, 2),
		MaxRate:       math.LegacyOneDec(),
		MaxChangeRate: math.LegacyOneDec(),
	}

	blsSecretKey, _ := bls.GenerateBlsKey()
	blsPubKey := blsSecretKey.PublicKey().Marshal()
	blsProofBzBts, _ := blsSecretKey.Sign(tmhash.Sum(blsPubKey), votepool.DST)
	blsProofBz, _ := blsProofBzBts.Marshal()

	proposalID, txHash, err := s.Client.CreateValidator(s.ClientContext,
		description,
		rates,
		delegationAmount,
		newValidatorAddr.String(),
		newValEd25519PubKey,
		newValAccount.GetAddress().String(),
		newRelayerAccount.GetAddress().String(),
		newChallengerAccount.GetAddress().String(),
		hex.EncodeToString(blsPubKey),
		hex.EncodeToString(blsProofBz),
		math.NewIntWithDecimal(governanceMinDeposit, gnfdsdktypes.DecimalMOCA),
		"create new validator",
		"create new validator",
		"",
		gnfdsdktypes.TxOption{},
	)
	s.Require().NoError(err)
	_, err = s.Client.WaitForTx(s.ClientContext, txHash)
	s.Require().NoError(err)

	s.Client.SetDefaultAccount(validator0Account)
	_, err = s.Client.VoteProposal(s.ClientContext, proposalID, govTypesV1.OptionYes, types.VoteProposalOptions{})
	s.Require().NoError(err)

	for {
		p, err := s.Client.GetProposal(s.ClientContext, proposalID)
		s.Require().NoError(err)
		s.Require().NotNil(p)
		s.T().Logf("Proposal: %d, %s, votingEnd=%v, tally=%+v", p.Id, p.Status, p.VotingEndTime, p.FinalTallyResult)
		if p.Status == govTypesV1.ProposalStatus_PROPOSAL_STATUS_PASSED {
			break
		} else if p.Status == govTypesV1.ProposalStatus_PROPOSAL_STATUS_FAILED {
			s.Require().True(false)
		}
		time.Sleep(1 * time.Second)
	}
	err = s.Client.WaitForNBlocks(s.ClientContext, 1)
	s.Require().NoError(err)

	// query the new validator, status is BONDED
	validators, err := s.Client.ListValidators(context.Background(), "BOND_STATUS_BONDED")
	s.Require().NoError(err)
	isPresent := false
	for _, v := range validators.Validators {
		if v.SelfDelAddress == newValidatorAddr.String() {
			isPresent = true
		}
	}
	s.Require().True(isPresent)

	// unbound
	s.Client.SetDefaultAccount(newValAccount)
	txHash, err = s.Client.Undelegate(s.ClientContext, newValidatorAddr.String(), delegationAmount, gnfdsdktypes.TxOption{})
	s.Require().NoError(err)
	_, err = s.Client.WaitForTx(s.ClientContext, txHash)
	s.Require().NoError(err)
	err = s.Client.WaitForNBlocks(s.ClientContext, 3)
	s.Require().NoError(err)

	// query the new validator status is UNBONDING
	validators, err = s.Client.ListValidators(context.Background(), "BOND_STATUS_UNBONDING")
	s.Require().NoError(err)
	isPresent = true
	for _, v := range validators.Validators {
		s.T().Log(v)
		if v.SelfDelAddress == newValidatorAddr.String() {
			isPresent = false
		}
	}
	s.Require().False(isPresent)

	// delegate validator
	txHash, err = s.Client.DelegateValidator(s.ClientContext, newValidatorAddr.String(), delegationAmount, gnfdsdktypes.TxOption{})
	s.Require().NoError(err)
	_, err = s.Client.WaitForTx(s.ClientContext, txHash)
	s.Require().NoError(err)

	// unjail the new validator
	txHash, err = s.Client.UnJailValidator(s.ClientContext, gnfdsdktypes.TxOption{})
	s.Require().NoError(err)
	_, err = s.Client.WaitForTx(s.ClientContext, txHash)
	s.Require().NoError(err)

	// query the new validator, status is BONDED again
	validators, err = s.Client.ListValidators(context.Background(), "BOND_STATUS_BONDED")
	s.Require().NoError(err)
	isPresent = false
	for _, v := range validators.Validators {
		if v.SelfDelAddress == newValidatorAddr.String() {
			isPresent = true
		}
	}
	s.Require().True(isPresent)

	// create a proposal to impeach the new Validator
	s.Client.SetDefaultAccount(validator0Account)
	proposalID, txHash, err = s.Client.ImpeachValidator(
		context.Background(),
		newValidatorAddr.String(),
		math.NewIntWithDecimal(governanceMinDeposit, gnfdsdktypes.DecimalMOCA),
		"title",
		"summary ",
		"meta",
		gnfdsdktypes.TxOption{},
	)
	s.Require().NoError(err)
	_, err = s.Client.WaitForTx(s.ClientContext, txHash)
	s.Require().NoError(err)

	// vote for the proposal
	txHash, err = s.Client.VoteProposal(s.ClientContext, proposalID, govTypesV1.OptionYes, types.VoteProposalOptions{})
	s.Require().NoError(err)
	_, err = s.Client.WaitForTx(s.ClientContext, txHash)
	s.Require().NoError(err)
	err = s.Client.WaitForNBlocks(s.ClientContext, 10)
	s.Require().NoError(err)
	validators, err = s.Client.ListValidators(context.Background(), "BOND_STATUS_BONDED")
	s.Require().NoError(err)
	isPresent = false
	for _, v := range validators.Validators {
		if v.SelfDelAddress == newValidatorAddr.String() {
			isPresent = true
		}
	}
	s.Require().False(isPresent)
}

func TestValidatorTestSuite(t *testing.T) {
	suite.Run(t, new(ValidatorTestSuite))
}
