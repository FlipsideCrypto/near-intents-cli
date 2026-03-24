---
name: near-intents-trading
description: Use when user asks about crypto trading, swaps, portfolio rebalancing, token balances, or managing holdings across NEAR, Ethereum, Solana, Bitcoin, or other chains. Triggers on keywords like swap, rebalance, portfolio, balance, holdings, trade, DeFi.
metadata:
  version: "0.1.9"
---

# NEAR Intents Trading

Orchestrate crypto portfolio management and cross-chain swaps using two CLI tools.

## Install

If `near-intents` or `portfolio` are not installed:

### macOS / Linux

```
curl -fsSL https://raw.githubusercontent.com/FlipsideCrypto/near-intents-cli/main/install.sh | sh
```

Or with a specific version: `VERSION=v0.1.0 curl -fsSL ... | sh`

Custom install dir: `INSTALL_DIR=~/.local/bin curl -fsSL ... | sh`

### Windows

**Option A — Git Bash / MSYS2 (recommended):**

The install script works in Git Bash or MSYS2. Open Git Bash and run:

```
curl -fsSL https://raw.githubusercontent.com/FlipsideCrypto/near-intents-cli/main/install.sh | sh
```

Binaries install to `~/.local/bin`. If that's not in your PATH, add it:

```
echo 'export PATH="$PATH:$HOME/.local/bin"' >> ~/.bashrc && source ~/.bashrc
```

**Option B — Manual download (PowerShell / cmd):**

1. Go to the [latest release](https://github.com/FlipsideCrypto/near-intents-cli/releases/latest)
2. Download `near-intents_<version>_windows_amd64.zip` and `portfolio_<version>_windows_amd64.zip`
3. Extract both `.zip` files
4. Move `near-intents.exe` and `portfolio.exe` to a directory in your PATH (e.g. `C:\Users\<you>\bin`)
5. Add that directory to your PATH if it isn't already:
   - Search "Environment Variables" in Start → Edit the user `Path` variable → Add the directory

Verify: open a new terminal and run `near-intents --version`

## Tools

| Tool | Purpose | Onboard command |
|------|---------|-----------------|
| `portfolio` | Read balances across all chains | `portfolio llm onboard` |
| `near-intents` | Execute swaps + get intel | `near-intents llm onboard` |

**You MUST run both onboard commands before any other action in a session — no exceptions.** Do not attempt to guess command names, flag names, or asset ID formats. Every mistake that wastes round trips (wrong flags, wrong token format, unknown commands) is documented in the onboard output. Running it takes seconds; skipping it costs minutes of failed attempts.

```
near-intents llm onboard
portfolio llm onboard
```

Run these immediately after updating. Do not proceed until you have read the output.

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

2. **Read the onboard docs** — run both of these and read the full output before proceeding:
   ```
   near-intents llm onboard
   portfolio llm onboard
   ```
   This is not optional. The onboard output contains exact command syntax, required flags, asset ID formats, and common mistakes. Skipping it and guessing will waste time.

3. Run `portfolio setup --list` — are addresses configured? If not, ask the user for their wallet addresses and add them.
4. Run `portfolio balances` (or `near-intents balances --account <id>` for NEAR-only) — establish current holdings.
5. If the user wants recommendations, summarize the balances and pass to `near-intents intel --message "Here's my portfolio: [summary]. How should I rebalance?"`.

## New User Detection

Run `portfolio setup --list` at the start of every session. If it's empty:

1. Ask: "What chains do you hold crypto on?" and "Do you have a NEAR account?"
2. No NEAR account → run `near-intents llm topic new-account` and follow it
3. No NEAR account, no other crypto → they need an exchange first
4. Add all addresses: `portfolio setup --add --chain <near|evm|solana|bitcoin> --address <addr>`
5. Collect API keys upfront: Flipside (`~/.near-intents.json`), Ankr (`~/.portfolio.json`)
6. Verify: `portfolio balances` should return data before starting any swaps

Do setup upfront — discovering missing config mid-swap wastes time.

## Command Quick Reference

| Command | Required flags | Common optional flags |
|---------|---------------|----------------------|
| `tokens` | — | `--search <term>`, `--chain <chain>` |
| `balances` | `--account <near_id>` | `--pretty` |
| `quote` | `--from`, `--to`, `--amount` | `--from-chain`, `--to-chain`, `--native`, `--slippage` |
| `swap` | `--from`, `--to`, `--amount`, `--recipient`, `--refund-to` | `--from-chain`, `--to-chain`, `--native`, `--sender` (required with --native), `--deadline` |
| `submit-tx` | `--deposit-address`, `--tx-hash` | `--near-sender` |
| `status` | `--deposit-address` | `--deposit-memo` |
| `intel` | `--message "<text>"` | `--flipside-api-key`, `--agent` |

## Key Concepts

- **Signing URL is the default.** Most swaps should use the cross-chain signing URL flow — user gets a link, opens it, connects wallet, signs. No near-cli needed, no wrapping, no storage deposits. Only use native mode if the user explicitly asks for it.
- **Ask the user, don't assume.** Present both options (signing URL vs native CLI) and let them choose. Don't probe for near-cli or check `~/.near-credentials/` unless the user wants native mode.
- **`assetId` fields** in balance output feed directly into `near-intents swap --from` / `--to`.
- **Flipside intel** is for analytical recommendations ("how should I rebalance?"), not balance lookups (use `portfolio balances` for that).
- **intents vs wallet**: Tokens in `near-intents` chain are immediately swappable. Tokens in wallet may need extra steps depending on mode.
- **No withdraw CLI command.** After native swaps, tokens land in intents.near. Withdraw via `ft_withdraw` on `intents.near` using near-cli directly. See onboard docs for exact syntax. (Only relevant in native mode.)

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
| `swap --refund <addr>` | Flag is `--refund-to`, not `--refund` |
| `status --correlation-id <id>` | Flag is `--deposit-address`, not `--correlation-id` |
| `intel --account <acct>` | Intel has no `--account` flag — pass account context inside `--message` |
| Native swap: missing deposit address storage registration | After getting the deposit address, register it on the token contract (`storage_deposit`) before ft_transfer_call — see `llm topic native-swaps` |
| Suggesting `brew install near-cli-rs` | That formula doesn't exist. Use `npm install -g near-cli-rs` |
| Assuming new user already has a NEAR account | Check `portfolio setup --list` first — if empty, run new user detection flow |
