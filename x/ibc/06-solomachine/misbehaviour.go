package solomachine

import (
	"bytes"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	clientexported "github.com/cosmos/cosmos-sdk/x/ibc/02-client/exported"
	clienttypes "github.com/cosmos/cosmos-sdk/x/ibc/02-client/types"
	"github.com/cosmos/cosmos-sdk/x/ibc/06-solomachine/types"
)

// CheckMisbehaviourAndUpdateState determines whether or not the currently registered
// public key signed over two different messages with the same sequence. If this is true
// the client state is updated to a frozen status.
func CheckMisbehaviourAndUpdateState(
	clientState clientexported.ClientState,
	_ clientexported.ConsensusState,
	misbehaviour clientexported.Misbehaviour,
) (clientexported.ClientState, error) {

	// cast the interface to specific types before checking for misbehaviour
	smClientState, ok := clientState.(types.ClientState)
	if !ok {
		return nil, sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "client state type %T is not solo machine", clientState)
	}

	if smClientState.IsFrozen() {
		return nil, sdkerrors.Wrapf(clienttypes.ErrClientFrozen, "client is already frozen")
	}

	evidence, ok := misbehaviour.(types.Evidence)
	if !ok {
		return nil, sdkerrors.Wrapf(clienttypes.ErrInvalidClientType, "evidence type %T is not solo machine", misbehaviour)
	}

	if err := checkMisbehaviour(smClientState, evidence); err != nil {
		return nil, err
	}

	smClientState.Frozen = true
	return smClientState, nil
}

// checkMisbehaviour checks if the currently registered public key has signed
// over two different messages at the same sequence.
func checkMisbehaviour(clientState types.ClientState, evidence types.Evidence) error {
	pubKey := clientState.ConsensusState.PubKey

	// assert that provided signature data are different
	if bytes.Equal(evidence.SignatureOne.Data, evidence.SignatureTwo.Data) {
		return sdkerrors.Wrap(clienttypes.ErrInvalidEvidence, "evidence signatures have identical data messages")
	}

	data := append(sdk.Uint64ToBigEndian(evidence.Sequence), evidence.SignatureOne.Data...)

	// check first signature
	if err := types.CheckSignature(pubKey, data, evidence.SignatureOne.Signature); err != nil {
		return sdkerrors.Wrap(err, "evidence signature one failed to be verified")
	}

	data = append(sdk.Uint64ToBigEndian(evidence.Sequence), evidence.SignatureTwo.Data...)

	// check second signature
	if err := types.CheckSignature(pubKey, data, evidence.SignatureTwo.Signature); err != nil {
		return sdkerrors.Wrap(err, "evidence signature two failed to be verified")
	}

	return nil
}
