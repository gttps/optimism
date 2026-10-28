// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title DepositorAllowlist
/// @notice Selects the addresses allowed to deposit through the OptimismPortal by L1 chain ID.
///         A chain without a list admits any depositor, so devnets and test suites keep working.
library DepositorAllowlist {
    /// @notice Returns the addresses allowed to make deposits on the current L1 chain.
    /// @return depositors_ The allowed depositors, empty when the chain is unrestricted.
    function allowedDepositors() internal view returns (address[] memory depositors_) {
        if (block.chainid == 11155111) {
            depositors_ = new address[](4);
            depositors_[0] = 0xE5327a59e33CAB54AcFFFBEba6bb167AaEDCD953;
            depositors_[1] = 0x2EA11DcDAEE45D82aAe2Ad6e7255f8d5bb70AbFF;
            depositors_[2] = 0x7d9F14FCd14fdf7A47BC2657949e087aA24f62bA;
            depositors_[3] = 0xcE38EA73Aac620E9aF3d45b53088B9E7294C5a2C;
        }
        // TODO: add the mainnet depositors. Until then mainnet admits any depositor.
    }

    /// @notice Checks if an address is allowed to make deposits.
    /// @param _depositor The address to check.
    /// @return bool True if the address is an allowed depositor.
    function isAllowed(address _depositor) internal view returns (bool) {
        address[] memory depositors = allowedDepositors();
        if (depositors.length == 0) return true;
        for (uint256 i = 0; i < depositors.length; i++) {
            if (depositors[i] == _depositor) return true;
        }
        return false;
    }
}
