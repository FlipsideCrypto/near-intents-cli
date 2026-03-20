# Using near-cli-rs for NEAR Intents

This topic covers how to use `near-cli-rs` (the `near` CLI) alongside `near-intents` to execute swaps on NEAR. When near-cli-rs is available, the agent can sign and submit transactions directly — enabling fully autonomous native swaps without sending the user to a browser.

near-cli-rs is an **optional accelerator**. Its presence unlocks autonomous on-chain execution but doesn't force any path. The user's intent drives the decision.

## Decision Framework

### Which swap mode?

```
User wants to swap tokens
├── Cross-chain (tokens on different blockchains, e.g., ETH → NEAR)
│   → Use: near-intents swap (signing URL flow)
│   → near-cli-rs still helps with post-swap tasks if destination is NEAR
│     (withdraw from intents, check balances, register storage)
│
└── NEAR-to-NEAR (both tokens already on NEAR)
    ├── near-cli-rs available + keys imported
    │   → Fully agentic: agent signs & submits tx directly
    ├── near-cli-rs available, keys NOT imported
    │   → Guide key import (see Onboarding below), then fully agentic
    └── near-cli-rs NOT available
        → Use: near-intents swap --native (signing URL fallback)
```

### Is the user onboarded?

```
Does the user have a NEAR account with keys in near-cli-rs?
├── Yes → Ready to execute
└── No
    ├── near-cli-rs available → Guide account creation/import (see Onboarding)
    └── near-cli-rs NOT available → Direct to web wallet creation
```

## Detection & Setup

Check if near-cli-rs is installed and ready:

```bash
# Is near-cli-rs installed?
which near
near --version

# Which networks are configured?
near config show-connections

# Does the account exist on-chain? What keys does it have?
near account list-keys <account_id> network-config <network> now
```

**Important:** `list-keys` queries **on-chain** access keys. It confirms the account exists and has keys, but does NOT confirm the local keychain has the corresponding private key. To verify local signing capability, check for legacy keychain files at `~/.near-credentials/<network>/<account_id>.json` or attempt a transaction — near-cli-rs will error if no local key is available.

**Network defaulting:** Use `mainnet` unless the user explicitly says testnet or the account ID ends in `.testnet`.

## Account Onboarding

### Import an existing account

For autonomous agent use, prefer private key or seed phrase import (no browser required):

```bash
# Via private key (best for autonomous use)
near account import-account using-private-key <private_key> network-config <network>

# Via seed phrase
near account import-account using-seed-phrase "<word1 word2 ... word12>" --hd-path "m/44'/397'/0'" network-config <network>

# Via web wallet (interactive only — requires browser, not suitable for autonomous agents)
near account import-account using-web-wallet network-config <network>
```

### Create a new account

```bash
# Testnet — free via faucet
near account create-account sponsor-by-faucet-service <name>.testnet autogenerate-new-keypair save-to-keychain network-config testnet

# Mainnet — fund from existing account
near account create-account fund-myself <new_account_id> '<amount> NEAR' autogenerate-new-keypair save-to-keychain sign-as <funding_account> network-config mainnet sign-with-keychain send
```

### Verify setup

```bash
# View account summary (balance, storage, keys, delegated stake)
near account view-account-summary <account_id> network-config <network> now

# List on-chain access keys
near account list-keys <account_id> network-config <network> now
```

## Pre-Swap Operations

Before executing a native swap, ensure the account is ready.

### Check NEAR balance

```bash
near tokens <account_id> view-near-balance network-config <network> now
```

### Wrap NEAR → wNEAR

Raw NEAR cannot be used in native swaps — it must be wrapped first. The first call also registers storage on `wrap.near` (~0.00125 NEAR).

```bash
near contract call-function as-transaction wrap.near near_deposit json-args '{}' prepaid-gas '30 Tgas' attached-deposit '<amount> NEAR' sign-as <account_id> network-config <network> sign-with-keychain send
```

### Register storage on destination token contract

Before an account can hold any NEP-141 token, it must have storage registered on that token's contract. This costs ~0.0125 NEAR and only needs to be done once per token per account.

```bash
near contract call-function as-transaction <token_contract> storage_deposit json-args '{"account_id": "<account_id>"}' prepaid-gas '30 Tgas' attached-deposit '0.0125 NEAR' sign-as <account_id> network-config <network> sign-with-keychain send
```

