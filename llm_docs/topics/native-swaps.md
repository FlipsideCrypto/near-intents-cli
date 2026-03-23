# Native Swaps (NEAR-only intents)

## What are native swaps?

Native swaps let users swap tokens that are already inside the NEAR Intents system without involving any external blockchain. The entire swap happens on NEAR and finalizes in ~1 second.

This is in contrast to cross-chain swaps, where tokens move between different blockchains (e.g., Ethereum → NEAR) and take minutes due to chain finality requirements.

## The NEAR Intents system

The NEAR Intents protocol is built around a **Verifier contract** deployed at `intents.near` on the NEAR blockchain. This contract:

- Holds token balances for users (like an on-chain escrow)
- Tracks deposits and withdrawals
- Coordinates with solvers/market makers to execute swaps

When tokens are "inside the intents system," they exist as NEP-141 fungible token balances managed by this contract.

## Wrapped and bridged assets

All tokens in the intents system are NEP-141 tokens on NEAR. They fall into two categories:

**NEAR-native tokens** — tokens that originate on NEAR:
- `nep141:wrap.near` — Wrapped NEAR (wNEAR)
- `nep141:usdt.tether-token.near` — USDT on NEAR

**Bridged tokens** — representations of tokens from other chains, using the OMFT (Omnichain Fungible Token) format:
- `nep141:eth-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48.omft.near` — USDC bridged from Ethereum
- `nep141:arb-0xaf88d065e77c8cc2239327c5edb3a432268e5831.omft.near` — USDC from Arbitrum
- `nep141:sol-EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v.omft.near` — USDC from Solana

The pattern is `nep141:{chain}-{contractAddress}.omft.near`. These bridged tokens are fully fungible within the intents system — a user can swap Ethereum-bridged USDC for Arbitrum-bridged USDC, or for wNEAR, all on NEAR.

## How native swaps work

### The API difference

The same 1Click API is used for both native and cross-chain swaps. The difference is three fields in the quote request:

| Field | Cross-chain | Native |
|-------|-------------|--------|
| `depositType` | `ORIGIN_CHAIN` | `INTENTS` |
| `recipientType` | `DESTINATION_CHAIN` | `INTENTS` |
| `refundType` | `ORIGIN_CHAIN` | `INTENTS` |

The CLI handles this automatically with the `--native` flag.

### The deposit mechanism: `ft_transfer_call`

In cross-chain mode, the user sends tokens on an external chain to a generated deposit address, and the protocol bridges them in.

In native mode, the user already has tokens on NEAR. They deposit into the intents system using the NEP-141 standard `ft_transfer_call` method:

1. The API returns a `depositAddress` — a routing key within the intents system
2. The user calls `ft_transfer_call` on the token contract (e.g., `wrap.near`) with:
   - `receiver_id`: `"intents.near"` (the Verifier contract)
   - `amount`: the raw token amount
   - `msg`: the `depositAddress` from the API (routes the deposit to the correct swap)
3. The Verifier contract receives the tokens and the solver fulfills the swap
4. The destination tokens are credited to the recipient's intents account

### The `msg` field

The `msg` parameter in `ft_transfer_call` controls how the Verifier contract handles the deposit:

- **Empty string** (`""`) — tokens are credited to the sender's intents balance (no swap, just a deposit)
- **Account ID** (e.g., `"alice.near"`) — tokens are credited to a different account's balance
- **Deposit address from API** — routes the deposit to a pending swap intent

When using the CLI's `--native` flag, the `msg` field is automatically set to the API's deposit address.

### The `nearTransaction` output

When `--native` is used with the `swap` command, the output includes a `nearTransaction` object instead of a `signingUrl`:

```json
{
  "nearTransaction": {
    "contractId": "wrap.near",
    "method": "ft_transfer_call",
    "args": {
      "receiver_id": "intents.near",
      "amount": "1000000000000000000000000",
      "msg": "deposit-address-from-api"
    },
    "gas": "100 Tgas",
    "deposit": "1 yoctoNEAR",
    "signerId": "alice.near"
  }
}
```

This contains everything needed to construct and submit a NEAR transaction. The consuming tool (e.g., `near-cli`, a NEAR SDK, or another agent) is responsible for signing and broadcasting it.

Field reference:
- `contractId` — the NEP-141 token contract to call (derived from the asset ID)
- `method` — always `ft_transfer_call`
- `args.receiver_id` — always `intents.near`
- `args.amount` — raw token amount in smallest units
- `args.msg` — the deposit address for swap routing
- `gas` — recommended gas limit (100 TGas covers the cross-contract call)
- `deposit` — attached NEAR deposit (1 yoctoNEAR required by NEP-141 standard)
- `signerId` — the NEAR account that will sign (from `--sender` flag)

