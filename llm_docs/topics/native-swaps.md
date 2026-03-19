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

## Complete native swap workflow

```bash
# 1. Find available NEAR tokens
near-intents tokens --chain near

# 2. Preview the rate
near-intents quote --native --from wNEAR --to USDC --amount 10

# 3. Execute the swap
near-intents swap --native --from wNEAR --to USDC --amount 10 \
  --recipient alice.near --refund-to alice.near --sender alice.near

# 4. User/agent executes the nearTransaction on NEAR
#    (using near-cli, NEAR SDK, or another tool)

# 5. Submit the transaction hash (speeds up processing)
near-intents submit-tx --deposit-address <addr> --tx-hash <hash> --near-sender alice.near

# 6. Check status
near-intents status --deposit-address <addr>
```

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
