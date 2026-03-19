# NEAR Intents CLI — Agent Onboarding

You have access to `near-intents`, a CLI for cross-chain token swaps via the NEAR Intents (Defuse Protocol 1Click) API.

## Setup & Authentication

Authentication is via JWT bearer token, obtained from the Partner Dashboard at https://partners.near-intents.org/

Set your token (pick one):
- Flag: `near-intents --jwt <token> <command>`
- Environment variable: `NEAR_INTENTS_JWT_TOKEN=<jwt>`
- Config file: `~/.near-intents.json` → `{"token": "<jwt>"}`

**Without a token, swaps still work** but incur a platform fee. All commands function without authentication.

## Swap Workflow

Follow these steps in order for every swap:

### Step 1: Resolve tokens
```
near-intents tokens --search USDC --chain ethereum
```
Find the tokens you need. Note their `assetId`, `decimals`, and `contractAddress`.

### Step 2: Preview the rate (optional but recommended)
```
near-intents quote --from USDC --from-chain ethereum --to wNEAR --to-chain near --amount 10
```
This is a dry run — no deposit address generated. Check the rate before committing.

### Step 3: Execute the swap
```
near-intents swap --from USDC --from-chain ethereum --to wNEAR --to-chain near --amount 10 --recipient alice.near --refund-to 0xYourEthAddr
```
Returns:
- `depositAddress` — where the user sends tokens (~10 min validity)
- `signingUrl` — URL for the user to connect wallet and sign the deposit

### Step 4: User signs the deposit
Direct the user to the `signingUrl`. They will:
1. Connect their wallet (MetaMask, Phantom, NEAR Wallet, etc.)
2. Sign a standard transfer to the deposit address
3. The page handles the rest

### Step 5: Submit transaction hash (optional, speeds up processing)
```
near-intents submit-tx --deposit-address <addr> --tx-hash <hash>
```

### Step 6: Poll for completion
```
near-intents status --deposit-address <addr>
```
Repeat until you get a terminal status.

## Native Mode (NEAR-only swaps)

Use `--native` when swapping tokens that are already inside the NEAR Intents system (wrapped/bridged assets on NEAR). These swaps finalize in ~1 second — no cross-chain wait.

### When to use `--native`

- User wants to swap between wrapped assets on NEAR (e.g., wNEAR ↔ USDC on NEAR)
- Both source and destination tokens are on the `near` blockchain
- User wants sub-second swap finality
- User already has tokens deposited in the intents system

Do NOT use `--native` when tokens need to move between different blockchains (e.g., Ethereum USDC → NEAR wNEAR). Use the standard cross-chain flow instead.

### Native swap workflow

#### Step 1: Preview the rate
```
near-intents quote --native --from wNEAR --to USDC --amount 10
```
Same output as a regular quote. `--from-chain` / `--to-chain` default to `near` in native mode.

#### Step 2: Execute the swap
```
near-intents swap --native --from wNEAR --to USDC --amount 10 \
  --recipient alice.near --refund-to alice.near --sender alice.near
```
Returns a `nearTransaction` object instead of a `signingUrl`:
```json
{
  "nearTransaction": {
    "contractId": "wrap.near",
    "method": "ft_transfer_call",
    "args": {
      "receiver_id": "intents.near",
      "amount": "1000000000000000000000000",
      "msg": "<depositAddress>"
    },
    "gas": "100 Tgas",
    "deposit": "1 yoctoNEAR",
    "signerId": "alice.near"
  }
}
```
The `nearTransaction` contains everything needed to construct a NEAR transaction. The user (or another tool) must execute this `ft_transfer_call` on the NEAR blockchain.

#### Step 3: Submit transaction hash
```
near-intents submit-tx --deposit-address <addr> --tx-hash <hash> --near-sender alice.near
```

#### Step 4: Poll for completion
```
near-intents status --deposit-address <addr>
```

## Commands Reference

### `near-intents tokens`
List/search supported tokens. Filtering is client-side.

| Flag | Description |
|------|-------------|
| `--chain <name>` | Filter by blockchain (ethereum, solana, near, base, arbitrum, etc.) |
| `--search <term>` | Match on symbol, blockchain, or asset ID |

Returns: `[{assetId, symbol, blockchain, decimals, price, contractAddress}, ...]`

### `near-intents quote`
Dry-run quote — check rates without committing.

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--from` | Yes | — | Origin token (asset ID like `nep141:wrap.near` or symbol like `USDC`) |
| `--to` | Yes | — | Destination token |
| `--from-chain` | When --from is a symbol | — | Origin blockchain |
| `--to-chain` | When --to is a symbol | — | Destination blockchain |
| `--amount` | Yes | — | Human-readable amount (e.g., 1.5 for 1.5 USDC) |
| `--swap-type` | No | EXACT_INPUT | EXACT_INPUT, EXACT_OUTPUT, or FLEX_INPUT |
| `--slippage` | No | 100 | Slippage in basis points (100 = 1%) |
| `--recipient` | No | — | Optional for quote |
| `--refund-to` | No | — | Optional for quote |
| `--app-fee` | No | — | Partner fee in basis points |
| `--fee-recipient` | No | — | NEAR address for partner fees |
| `--native` | No | false | Use NEAR-native intents mode (both tokens must be on NEAR) |

### `near-intents swap`
Execute swap — generates deposit address and signing URL.

All flags from `quote`, plus:

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--recipient` | Yes | — | Destination address for swapped tokens |
| `--refund-to` | Yes | — | Refund address on origin chain |
| `--deadline` | No | 1h | Deadline duration (e.g., 30m, 2h) |
| `--native` | No | false | Use NEAR-native intents mode |
| `--sender` | When `--native` | — | NEAR account that will sign the ft_transfer_call |