## Critical: Prerequisites before swapping

Native swaps require several on-chain setup steps. Skipping these will cause failures.

### 1. Wrapping NEAR

Raw NEAR cannot be used in native swaps — it must be wrapped to wNEAR first via `wrap.near`:

```json
{
  "contractId": "wrap.near",
  "method": "near_deposit",
  "args": {},
  "gas": "30 Tgas",
  "deposit": "<amount_in_NEAR>"
}
```

The first call to `near_deposit` also registers storage on `wrap.near`, consuming ~0.00125 NEAR for storage. Account for this in the deposit amount.

### 2. Storage registration on destination token contracts

Before an account can hold any NEP-141 token, it must have storage registered on that token's contract. Without this, transfers fail silently or revert.

```json
{
  "contractId": "<destination_token_contract>",
  "method": "storage_deposit",
  "args": { "account_id": "<user_account>" },
  "gas": "30 Tgas",
  "deposit": "0.0125 NEAR"
}
```

This costs ~0.0125 NEAR per token contract registration. You only need to do this once per token per account.

### 3. Swap output stays in intents balance, NOT the FT contract

**This is the most common gotcha.** After a native swap completes, the output tokens are credited to the user's balance *inside* `intents.near` (as a multi-token balance), NOT as actual NEP-141 tokens in the destination token's contract. This means:

- You **cannot** immediately do another `ft_transfer_call` with the output tokens
- The destination token contract will report "account doesn't have enough balance"
- You must **withdraw** tokens from intents before using them

### 4. Withdrawing tokens from the intents system

To move tokens from the intents internal balance to the actual FT contract balance, call `ft_withdraw` on `intents.near`:

```json
{
  "contractId": "intents.near",
  "method": "ft_withdraw",
  "args": {
    "token": "<contract_account_id>",
    "amount": "<raw_amount>",
    "receiver_id": "<destination_account_id>"
  },
  "gas": "100 Tgas",
  "deposit": "1 yoctoNEAR"
}
```

**Important:** The `token` field takes the **bare contract account ID** (e.g., `17208628f84f5d6ad33f0da3bbbeb27ffcb398eac501a31bd6ad2011e36133a1`), NOT the `nep141:` prefixed asset ID. Using the full asset ID causes a deserialization error.

To derive the contract ID from an asset ID: strip the `nep141:` prefix. For example:
- `nep141:wrap.near` → `token: "wrap.near"`
- `nep141:eth-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48.omft.near` → `token: "eth-0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48.omft.near"`

## Token compatibility

**Only `nep141:` tokens work with `--native` mode.** Some tokens on the `near` blockchain have a `1cs_v1:` prefix in their asset ID (e.g., `1cs_v1:near:nep141:zec.omft.near` for ZEC). These require the cross-chain 1Click swap flow and will NOT work with `--native`.

Before using `--native`, check the asset ID returned by `near-intents tokens`. If it starts with `nep141:`, native mode works. If it starts with `1cs_v1:`, use the standard cross-chain flow.

## NEAR reserve budget

Native swaps consume NEAR for gas, storage registration, and wrapping. Budget accordingly:

**Formula:** reserve = 0.5 NEAR + (0.0125 × number of new destination tokens) + (0.01 × number of transactions)

Example: swapping into 3 new tokens with ~6 transactions total = 0.5 + (0.0125 × 3) + (0.01 × 6) = 0.5975 NEAR

This is in addition to the NEAR being swapped. Always check the account balance before starting.

## When to use native vs cross-chain

**Use native (`--native`) when:**
- User wants to swap between any tokens on the `near` blockchain
- Both tokens are NEP-141 (asset IDs start with `nep141:`)
- User already has tokens in the intents system or on NEAR
- Speed matters — native swaps finalize in ~1 second

**Use cross-chain (default) when:**
- Tokens are on different blockchains (e.g., ETH on Ethereum → wNEAR on NEAR)
- The source token is not on NEAR
- User needs to bridge tokens into or out of NEAR
- The token has a `1cs_v1:` asset ID prefix (even if on the `near` blockchain)

## Complete native swap walkthrough

This is a step-by-step walkthrough with real commands. Replace `alice.near` with the actual account ID throughout.

### Pre-flight checklist

Before starting, verify:
```bash
# near-cli-rs installed
near --version

# Credentials exist locally
ls ~/.near-credentials/mainnet/

# Account exists on-chain and has NEAR
near account view-account-summary alice.near network-config mainnet now
```

Budget check: you need the swap amount + 0.5 NEAR reserve + 0.0125 per new token + 0.01 per transaction.

---

### Step 1: Get a rate quote

```bash
near-intents quote --native --from nep141:wrap.near --to USDC --amount 10 --pretty
```

Verify the rate looks reasonable before committing.

---

