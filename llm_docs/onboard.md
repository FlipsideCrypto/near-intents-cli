# NEAR Intents CLI — Agent Onboarding

You have access to `near-intents`, a CLI for cross-chain token swaps via the NEAR Intents (Defuse Protocol 1Click) API.

## IMPORTANT: Communicate before acting

Crypto transactions are irreversible. The user may not understand what a swap actually involves — wrapping tokens, storage deposits, gas costs, intents balances, withdrawal steps. What sounds simple ("swap my NEAR for some USDC") actually involves multiple on-chain transactions with real costs.

**Before executing any swap or on-chain action, you MUST:**

1. **Explain what you're about to do in plain language.** Not CLI flags — tell the user what will happen to their tokens, what fees they'll pay, and how many steps are involved. For example: "To swap 10 NEAR for USDC, I'll need to: (1) wrap your NEAR into wNEAR, (2) register your account on the USDC contract (~0.0125 NEAR fee), (3) execute the swap, (4) withdraw the USDC from the intents system to your wallet. This will cost roughly 0.06 NEAR in fees on top of the 10 NEAR being swapped."
2. **Confirm the user understands and wants to proceed.** Wait for explicit confirmation before executing.
3. **Clarify what the user will end up with.** "You'll receive ~X USDC in your NEAR wallet" — not "the swap will execute." Be concrete about the outcome.
4. **Flag anything unexpected.** If a token requires cross-chain flow instead of native, if a route isn't available, if the rate is significantly different from what the user might expect — say so before proceeding.

Do NOT assume the user understands:
- The difference between NEAR and wNEAR
- That tokens land in an intents balance and need withdrawal
- What storage registration is or why it costs NEAR
- That cross-chain swaps take minutes while native swaps take seconds
- That a `signingUrl` must be used (not replicated manually)

When in doubt, over-explain. A confused user who approves a transaction they didn't understand is worse than a user who has to read an extra sentence.

## Setup & Authentication

Authentication is via JWT bearer token, obtained from the Partner Dashboard at https://partners.near-intents.org/

Set your token (pick one):
- Flag: `near-intents --jwt <token> <command>`
- Environment variable: `NEAR_INTENTS_JWT_TOKEN=<jwt>`
- Config file: `~/.near-intents.json` → `{"token": "<jwt>"}`

**Without a token, swaps still work** but incur a platform fee. All commands function without authentication.

### Check for existing NEAR wallets

Before asking the user for their NEAR account, check for local near-cli credentials:
- `~/.near-credentials/mainnet/` — contains `<account>.json` files for each account
- The account name is the filename (e.g., `alice.near.json` → account is `alice.near`)

If credentials exist, use that account as the default for `--sender`, `--recipient`, and `--refund-to`.

## Token Resolution

Two ways to specify tokens:

1. **Asset ID** (exact): `--from nep141:wrap.near`
2. **Symbol + chain** (resolved): `--from USDC --from-chain ethereum`

**Never construct asset IDs manually.** Always use `near-intents tokens` to find the correct asset ID.

### Asset ID formats

- **`nep141:` prefix** — standard NEAR NEP-141 tokens. These work with `--native` mode.
  - NEAR native: `nep141:wrap.near`
  - Bridged tokens: `nep141:{chain}-{contractAddress}.omft.near`
- **`1cs_v1:` prefix** — tokens that require the cross-chain 1Click swap flow. These do NOT work with `--native`. Example: `1cs_v1:near:nep141:zec.omft.near` (ZEC). If you see a `1cs_v1:` asset ID, you must use the standard cross-chain flow (no `--native` flag).

## Cross-chain Swap Workflow

Follow these steps for swaps between different blockchains:

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

**WARNING: The user MUST use the `signingUrl` to sign and submit the deposit. Do NOT manually construct or replicate the deposit transaction (e.g., via ft_transfer_call or a direct transfer to the deposit address). The signing page handles chain-specific deposit logic. Manually sending tokens to the deposit address will result in stuck funds until the deadline expires.**

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

Add `--native` to `quote` or `swap` to swap wrapped/bridged assets already on NEAR. These swaps finalize in ~1 second. Use this when both tokens are `nep141:` assets on the `near` blockchain.

**Key things to know before using native mode:**

1. **Raw NEAR must be wrapped first.** Users typically have NEAR, not wNEAR. You must wrap it via `near_deposit` on `wrap.near` before swapping.
2. **Swap output lands in the intents balance, NOT the wallet.** After a native swap, output tokens are inside `intents.near`, not in the token's FT contract. You must call `ft_withdraw` on `intents.near` to move them out.
3. **Storage registration required.** Before receiving or withdrawing any NEP-141 token, the account needs `storage_deposit` on that token's contract (~0.0125 NEAR each, once per token).
4. **Only `nep141:` tokens work with `--native`.** Tokens with `1cs_v1:` prefix require the cross-chain flow.
5. **Budget NEAR for gas and storage.** Reserve = 0.5 NEAR + (0.0125 × number of new tokens) + (0.01 × number of transactions).

### Native swap end-to-end (starting from raw NEAR)

```
# 0. Check for existing NEAR account in ~/.near-credentials/mainnet/

# 1. Wrap NEAR → wNEAR (on-chain: near_deposit on wrap.near)
#    Attach the NEAR amount to swap. First call also registers storage (~0.00125 NEAR).

# 2. Register storage on destination token contract (on-chain: storage_deposit, ~0.0125 NEAR)

# 3. Preview the rate
near-intents quote --native --from wNEAR --to USDC --amount 10

# 4. Execute the swap
near-intents swap --native --from wNEAR --to USDC --amount 10 \
  --recipient alice.near --refund-to alice.near --sender alice.near

# 5. Execute the nearTransaction on NEAR (on-chain: ft_transfer_call from the response)

# 6. Submit tx hash
near-intents submit-tx --deposit-address <addr> --tx-hash <hash> --near-sender alice.near

# 7. Poll status until SUCCESS
near-intents status --deposit-address <addr>

# 8. Withdraw output tokens from intents (on-chain: ft_withdraw on intents.near)
#    The "token" field is the bare contract ID (strip nep141: prefix), NOT the full asset ID.

# 9. If chaining another swap: repeat from step 2
```

For full details on native swaps, wrapped assets, ft_transfer_call, withdrawals, and chaining, run: `near-intents llm topic native-swaps`

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
| `--native` | No | false | Use NEAR-native intents mode (both tokens must be on NEAR, nep141: only) |

### `near-intents swap`
Execute swap — generates deposit address and signing URL (or nearTransaction in native mode).

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
- **Funds stuck after manual deposit**: If you manually sent tokens to a deposit address instead of using the `signingUrl`, funds are locked until the deadline expires and are then refunded.

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

## Deep-dive topics

For detailed knowledge on specific topics, run `near-intents llm topics` to see what's available, then `near-intents llm topic <name>` to read a topic.