### Check FT balance

```bash
near contract call-function as-read-only <token_contract> ft_balance_of json-args '{"account_id": "<account_id>"}' network-config <network> now
```

## Executing Native Swaps

This is the core loop — the bridge between `near-intents` and `near-cli-rs`.

### Step 1: Preview the rate

```bash
near-intents quote --native --from <token> --to <token> --amount <amount>
```

### Step 2: Execute the swap to get a `nearTransaction`

```bash
near-intents swap --native --from <token> --to <token> --amount <amount> \
  --recipient <account_id> --refund-to <account_id> --sender <account_id>
```

This returns a `nearTransaction` object:

```json
{
  "nearTransaction": {
    "contractId": "wrap.near",
    "method": "ft_transfer_call",
    "args": {
      "receiver_id": "intents.near",
      "amount": "1000000000000000000000000",
      "msg": "<deposit-address>"
    },
    "gas": "100 Tgas",
    "deposit": "1 yoctoNEAR",
    "signerId": "alice.near"
  }
}
```

### Step 3: Translate `nearTransaction` to a near-cli-rs command

Map each field from the JSON to the near-cli-rs command structure:

```
near contract call-function as-transaction <contractId> <method> \
  json-args '<args as JSON string>' \
  prepaid-gas '<gas>' \
  attached-deposit '<deposit>' \
  sign-as <signerId> \
  network-config <network> \
  sign-with-keychain send
```

Concrete example using the JSON above:

```bash
near contract call-function as-transaction wrap.near ft_transfer_call \
  json-args '{"receiver_id":"intents.near","amount":"1000000000000000000000000","msg":"<deposit-address>"}' \
  prepaid-gas '100 Tgas' \
  attached-deposit '1 yoctoNEAR' \
  sign-as alice.near \
  network-config mainnet \
  sign-with-keychain send
```

### Step 4: Extract the transaction hash and submit it

After successful submission, near-cli-rs prints the transaction result. Look for the transaction hash in the output — it appears as `Transaction ID: <hash>` or in the explorer URL (e.g., `https://nearblocks.io/txns/<hash>`). Parse this hash and submit it to speed up processing:

```bash
near-intents submit-tx --deposit-address <addr> --tx-hash <hash> --near-sender <account_id>
```

### Step 5: Poll for completion

```bash
near-intents status --deposit-address <addr>
```

Repeat until you get a terminal status (`SUCCESS`, `FAILED`, `REFUNDED`, or `INCOMPLETE_DEPOSIT`).

## Post-Swap Operations

### Withdraw tokens from intents

After a native swap completes, output tokens are in the user's balance **inside** `intents.near`, NOT in the actual FT contract. You must withdraw before the tokens can be used:

```bash
near contract call-function as-transaction intents.near ft_withdraw \
  json-args '{"token":"<contract_id>","amount":"<raw_amount>","receiver_id":"<account_id>"}' \
  prepaid-gas '100 Tgas' \
  attached-deposit '1 yoctoNEAR' \
  sign-as <account_id> \
  network-config <network> \
  sign-with-keychain send
```

**Critical:** The `token` field takes the **bare contract account ID**, NOT the `nep141:` prefixed asset ID. Strip the prefix:
- `nep141:wrap.near` → `"token": "wrap.near"`
- `nep141:eth-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48.omft.near` → `"token": "eth-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48.omft.near"`

### Verify destination balance

```bash
near contract call-function as-read-only <token_contract> ft_balance_of \
  json-args '{"account_id": "<account_id>"}' \
  network-config <network> now
```

### Chaining multiple swaps

When doing sequential swaps (e.g., wNEAR → USDC → another token), you MUST withdraw between each swap. Tokens stay in the intents balance until explicitly withdrawn. Skipping withdrawal causes "account doesn't have enough balance" errors.

## Cross-Chain Swaps (Where near-cli-rs Helps)

Even in cross-chain flows (handled via signing URL), near-cli-rs is useful when the destination is NEAR:

- **Withdraw** received tokens from intents after swap completes
- **Check balances** of received tokens
- **Register storage** on new token contracts before receiving
- **Wrap/unwrap NEAR** as needed

