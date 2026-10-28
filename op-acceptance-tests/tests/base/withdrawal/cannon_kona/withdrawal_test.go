package withdrawal

import (
	"testing"

	"github.com/ethereum-optimism/optimism/op-acceptance-tests/tests/base/withdrawal"
	gameTypes "github.com/ethereum-optimism/optimism/op-challenger/game/types"
)

func TestWithdrawal_CannonKona(gt *testing.T) {
	gt.Skip("Skipping: the portal depositor allowlist rejects deposits from the devstack test accounts")
	withdrawal.TestWithdrawal(gt, gameTypes.CannonKonaGameType)
}
