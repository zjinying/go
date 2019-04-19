package txnbuild

import (
	"errors"
	"time"
)

// TimeoutInfinite allows an indefinite upper bound to be set for Transaction.MaxTime. This should not
// normally be needed.
const TimeoutInfinite = int64(0)

// Timebounds represents the time window during which a Stellar transaction is considered valid.
//
// MinTime and MaxTime represent Stellar timebounds - a window of time over which the Transaction will be
// considered valid. In general, all Transactions benefit from setting an upper timebound, because once submitted,
// the status of a pending Transaction may remain unresolved for a long time if the network is congested.
// With an upper timebound, the submitter has a guaranteed time at which the Transaction is known to have either
// succeeded or failed.
//
// Create a Timebounds struct using one of SetTimebounds(), SetTimeout(), or SetNoTimeout().
type Timebounds struct {
	MinTime  int64
	MaxTime  int64
	wasBuilt bool
}

// Validate for Timebounds sanity-checks the configured Timebound limits, and confirms the object was built
// using a factory method. This is done to ensure that default Timebound structs (which have no limits) are not
// valid - you must explicitly specifiy the Timebound you require.
func (tb *Timebounds) Validate() error {
	if !tb.wasBuilt {
		return errors.New("timebounds must be constructed using SetTimebounds(), SetTimeout(), or SetNoTimeout()")
	}
	if tb.MinTime < 0 {
		return errors.New("invalid timebound: minTime cannot be negative")
	}

	if tb.MaxTime < 0 {
		return errors.New("invalid timebound: maxTime cannot be negative")
	}

	if tb.MaxTime != TimeoutInfinite {
		if tb.MaxTime < tb.MinTime {
			return errors.New("invalid timebound: maxTime < minTime")
		}
	}

	return nil
}

// SetTimebounds is a factory method that constructs a Timebounds object from a min and max time.
// A Transaction cannot be built unless a Timebounds object is provided through a factory method.
func SetTimebounds(minTime, maxTime int64) Timebounds {
	return Timebounds{minTime, maxTime, true}
}

// SetTimeout is a factory method that sets the minTime, and sets the MaxTime to be the duration in seconds
// in the future specified by 'timeout'.
// A Transaction cannot be built unless a Timebounds object is provided through a factory method.
// This method uses the provided system time - make sure it is accurate.
func SetTimeout(minTime, timeout int64) Timebounds {
	return Timebounds{minTime, time.Now().UTC().Unix() + timeout, true}
}

// SetNoTimeout is a factory method that sets the minTime, and sets the MaxTime to a value representing an indefinite
// upper time bound. This is rarely needed, but is helpful for certain smart contracts, and for deterministic testing.
// A Transaction cannot be built unless a Timebounds object is provided through a factory method.
func SetNoTimeout(minTime int64) Timebounds {
	return Timebounds{minTime, TimeoutInfinite, true}
}
