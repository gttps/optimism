// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

/// @title DepositorAllowlist
/// @notice Selects the addresses allowed to deposit through the OptimismPortal by L1 chain ID.
///         Only mainnet and Sepolia are restricted; every other chain admits any depositor so
///         that devnets and test suites keep working.
library DepositorAllowlist {
    /// @notice Returns whether deposits on the current L1 chain are restricted to the allowlist.
    function isRestricted() internal view returns (bool) {
        return block.chainid == 1 || block.chainid == 11155111;
    }

    /// @notice Returns the addresses allowed to make deposits on a restricted L1 chain.
    /// @return depositors_ The allowed depositors.
    function allowedDepositors() internal view returns (address[5] memory depositors_) {
        // TODO: replace placeholder anvil test accounts with the real depositors.
        if (block.chainid == 1) {
            depositors_ = [
                0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266,
                0x70997970C51812dc3A010C7d01b50e0d17dc79C8,
                0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC,
                0x90F79bf6EB2c4f870365E785982E1f101E93b906,
                0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65
            ];
        } else if (block.chainid == 11155111) {
            depositors_ = [
                0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266,
                0x70997970C51812dc3A010C7d01b50e0d17dc79C8,
                0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC,
                0x90F79bf6EB2c4f870365E785982E1f101E93b906,
                0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65
            ];
        }
    }

    /// @notice Checks if an address is allowed to make deposits.
    /// @param _depositor The address to check.
    /// @return bool True if the address is an allowed depositor.
    function isAllowed(address _depositor) internal view returns (bool) {
        if (!isRestricted()) return true;
        address[5] memory depositors = allowedDepositors();
        for (uint256 i = 0; i < depositors.length; i++) {
            if (depositors[i] == _depositor) return true;
        }
        return false;
    }
}
