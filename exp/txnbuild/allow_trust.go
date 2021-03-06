package txnbuild

import (
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/xdr"
)

// AllowTrust represents the Stellar allow trust operation. See
// https://www.stellar.org/developers/guides/concepts/list-of-operations.html
type AllowTrust struct {
	Trustor   string
	Type      Asset
	Authorize bool
}

// BuildXDR for AllowTrust returns a fully configured XDR Operation.
func (at *AllowTrust) BuildXDR() (xdr.Operation, error) {
	var xdrOp xdr.AllowTrustOp

	// Set XDR address associated with the trustline
	err := xdrOp.Trustor.SetAddress(at.Trustor)
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Failed to set trustor address")
	}

	// Validate this is an issued asset
	if at.Type.IsNative() {
		return xdr.Operation{}, errors.New("Trustline doesn't exist for a native (XLM) asset")
	}

	// AllowTrust has a special asset type - map to it
	xdrAsset, err := at.Type.ToXDR()
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Can't convert asset for trustline to XDR")
	}

	xdrOp.Asset, err = xdrAsset.ToAllowTrustOpAsset(at.Type.GetCode())
	if err != nil {
		return xdr.Operation{}, errors.Wrap(err, "Can't convert asset for trustline to allow trust asset type")
	}

	// Set XDR auth flag
	xdrOp.Authorize = at.Authorize

	opType := xdr.OperationTypeAllowTrust
	body, err := xdr.NewOperationBody(opType, xdrOp)

	return xdr.Operation{Body: body}, errors.Wrap(err, "Failed to build XDR OperationBody")
}
