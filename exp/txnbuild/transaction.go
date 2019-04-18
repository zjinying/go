/*
Package txnbuild implements transactions and operations on the Stellar network.
TODO: More explanation + links here
*/
package txnbuild

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// TimeoutInfinite allows an indefinite upper bound to be set for Transaction.MaxTime. This should not
// normally be needed.
const TimeoutInfinite = int64(0)

// Account represents the aspects of a Stellar account necessary to construct transactions.
type Account interface {
	GetAccountID() string
	IncrementSequenceNumber() (xdr.SequenceNumber, error)
}

// Transaction represents a Stellar Transaction.
type Transaction struct {
	SourceAccount  Account
	Operations     []Operation
	xdrTransaction xdr.Transaction
	BaseFee        uint32
	Memo           Memo
	MinTime        int64
	MaxTime        int64
	Network        string
	xdrEnvelope    *xdr.TransactionEnvelope
}

// Hash provides a signable object representing the Transaction on the specified network.
func (tx *Transaction) Hash() ([32]byte, error) {
	return network.HashTransaction(&tx.xdrTransaction, tx.Network)
}

// MarshalBinary returns the binary XDR representation of the Transaction.
func (tx *Transaction) MarshalBinary() ([]byte, error) {
	var txBytes bytes.Buffer
	_, err := xdr.Marshal(&txBytes, tx.xdrEnvelope)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to marshal XDR")
	}

	return txBytes.Bytes(), nil
}

// Base64 returns the base 64 XDR representation of the Transaction.
func (tx *Transaction) Base64() (string, error) {
	bs, err := tx.MarshalBinary()
	if err != nil {
		return "", errors.Wrap(err, "Failed to get XDR bytestring")
	}

	return base64.StdEncoding.EncodeToString(bs), nil
}

// SetDefaultFee sets a sensible minimum default for the Transaction fee, if one has not
// already been set. It is a linear function of the number of Operations in the Transaction.
func (tx *Transaction) SetDefaultFee() {
	// TODO: Generalise to pull this from a client call
	var DefaultBaseFee uint32 = 100
	if tx.BaseFee == 0 {
		tx.BaseFee = DefaultBaseFee
	}
	if tx.xdrTransaction.Fee == 0 {
		tx.xdrTransaction.Fee = xdr.Uint32(int(tx.BaseFee) * len(tx.xdrTransaction.Operations))
	}
}

// SetTimeout sets the value of tx.MaxTime to be the duration in the future from now specified by 'timeout'.
//
// The value of tx.MinTime is not changed.
// A Transaction cannot be built unless tx.MaxTime is set, either via this method, or directly.
//
// tx.MinTime and tx.MaxTime represent Stellar timebounds - a window of time over which the Transaction will be
// considered valid. In general, all Transactions benefit from setting an upper timebound, because once submitted,
// the status of a pending Transaction may remain unresolved for a long time if the network is congested.
// With an upper timebound, the submitter has a guaranteed time at which the Transaction is known to have either
// succeeded or failed.
//
// This method uses the provided system time - make sure it is accurate.
//
// Rarely (e.g. for certain smart contracts), it is necessary to set an indefinite upper time bound. To do this,
// set tx.MaxTime = TimeoutInfinite, and do not call this method.
func (tx *Transaction) SetTimeout(timeout time.Duration) error {
	// Don't set the timeout if the max time is already set
	if tx.MaxTime != 0 {
		return errors.New("Transaction.MaxTime has already been set - setting timeout would overwrite it")
	}

	if timeout.Seconds() <= 0 {
		return errors.New("timeout cannot be negative")
	}

	maxTimeUnix := time.Now().UTC().Add(timeout).Unix()

	if maxTimeUnix < tx.MinTime {
		return fmt.Errorf("invalid timeout: provided timeout '%v' would produce Transaction.MaxTime < Transaction.MinTime", timeout)
	}

	tx.MaxTime = maxTimeUnix

	return nil
}

// Build for Transaction completely configures the Transaction. After calling Build,
// the Transaction is ready to be serialised or signed.
func (tx *Transaction) Build() error {
	// Set account ID in XDR
	// TODO: Validate provided key before going further
	tx.xdrTransaction.SourceAccount.SetAddress(tx.SourceAccount.GetAccountID())

	// TODO: Validate Seq Num is present in struct
	seqnum, err := tx.SourceAccount.IncrementSequenceNumber()
	if err != nil {
		return errors.Wrap(err, "Failed to parse sequence number")
	}
	tx.xdrTransaction.SeqNum = seqnum

	for _, op := range tx.Operations {
		xdrOperation, err := op.BuildXDR()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("Failed to build operation %T", op))
		}
		tx.xdrTransaction.Operations = append(tx.xdrTransaction.Operations, xdrOperation)
	}

	// TODO: Make setting the timebounds to 'something' mandatory
	// TODO: Only build if the maxTime has been set. Consider making TimeoutInfinite something other than 0 to
	// disambiguate
	// TODO: Add helper method to client to get time from server

	// Set the timebounds. Since they're optional, we don't bother if they weren't set.
	if tx.MinTime > 0 || tx.MaxTime > 0 {
		tx.xdrTransaction.TimeBounds = &xdr.TimeBounds{MinTime: xdr.Uint64(tx.MinTime), MaxTime: xdr.Uint64(tx.MaxTime)}
	}

	// Handle the memo, if one is present
	if tx.Memo != nil {
		xdrMemo, err := tx.Memo.ToXDR()
		if err != nil {
			return errors.Wrap(err, "Couldn't build memo XDR")
		}
		tx.xdrTransaction.Memo = xdrMemo
	}

	// Set a default fee, if it hasn't been set yet
	tx.SetDefaultFee()

	return nil
}

// Sign for Transaction signs a previously built transaction. A signed transaction may be
// submitted to the network.
func (tx *Transaction) Sign(kp *keypair.Full) error {
	// TODO: Only sign if Transaction has been previously built
	// TODO: Validate network set before sign
	// Initialise transaction envelope
	if tx.xdrEnvelope == nil {
		tx.xdrEnvelope = &xdr.TransactionEnvelope{}
		tx.xdrEnvelope.Tx = tx.xdrTransaction
	}

	// Hash the transaction
	hash, err := tx.Hash()
	if err != nil {
		return errors.Wrap(err, "Failed to hash transaction")
	}

	// Sign the hash
	// TODO: Allow multiple signers
	sig, err := kp.SignDecorated(hash[:])
	if err != nil {
		return errors.Wrap(err, "Failed to sign transaction")
	}

	// Append the signature to the envelope
	tx.xdrEnvelope.Signatures = append(tx.xdrEnvelope.Signatures, sig)

	return nil
}

// BuildSignEncode performs all the steps to produce a final transaction suitable
// for submitting to the network.
func (tx *Transaction) BuildSignEncode(keypair *keypair.Full) (string, error) {
	err := tx.Build()
	if err != nil {
		return "", errors.Wrap(err, "Couldn't build transaction")
	}

	err = tx.Sign(keypair)
	if err != nil {
		return "", errors.Wrap(err, "Couldn't sign transaction")
	}

	txeBase64, err := tx.Base64()
	if err != nil {
		return "", errors.Wrap(err, "Couldn't encode transaction")
	}

	return txeBase64, err
}