### Step 2: Initiate the swap — get the deposit address

```bash
near-intents swap --native \
  --from nep141:wrap.near --to USDC --amount 10 \
  --recipient alice.near --refund-to alice.near --sender alice.near \
  --pretty
```

Save the output — you need `depositAddress` and the `nearTransaction` object.

---

### Step 3: Register your account on wrap.near (if first time)

Your account needs a storage slot on the wNEAR contract before it can hold wNEAR.

```bash
near contract call-function as-transaction wrap.near storage_deposit \
  json-args '{"account_id":"alice.near"}' \
  prepaid-gas '30 Tgas' attached-deposit '0.0125 NEAR' \
  sign-as alice.near network-config mainnet sign-with-keychain send
```

*Skip if you have used wNEAR before — it's a one-time registration per account.*

---

### Step 4: Wrap NEAR → wNEAR

```bash
near contract call-function as-transaction wrap.near near_deposit \
  json-args '{}' \
  prepaid-gas '30 Tgas' attached-deposit '10 NEAR' \
  sign-as alice.near network-config mainnet sign-with-keychain send
```

---

### Step 5: Register the deposit address on wrap.near ⚠️ CRITICAL — commonly missed

The deposit address returned by the swap API is a NEAR account. It also needs a storage slot on wrap.near before it can receive your wNEAR. Without this step, the ft_transfer_call in step 6 will fail with "account not registered."

```bash
near contract call-function as-transaction wrap.near storage_deposit \
  json-args '{"account_id":"<depositAddress>"}' \
  prepaid-gas '30 Tgas' attached-deposit '0.0125 NEAR' \
  sign-as alice.near network-config mainnet sign-with-keychain send
```

Replace `<depositAddress>` with the exact value from the swap output.

---

### Step 6: Submit the ft_transfer_call

Use the exact values from the `nearTransaction` in the swap response — do not modify them:

```bash
near contract call-function as-transaction wrap.near ft_transfer_call \
  json-args '{"receiver_id":"intents.near","amount":"<amount_from_response>","msg":"<msg_from_response>"}' \
  prepaid-gas '100 Tgas' attached-deposit '1 yoctoNEAR' \
  sign-as alice.near network-config mainnet sign-with-keychain send
```

---

### Step 7: Submit the transaction hash

Parse the transaction hash from the near-cli output (look for `Transaction ID:` or the nearblocks.io URL), then:

```bash
near-intents submit-tx --deposit-address <depositAddress> --tx-hash <hash> --near-sender alice.near
```

---

### Step 8: Poll for completion

```bash
near-intents status --deposit-address <depositAddress> --pretty
```

Repeat every 5–10 seconds until status is `SUCCESS`, `FAILED`, `REFUNDED`, or `INCOMPLETE_DEPOSIT`.

---

### Step 9: Withdraw output tokens from intents

After `SUCCESS`, the output tokens are inside `intents.near`, not in your wallet yet. Withdraw them:

```bash
near contract call-function as-transaction intents.near ft_withdraw \
  json-args '{"token":"<bare_contract_id>","amount":"<raw_amount>","receiver_id":"alice.near"}' \
  prepaid-gas '100 Tgas' attached-deposit '1 yoctoNEAR' \
  sign-as alice.near network-config mainnet sign-with-keychain send
```

The `token` field is the bare contract ID — strip the `nep141:` prefix:
- `nep141:usdc.omft.near` → `"token": "usdc.omft.near"`
- `nep141:wrap.near` → `"token": "wrap.near"`

## Chaining multiple swaps

When doing multiple native swaps in sequence (e.g., wNEAR → USDC → SOL), you MUST withdraw the output tokens from intents between swaps:

1. Execute first swap (wNEAR → USDC)
2. Wait for terminal status
3. Call `ft_withdraw` on `intents.near` to move USDC from intents balance to the FT contract
4. Register storage on next destination token contract if needed
5. Execute second swap (USDC → SOL)

Skipping the withdrawal step causes "account doesn't have enough balance" errors because the tokens exist inside intents, not in the FT contract.

## Required flags for native swap

| Flag | Command | Description |
|------|---------|-------------|
| `--native` | `quote`, `swap` | Enables native intents mode |
| `--sender` | `swap` | NEAR account signing the transaction (required with `--native`) |
| `--recipient` | `swap` | NEAR account receiving the swapped tokens |
| `--refund-to` | `swap` | NEAR account for refunds if swap fails |
| `--from` | `quote`, `swap` | Origin token (symbol or asset ID) |
| `--to` | `quote`, `swap` | Destination token (symbol or asset ID) |
| `--amount` | `quote`, `swap` | Human-readable amount |

Note: `--from-chain` and `--to-chain` default to `near` when `--native` is set. You can omit them unless needed for disambiguation.