### Unwrap wNEAR → NEAR

```bash
near contract call-function as-transaction wrap.near near_withdraw \
  json-args '{"amount":"<raw_amount_in_yocto>"}' \
  prepaid-gas '30 Tgas' \
  attached-deposit '1 yoctoNEAR' \
  sign-as <account_id> \
  network-config <network> \
  sign-with-keychain send
```

## Quick Reference

| Category | Command | Purpose |
|----------|---------|---------|
| Detect | `which near` | Check if near-cli-rs is installed |
| Detect | `near --version` | Get CLI version |
| Detect | `near config show-connections` | List configured networks |
| Account | `near account view-account-summary <acct> network-config <net> now` | View account info |
| Account | `near account list-keys <acct> network-config <net> now` | List on-chain access keys |
| Account | `near account import-account using-private-key <key> network-config <net>` | Import by private key |
| Account | `near account import-account using-seed-phrase "<words>" --hd-path "m/44'/397'/0'" network-config <net>` | Import by seed phrase |
| Account | `near account create-account sponsor-by-faucet-service <name>.testnet autogenerate-new-keypair save-to-keychain network-config testnet` | Create testnet account |
| Account | `near account create-account fund-myself <name> '<amt> NEAR' autogenerate-new-keypair save-to-keychain sign-as <funder> network-config <net> sign-with-keychain send` | Create funded account |
| Balance | `near tokens <acct> view-near-balance network-config <net> now` | Check NEAR balance |
| Balance | `near contract call-function as-read-only <token> ft_balance_of json-args '{"account_id":"<acct>"}' network-config <net> now` | Check FT balance |
| Wrap | `near contract call-function as-transaction wrap.near near_deposit json-args '{}' prepaid-gas '30 Tgas' attached-deposit '<amt> NEAR' sign-as <acct> network-config <net> sign-with-keychain send` | Wrap NEAR → wNEAR |
| Wrap | `near contract call-function as-transaction wrap.near near_withdraw json-args '{"amount":"<raw>"}' prepaid-gas '30 Tgas' attached-deposit '1 yoctoNEAR' sign-as <acct> network-config <net> sign-with-keychain send` | Unwrap wNEAR → NEAR |
| Storage | `near contract call-function as-transaction <token> storage_deposit json-args '{"account_id":"<acct>"}' prepaid-gas '30 Tgas' attached-deposit '0.0125 NEAR' sign-as <acct> network-config <net> sign-with-keychain send` | Register token storage |
| Swap | `near contract call-function as-transaction <contract> ft_transfer_call json-args '<args>' prepaid-gas '100 Tgas' attached-deposit '1 yoctoNEAR' sign-as <acct> network-config <net> sign-with-keychain send` | Execute swap deposit |
| Withdraw | `near contract call-function as-transaction intents.near ft_withdraw json-args '{"token":"<contract>","amount":"<raw>","receiver_id":"<acct>"}' prepaid-gas '100 Tgas' attached-deposit '1 yoctoNEAR' sign-as <acct> network-config <net> sign-with-keychain send` | Withdraw from intents |

## Common Errors & Fixes

| Error | Cause | Fix |
|-------|-------|-----|
| Account doesn't exist | No on-chain account | Create or import account (see Onboarding) |
| Access key not found | Keys not in near-cli-rs keychain | Import via private key or seed phrase |
| Not enough balance | Insufficient NEAR or wNEAR | Check balance, acquire/wrap more NEAR |
| Storage not registered | Missing `storage_deposit` on token contract | Run `storage_deposit` (see Pre-Swap) |
| Tokens stuck in intents | Didn't withdraw after swap | Call `ft_withdraw` on `intents.near` (see Post-Swap) |
| Wrong token field in `ft_withdraw` | Used `nep141:` prefixed asset ID | Strip prefix, use bare contract ID |
| Transaction expired | Took too long to sign | Re-run the swap command for a fresh deposit address |
| `FunctionCallError` | Contract error (bad args, insufficient deposit) | Read the error — usually wrong amount, missing storage, or bad method args. Fix root cause, do NOT retry same params |
| `ActionError` | Protocol error (nonce, balance) | Check balance. If nonce error, retry immediately (transient) |
