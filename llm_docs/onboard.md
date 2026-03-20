# NEAR Intents CLI — Agent Onboarding

You have access to `near-intents`, a CLI for cross-chain token swaps via the NEAR Intents (Defuse Protocol 1Click) API.

## Translating What the User Wants Into What NEAR Intents Can Do

When someone says "rebalance my portfolio" or "swap my NEAR for ETH", they have no idea about wrapping, storage deposits, intents balances, signing URLs, or cross-chain routing. Your job is to bridge that gap — do the discovery work, then present a plain-language plan before touching anything.

### Step 1: Check which of their assets are actually supported

Not everything can be swapped. Before planning anything, check:

```
near-intents tokens --search <symbol>
```

For each asset the user holds, determine:
- **`nep141:` asset ID found** → native mode is possible (fast, NEAR-only)
- **`1cs_v1:` asset ID only** → cross-chain mode only, even though it's on NEAR
- **Found on another chain** (ethereum, solana, etc.) → cross-chain swap, browser signing required
- **Not found at all** → can't swap it — tell the user upfront, don't skip over it

### Step 2: Map each holding to the simplest available path

| What they have | Where it is | What needs to happen |
|----------------|-------------|----------------------|
| Raw NEAR | Wallet | Wrap to wNEAR first, then native swap |
| wNEAR or nep141 token | Wallet | Check storage on destination, then native swap |
| Any nep141 token | Intents balance | Native swap directly — fastest path, already in the engine |
| ETH / SOL / BTC | Other chain | Cross-chain swap — user opens a browser signing URL |
| 1cs_v1 token | NEAR wallet | Cross-chain mode even though it's on NEAR |
| Unsupported token | Anywhere | Can't swap — tell the user |

### Step 3: Present a plain-language plan before doing anything

Don't start with CLI commands. Start with a summary the user can actually understand:

> "Here's what I can do with your portfolio:
>
> **Fast swaps (1–2 seconds, fully automated):**
> - Your 500 NEAR → ~450 USDC. I'll wrap the NEAR first, then swap. You'll end up with USDC in your NEAR wallet after a withdrawal step.
>
> **Slower swaps (5–15 min, requires browser):**
> - Your 0.05 ETH on Ethereum → ~180 NEAR. I'll generate a signing link you'll need to open to authorize the transfer.
>
> **Can't swap:**
> - Your XYZ token isn't listed on NEAR Intents — I'll leave that alone.
>
> Total estimated fees: ~0.06 NEAR + gas. Want me to proceed?"

Key things to include in your summary:
- Which swaps are **fast/automated** (native) vs **require browser action** (cross-chain)
- What they'll **actually end up with** — token, chain, and where (wallet vs intents balance)
- Any **assets you can't touch** and why
- A **rough fee estimate**
- An explicit **ask for confirmation** before doing anything

### What NEAR Intents cannot do

Be upfront about limitations rather than discovering them mid-execution:
- **No fiat on/off ramps** — crypto to crypto only
- **No unsupported tokens** — if it's not in `near-intents tokens`, it can't be swapped
- **No cross-chain withdrawal** — after a native swap, tokens land in the NEAR intents balance. Getting them to Ethereum requires a separate cross-chain swap.
- **No guaranteed rates** — quotes expire, slippage applies. The rate at confirmation may differ slightly from the rate at execution.
- **Cross-chain swaps need the user's browser** — you can't fully automate them. The user must open the `signingUrl`.

---

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

## Before you start: Discovery checklist

When a user asks you to do something with swaps (e.g., "rebalance my portfolio", "swap some NEAR for USDC", "buy some ETH"), don't jump into execution. Walk through these steps first:

### 1. Understand what they actually want
- What tokens do they want to end up with? In what amounts or proportions?
- Are they swapping on NEAR only, or across chains (e.g., Ethereum ↔ NEAR)?
- Do they want tokens in their NEAR wallet, or on another chain?
- If they say "rebalance" — into what? Equal weights? Specific allocations? Ask.

### 2. Identify the user's NEAR account
- Check `~/.near-credentials/mainnet/` for existing near-cli credentials — files are named `<account>.json`
- If found, confirm with the user: "I see a NEAR account `alice.near` on this machine — is that the one you want to use?"
- If not found, ask the user for their NEAR account ID

### 3. Check if near-cli is available
- For native swaps, the user needs to sign NEAR transactions. Check if `near` CLI is installed: `which near`
- If not installed, the user will need another way to sign transactions (wallet app, SDK, etc.) — let them know upfront
- For cross-chain swaps, near-cli is not needed (the `signingUrl` handles it)

