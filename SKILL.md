---
name: near-intents-trading
description: Use when user asks about crypto trading, swaps, portfolio rebalancing, token balances, or managing holdings across NEAR, Ethereum, Solana, Bitcoin, or other chains. Triggers on keywords like swap, rebalance, portfolio, balance, holdings, trade, DeFi.
---

# NEAR Intents Trading

Orchestrate crypto portfolio management and cross-chain swaps using two CLI tools.

## Install

If `near-intents` or `portfolio` are not installed, run:

```
curl -fsSL https://raw.githubusercontent.com/FlipsideCrypto/near-intents-cli/main/install.sh | sh
```

Or with a specific version: `VERSION=v0.1.0 curl -fsSL ... | sh`

Custom install dir: `INSTALL_DIR=~/.local/bin curl -fsSL ... | sh`

## Tools

| Tool | Purpose | Onboard command |
|------|---------|-----------------|
| `portfolio` | Read balances across all chains | `portfolio llm onboard` |
| `near-intents` | Execute swaps + get intel | `near-intents llm onboard` |

**Run the onboard command for each tool BEFORE your first use.** The onboard output contains the full flag reference, required vs optional flags, and critical gotchas (wrapping, storage deposits, withdrawal steps). Do not guess at syntax — flags like `--sender`, `--recipient`, `--refund-to` are required for swaps and not obvious.

## The Loop

```
OBSERVE  → portfolio balances / near-intents balances
DECIDE   → near-intents intel (feed it the portfolio summary)
PLAN     → near-intents quote (price out each swap)
CONFIRM  → present plan to user with fees and steps
EXECUTE  → near-intents swap + submit-tx
VERIFY   → near-intents status (poll until terminal)
```

Never skip OBSERVE. Never skip CONFIRM.

## Before Anything

1. **Update both tools** — always run this first to ensure you have the latest version:
   ```
   near-intents update && portfolio update
   ```
   If either binary is missing, install first (see Install above), then update.
2. Run `portfolio setup --list` — are addresses configured? If not, ask the user for their wallet addresses and add them.
3. Run `portfolio balances` (or `near-intents balances --account <id>` for NEAR-only) — establish current holdings.
4. If the user wants recommendations, summarize the balances and pass to `near-intents intel --message "Here's my portfolio: [summary]. How should I rebalance?"`.

## Key Concepts

- **intents vs wallet**: Tokens in `near-intents` chain are immediately swappable via native mode. Tokens in wallet need deposit steps first (wrapping, storage registration).
- **`assetId` fields** in balance output feed directly into `near-intents swap --from` / `--to`.
- **Two swap modes**: native (NEAR-only, fast, needs near-cli) and cross-chain (any chain, uses signing URL in browser).
- **Flipside intel** is for analytical recommendations ("how should I rebalance?"), not balance lookups (use `portfolio balances` for that).
- **No withdraw CLI command.** After native swaps, tokens land in intents.near. Withdraw via `ft_withdraw` on `intents.near` using near-cli directly. See onboard docs for exact syntax.

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Guessing flag names (--account, --correlation-id) | Run `llm onboard` — correct flags are --sender, --deposit-address, etc. |
| Querying balances without setup | Check `portfolio setup --list` first |
| Advising on rebalancing yourself | Use `near-intents intel` for recommendations |
| Executing swaps without confirmation | Always present plan with fees, wait for approval |
| Assuming all tokens are in wallet | Check intents balance separately — it's a different "chain" in the output |
| Trying `near-intents withdraw` | No such command. Use `ft_withdraw` on `intents.near` via near-cli directly |
| Using `nep141:` prefix in withdrawal args | Strip it — `ft_withdraw` takes bare contract ID (e.g., `wrap.near` not `nep141:wrap.near`) |
| Using `mt_withdraw` for standard tokens | Use `ft_withdraw` for NEP-141 tokens (wNEAR, USDC, etc.) — `mt_withdraw` is for NEP-245 only |
| "Send X to Y" = buy X | It means user **has X**, deliver to Y. Confirm direction before quoting: "You're sending [A], receiving [B] at [address] — right?" |
| Searching tokens on one chain only | Search all chains first (`--search BTC`), then choose the best route — native chain beats bridged |
| Defaulting to native swap mode | Default to cross-chain (signingUrl) unless user confirms near-cli is set up |
| Quoting cross-chain swap without refund address | For non-NEAR source chains, ask for a refund address on that chain before calling swap |
| Trying bridged token before native chain | Try native chain version first (BTC on bitcoin > wBTC on NEAR). Fall back if quote fails. |
