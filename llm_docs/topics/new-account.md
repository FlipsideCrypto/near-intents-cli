# New NEAR Account — Zero to Funded

This topic covers the complete path from "I have no NEAR account" to "I have a funded, named NEAR account and can execute swaps." Follow it when a user has never used NEAR before.

## Step 1: Install near-cli-rs

```bash
# Primary method — reliable, cross-platform
npm install -g near-cli-rs

# Fallback — if npm is not available (requires Rust toolchain)
cargo install near-cli-rs

# Verify
near --version
```

⚠️ **`brew install near-cli-rs` does NOT work.** The Homebrew formula does not exist. Do not suggest it.

## Step 2: Understand NEAR's implicit account model

NEAR has a unique bootstrapping model:

- An **implicit account** is a 64-character hex string derived from a public key (e.g., `a3b4c5d6...` × 64 chars)
- It does **not** exist on-chain until it receives its first deposit of NEAR
- Once funded with any amount of NEAR, it springs into existence and can sign transactions
- You can generate the keypair locally first and fund it later — the account ID is just the hex public key

This matters because: you can generate an account ID, give it to someone (or use it as a swap recipient), fund it, and then use it — all without needing an existing NEAR account.

## Step 3: Generate an implicit account

```bash
near account create-account fund-later use-auto-generation save-to-keychain network-config mainnet
```

This outputs something like:
```
New account created: a3b4c5d6e7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c3d4e5f6a7b8c9d0e1f2a3b4
```

That 64-char hex string is your implicit account ID. The private key is saved to `~/.near-credentials/mainnet/<hex>.json`.

The account **does not exist on-chain yet**. Fund it first.

## Step 4: Fund the implicit account

Pick the path that matches what the user has:

### Option A: User has NEAR on an exchange
Withdraw from the exchange directly to the implicit account ID (the 64-char hex string). Any amount ≥ 1 NEAR works. The account activates automatically on receipt.

### Option B: User has crypto on EVM chains (ETH, USDC, etc.)
Use near-intents cross-chain swap to fund the implicit account:

```bash
# Swap USDC from Base → wNEAR, delivered to the implicit account
near-intents swap \
  --from USDC --from-chain base \
  --to wNEAR --to-chain near \
  --amount 10 \
  --recipient <64-char-implicit-account-id> \
  --refund-to <your-base-address>
```

⚠️ **Uncertainty:** It is not confirmed whether the NEAR Intents engine will deliver to an implicit account that does not yet exist on-chain. If this fails, fall back to Option A (fund via exchange first) to activate the account, then use intents for subsequent swaps.

After the swap completes, the implicit account holds wNEAR in the intents balance. Withdraw it:

```bash
near contract call-function as-transaction intents.near ft_withdraw \
  json-args '{"token":"wrap.near","amount":"<raw_amount>","receiver_id":"<implicit_account>"}' \
  prepaid-gas '100 Tgas' attached-deposit '1 yoctoNEAR' \
  sign-as <implicit_account> network-config mainnet sign-with-keychain send
```

Then unwrap wNEAR → NEAR if needed:

```bash
near contract call-function as-transaction wrap.near near_withdraw \
  json-args '{"amount":"<raw_amount>"}' \
  prepaid-gas '30 Tgas' attached-deposit '1 yoctoNEAR' \
  sign-as <implicit_account> network-config mainnet sign-with-keychain send
```

### Option C: Another NEAR user sends them NEAR
They can send to the implicit account ID directly — it will activate on receipt.

## Step 5: (Optional) Create a named account

Once the implicit account has NEAR, create a human-readable named account:

```bash
near account create-account fund-myself <username>.near '0.1 NEAR' \
  autogenerate-new-keypair save-to-keychain \
  sign-as <implicit_account_id> \
  network-config mainnet sign-with-keychain send
```

- Replace `<username>` with the desired name (e.g., `alice`, `fug603`)
- The `0.1 NEAR` covers the on-chain storage cost for the account
- A new keypair is generated and saved to `~/.near-credentials/mainnet/<username>.near.json`
- The implicit account pays for the creation

After this, `<username>.near` is a fully functional NEAR account with its own keys.

## Step 6: Verify the account

```bash
near account view-account-summary <account_id> network-config mainnet now
```

Should show balance, storage usage, and access keys.

## Step 7: Configure portfolio tracking

```bash
portfolio setup --add --chain near --address <account_id>
```

## Reserve budget

Always keep at least 0.5 NEAR in the account for gas and storage. On top of that:
- Named account creation: ~0.1 NEAR
- Storage deposit per new token: ~0.0125 NEAR
- Per transaction: ~0.001 NEAR gas

## Gotchas

| Gotcha | Detail |
|--------|--------|
| `brew install near-cli-rs` fails | Formula doesn't exist. Use `npm install -g near-cli-rs` |
| Implicit account ID is 64 hex chars | This is normal — it's the public key hash. Not a bug. |
| Account doesn't exist error | Fund it first. Implicit accounts aren't on-chain until first deposit. |
| Named account costs NEAR | ~0.1 NEAR for storage. Can't create if implicit account has exactly 0. |
| Keys saved to `~/.near-credentials/mainnet/` | Both implicit and named accounts. Confirm the file exists after creation. |