### 4. Determine which swap mode to use
- **Native mode** (`--native`): Both source and destination tokens are on NEAR (`nep141:` assets). Fast (~1 sec), but requires on-chain NEAR transactions (wrapping, storage deposits, ft_transfer_call, withdrawals).
- **Cross-chain mode** (default): Tokens move between different blockchains. Slower (minutes), but the user just opens a signing URL in their browser.
- Some tokens on NEAR have `1cs_v1:` asset IDs — these require cross-chain mode even though they're on NEAR.
- Use `near-intents tokens --chain near --search <symbol>` to check asset ID prefixes.

### 5. Check token availability
- Run `near-intents tokens --search <symbol>` to verify the tokens exist and note their asset IDs
- If a token isn't found, tell the user — don't guess or assume

### 6. Present the plan and get confirmation
- Summarize: what swaps you'll do, what mode (native vs cross-chain), approximate fees, how many steps
- For native swaps: mention wrapping, storage registration, withdrawal steps and their costs
- Wait for the user to say yes before doing anything

## Before Any Swap

Before constructing a swap, establish what the user holds:

1. Run `near-intents balances --account <id>` to see NEAR wallet + intents balances
2. Intents balances are immediately swappable via `--native` mode
3. Wallet balances need deposit steps first (wrap NEAR, storage registration, etc.)
4. The output includes `assetId` fields that feed directly into `--from` / `--to` flags

If the user's NEAR account ID is not known, ask for it before running balances. Common patterns: `username.near`, `username.tg` (Telegram-linked). Do not guess.

For full cross-chain portfolio visibility, use the `portfolio` tool:
```
portfolio balances
```

## Setup & Authentication

Authentication is via JWT bearer token, obtained from the Partner Dashboard at https://partners.near-intents.org/

Set your token (pick one):
- Flag: `near-intents --jwt <token> <command>`
- Environment variable: `NEAR_INTENTS_JWT_TOKEN=<jwt>`
- Config file: `~/.near-intents.json` → `{"token": "<jwt>"}`

**Without a token, swaps still work** but incur a platform fee. All commands function without authentication.

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

**near-cli command structure:** all on-chain calls end with `sign-as <account> network-config mainnet sign-with-keychain send`. The `sign-with-keychain` picks up credentials from `~/.near-credentials/mainnet/`. The `send` at the end is what makes the call non-interactive — omitting it causes near-cli to prompt for confirmation, which fails in non-TTY agent environments.

```
# 0. Check for existing NEAR account credentials
ls ~/.near-credentials/mainnet/
# Files are named <account>.json — confirm with the user which account to use

# 1. Wrap NEAR → wNEAR
#    Replace 10 with the amount to wrap. First call also registers storage (~0.00125 NEAR).
near contract call-function as-transaction wrap.near near_deposit \
  json-args '{}' \
  prepaid-gas '30.0 Tgas' attached-deposit '10 NEAR' \
  sign-as alice.near network-config mainnet sign-with-keychain send

# 2. Register storage on destination token contract (~0.0125 NEAR, one-time per token)
near contract call-function as-transaction usdc.omft.near storage_deposit \
  json-args '{"account_id": "alice.near"}' \
  prepaid-gas '30.0 Tgas' attached-deposit '0.0125 NEAR' \
  sign-as alice.near network-config mainnet sign-with-keychain send

# 3. Preview the rate
near-intents quote --native --from wNEAR --to USDC --amount 10

# 4. Execute the swap — get back a nearTransaction object
near-intents swap --native --from wNEAR --to USDC --amount 10 \
  --recipient alice.near --refund-to alice.near --sender alice.near
# Response includes: depositAddress, nearTransaction.receiverId, nearTransaction.actions[].ft_transfer_call

# 5. Submit the ft_transfer_call from the nearTransaction response
#    Use the exact receiver_id, amount, and msg from the swap response — do not modify them
near contract call-function as-transaction wrap.near ft_transfer_call \
  json-args '{"receiver_id":"intents.near","amount":"<amount_from_response>","msg":"<msg_from_response>"}' \
  prepaid-gas '100.0 Tgas' attached-deposit '1 yoctoNEAR' \
  sign-as alice.near network-config mainnet sign-with-keychain send

# 6. Submit tx hash to speed up processing
near-intents submit-tx --deposit-address <addr> --tx-hash <hash> --near-sender alice.near

# 7. Poll status until SUCCESS
near-intents status --deposit-address <addr>

# 8. Withdraw output tokens from intents.near to wallet
#    Strip the nep141: prefix from the token contract ID
near contract call-function as-transaction intents.near ft_withdraw \
  json-args '{"token":"usdc.omft.near","amount":"<raw_amount>","receiver_id":"alice.near"}' \
  prepaid-gas '100.0 Tgas' attached-deposit '1 yoctoNEAR' \
  sign-as alice.near network-config mainnet sign-with-keychain send

# 9. If chaining another swap: repeat from step 2
```

**Executing multiple swaps:** Run ft_transfer_call transactions sequentially, not in parallel. They share the same signer nonce — parallel execution will cause nonce conflicts and failed transactions.

