package keeper

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	utilsort "github.com/ingenuity-build/quicksilver/utils/sort"
	cmtypes "github.com/ingenuity-build/quicksilver/x/claimsmanager/types"
)

// CalcUserHoldingsAllocations calculates allocations per user for a given zone, based upon claims submitted and zone.
func (k Keeper) CalcUserHoldingsAllocations(ctx sdk.Context, zone cmtypes.CustomZone) ([]cmtypes.UserAllocation, math.Int) {
	k.Logger(ctx).Info("CalcUserHoldingsAllocations", "zone", zone.ChainID, "allocations", zone.HoldingsAllocation)

	userAllocations := make([]cmtypes.UserAllocation, 0)
	supply := k.bankKeeper.GetSupply(ctx, zone.LocalDenom)

	if zone.HoldingsAllocation == 0 || supply.Amount.IsZero() {
		k.Logger(ctx).Info("holdings allocation is zero, nothing to allocate")
		return userAllocations, math.NewIntFromUint64(zone.HoldingsAllocation)
	}

	// calculate user totals and zone total (held assets)
	zoneAmount := math.ZeroInt()
	userAmountsMap := make(map[string]math.Int)

	k.IterateClaims(ctx, zone.ChainID, func(_ int64, claim cmtypes.Claim) (stop bool) {
		amount := math.NewIntFromUint64(claim.Amount)
		k.Logger(ctx).Info(
			"claim",
			"type", cmtypes.ClaimType_name[int32(claim.Module)],
			"user", claim.UserAddress,
			"zone", claim.ChainId,
			"amount", amount,
		)

		if _, exists := userAmountsMap[claim.UserAddress]; !exists {
			userAmountsMap[claim.UserAddress] = math.ZeroInt()
		}

		userAmountsMap[claim.UserAddress] = userAmountsMap[claim.UserAddress].Add(amount)

		// total zone assets held remotely
		zoneAmount = zoneAmount.Add(amount)

		return false
	})

	if zoneAmount.IsZero() {
		k.Logger(ctx).Info("zero claims for zone", "zone", zone.ChainID)
		return userAllocations, math.NewIntFromUint64(zone.HoldingsAllocation)
	}

	zoneAllocation := math.NewIntFromUint64(zone.HoldingsAllocation)
	tokensPerAsset := sdk.NewDecFromInt(zoneAllocation).Quo(sdk.NewDecFromInt(supply.Amount))

	k.Logger(ctx).Info("tokens per asset", "zone", zone.ChainID, "tpa", tokensPerAsset)

	for _, address := range utilsort.Keys(userAmountsMap) {
		amount := userAmountsMap[address]
		userAllocation := sdk.NewDecFromInt(amount).Mul(tokensPerAsset).TruncateInt()
		allocation := cmtypes.UserAllocation{
			Address: address,
			Amount:  userAllocation,
		}
		userAllocations = append(userAllocations, allocation)
		zoneAllocation = zoneAllocation.Sub(userAllocation)
		if zoneAllocation.LT(sdk.ZeroInt()) {
			panic("user allocation overflow")
		}
	}

	return userAllocations, zoneAllocation
}