### `near-intents submit-tx`
Submit deposit transaction hash. Optional but recommended.

| Flag | Required | Description |
|------|----------|-------------|
| `--deposit-address` | Yes | Deposit address from swap output |
| `--tx-hash` | Yes | Transaction hash of the deposit |
| `--near-sender` | No | NEAR account that sent deposit (NEAR only) |
| `--memo` | No | Deposit memo (Stellar only) |

### `near-intents status`
Check swap progress. One-shot check (you handle polling).

| Flag | Required | Description |
|------|----------|-------------|
| `--deposit-address` | Yes | Deposit address to check |
| `--deposit-memo` | No | Deposit memo (Stellar only) |

## Token Resolution

Two ways to specify tokens:

1. **Asset ID** (exact): `--from nep141:wrap.near`
2. **Symbol + chain** (resolved): `--from USDC --from-chain ethereum`

**Never construct asset IDs manually.** Always use `near-intents tokens` to find the correct asset ID.

Common asset ID patterns:
- NEAR native: `nep141:wrap.near`
- Bridged tokens: `nep141:{chain}-{contractAddress}.omft.near`

## Status States

| Status | Terminal? | Meaning |
|--------|-----------|---------|
| `PENDING_DEPOSIT` | No | Waiting for user to deposit |
| `KNOWN_DEPOSIT_TX` | No | Deposit transaction detected |
| `PROCESSING` | No | Swap in progress |
| `SUCCESS` | Yes | Swap complete — check `swapDetails` for output tx hashes |
| `INCOMPLETE_DEPOSIT` | Yes | Wrong amount deposited |
| `REFUNDED` | Yes | Funds refunded to refund address |
| `FAILED` | Yes | Swap failed |

## Error Recovery

- **No route available**: Try a different token pair, or check token list for supported assets.
- **Deposit address expired**: Run `swap` again to get a new deposit address.
- **Incomplete deposit**: Funds will be automatically refunded to the `--refund-to` address.
- **Status stuck on PROCESSING**: Keep polling — cross-chain swaps can take several minutes.

## Portfolio Intelligence

The `intel` command connects to Flipside AI agents for portfolio analysis, rebalancing recommendations, and on-chain intelligence.

### Setup

Requires a Flipside API key (get one at https://flipsidecrypto.xyz):
- Flag: `near-intents intel --flipside-api-key <key>`
- Environment variable: `FLIPSIDE_API_KEY=<key>`
- Config file: `~/.near-intents.json` → `{"flipside_api_key": "<key>"}`

### `near-intents intel`

Send a natural language message to a Flipside AI agent. The agent can look up wallet balances, analyze portfolio composition, and recommend rebalancing strategies.

| Flag | Required | Default | Description |
|------|----------|---------|-------------|
| `--message` | Yes | — | Natural language request for the agent |
| `--flipside-api-key` | Yes (if not in env/config) | — | Flipside API key |
| `--agent` | No | `trading_agent` | Flipside agent to query |

### Example: Portfolio Rebalancing

```bash
# Ask the agent to analyze wallets and suggest rebalancing
near-intents intel --message "I have these wallets:
- 0xABC123 on ethereum
- alice.near on near
- 7xKP... on solana
Analyze my balances and suggest how to rebalance evenly across stablecoins."

# The response includes the agent's analysis and recommendations.
# Use the recommendations to execute swaps:
near-intents swap --from USDC --from-chain ethereum --to USDT --to-chain near --amount 500 \
  --recipient alice.near --refund-to 0xABC123
```

### Intel → Swap Workflow

1. Use `intel` to get portfolio analysis and rebalancing recommendations
2. Parse the agent's recommendations
3. Use `quote` to preview each recommended swap
4. Use `swap` to execute the swaps the user approves
5. Track each swap with `status`

## Output Format

Every command returns JSON:
```json
{"success": true, "data": {...}, "error": null}
```
or on failure:
```json
{"success": false, "data": null, "error": {"code": "ERROR_CODE", "message": "details"}}
```

Use `--pretty` for indented JSON output. Default is compact JSON for machine consumption.

## Example: Swap 10 USDC (Ethereum) → wNEAR

```bash
# 1. Find tokens
near-intents tokens --search USDC --chain ethereum
near-intents tokens --search wNEAR --chain near

# 2. Check the rate
near-intents quote --from USDC --from-chain ethereum --to wNEAR --to-chain near --amount 10

# 3. Execute swap
near-intents swap --from USDC --from-chain ethereum --to wNEAR --to-chain near --amount 10 \
  --recipient alice.near --refund-to 0xYourEthAddress

# 4. (User signs at the signingUrl from the response)

# 5. Submit tx hash
near-intents submit-tx --deposit-address <addr> --tx-hash <hash>

# 6. Check status
near-intents status --deposit-address <addr>
```

## Example: Native swap wNEAR → USDC on NEAR

```bash
# 1. Check the rate
near-intents quote --native --from wNEAR --to USDC --amount 10

# 2. Execute native swap
near-intents swap --native --from wNEAR --to USDC --amount 10 \
  --recipient alice.near --refund-to alice.near --sender alice.near

# 3. User executes the nearTransaction on NEAR (via near-cli or programmatically)

# 4. Submit tx hash
near-intents submit-tx --deposit-address <addr> --tx-hash <hash> --near-sender alice.near

# 5. Check status
near-intents status --deposit-address <addr>
```