For full details on native swaps, wrapped assets, ft_transfer_call, withdrawals, and chaining, run: `near-intents llm topic native-swaps`

## Commands Reference

### `near-intents tokens`
List/search supported tokens. Filtering is client-side.

| Flag | Description |
|------|-------------|
| `--chain <name>` | Filter by blockchain (ethereum, solana, near, base, arbitrum, etc.) |
| `--search <term>` | Match on symbol, blockchain, or asset ID |

Returns: `[{assetId, symbol, blockchain, decimals, price, contractAddress}, ...]`

### `near-intents balances`
Show NEAR wallet and intents balances for an account.

| Flag | Required | Description |
|------|----------|-------------|
| `--account` | Yes | NEAR account ID (e.g., `alice.near`) |

Returns: `{totalUsd, chains: [{chain, address, totalUsd, tokens: [{symbol, balance, usd, contractAddress?, assetId?}]}]}`

Chains returned: `near` (native + FTs) and `near-intents` (tokens deposited in intents.near).

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

## Swapping a Full Balance

**This will fail if you pass the full human-readable balance as `--amount`.** The CLI converts your decimal amount to raw integer units, and rounding means the resulting raw amount will exceed the actual on-chain balance by a few yocto — the transaction will be rejected with "not enough balance" or "Insufficient sender balance".

Before swapping a full token balance:

1. Query the exact raw balance on-chain:
```
near contract call-function as-read-only wrap.near ft_balance_of \
  json-args '{"account_id": "alice.near"}' \
  network-config mainnet now
# Returns: "895665650336104516539012"
```

2. Convert to human-readable and **round down** (truncate, don't round):
   - Raw: `895665650336104516539012`
   - Decimals: 24 (wNEAR)
   - Human: `0.895665` (truncate to 6 decimal places to be safe)

3. Use the truncated amount: `--amount 0.895665`

This applies to any FT balance swap — wNEAR, USDC, bridged tokens. When in doubt, subtract a small buffer (e.g., 0.0001) from the human-readable amount.

## Common Agent Mistakes

These errors were observed during real agent testing. Read these BEFORE attempting swaps.

### Wrong flag names

| Wrong | Correct | Context |
|-------|---------|---------|
| `--account` | `--sender` | NEAR account signing the transaction (swap --native) |
| `--amount-side from` | (drop it) | `--swap-type EXACT_INPUT` is the default, no extra flag needed |
| `--correlation-id` | `--deposit-address` | Use the deposit address from swap output for status checks |

### near-cli command structure mistakes

| Mistake | Fix |
|---------|-----|
| `network-config mainnet send` | Must be `network-config mainnet sign-with-keychain send` — missing the signing method |
| `sign-with-keychain` without `send` | Causes near-cli to prompt interactively, which fails in non-TTY shells. Always end with `send` |
| Parallel ft_transfer_call transactions | Run sequentially — they share the signer nonce and will conflict if run in parallel |
| Swapping the full human-readable balance | Query `ft_balance_of` first and truncate the result — see "Swapping a Full Balance" above |

### Missing required swap flags

`near-intents swap` requires ALL THREE for native swaps:
- `--sender` — NEAR account signing the tx
- `--recipient` — where output tokens go
- `--refund-to` — where to refund on failure

For self-swaps, set all three to the same account.

### Withdrawal is NOT a CLI command

There is no `near-intents withdraw`. After a native swap, output tokens are inside `intents.near`. To move them to the wallet, call `ft_withdraw` on the `intents.near` contract directly using near-cli:

```
near contract call-function as-transaction intents.near ft_withdraw \
  json-args '{"token":"wrap.near","amount":"<raw_amount>","receiver_id":"alice.near"}' \
  prepaid-gas '100.0 Tgas' attached-deposit '1 yoctoNEAR' \
  sign-as alice.near network-config mainnet sign-with-keychain send
```

**Critical details:**
- Use `ft_withdraw` for NEP-141 tokens (wNEAR, USDC, ETH bridges, etc.) — this is almost all tokens
- Use `mt_withdraw` ONLY for NEP-245 multi-token contracts (rare)
- The `token` field is the **bare contract ID** (e.g., `wrap.near`), NOT the `nep141:`-prefixed asset ID
- Amount is the raw integer string (e.g., `"846915650336104516539012"`), not human-readable

### The nep141: prefix trap

The `nep141:` prefix appears in asset IDs throughout the system but must be STRIPPED when:
- Calling `ft_withdraw` on `intents.near` (the `token` field is a bare account ID)
- Calling `ft_balance_of` on token contracts

It must be KEPT when:
- Passing `--from` / `--to` to `near-intents quote` and `swap`
- Querying `mt_batch_balance_of` on `intents.near`

## Deep-dive topics

For detailed knowledge on specific topics, run `near-intents llm topics` to see what's available, then `near-intents llm topic <name>` to read a topic.
