---
name: near-intents-trading
description: Use when user asks about crypto trading, swaps, portfolio rebalancing, token balances, or managing holdings across NEAR, Ethereum, Solana, Bitcoin, or other chains. Triggers on keywords like swap, rebalance, portfolio, balance, holdings, trade, DeFi.
---

# NEAR Intents Trading

Orchestrate crypto portfolio management and cross-chain swaps using two CLI tools.

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

1. Run `portfolio setup --list` — are addresses configured? If not, ask the user for their wallet addresses and add them.
2. Run `portfolio balances` (or `near-intents balances --account <id>` for NEAR-only) — establish current holdings.
3. If the user wants recommendations, summarize the balances and pass to `near-intents intel --message "Here's my portfolio: [summary]. How should I rebalance?"`.

## Key Concepts

- **intents vs wallet**: Tokens in `near-intents` chain are immediately swappable via native mode. Tokens in wallet need deposit steps first (wrapping, storage registration).
- **`assetId` fields** in balance output feed directly into `near-intents swap --from` / `--to`.
- **Two swap modes**: native (NEAR-only, fast, needs near-cli) and cross-chain (any chain, uses signing URL in browser).
- **Flipside intel** is for analytical recommendations ("how should I rebalance?"), not balance lookups (use `portfolio balances` for that).

## Common Mistakes

| Mistake | Fix |
|---------|-----|
| Guessing command syntax | Run `llm onboard` for each tool first |
| Querying balances without setup | Check `portfolio setup --list` first |
| Advising on rebalancing yourself | Use `near-intents intel` for recommendations |
| Executing swaps without confirmation | Always present plan with fees, wait for approval |
| Assuming all tokens are in wallet | Check intents balance separately — it's a different "chain" in the output |
