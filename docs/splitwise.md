# Splitwise imports and personal spending

Open **Splitwise** in the sidebar, or **Statement Import → Splitwise**. Enter a stable group name and your member name (defaults to Sanjay), then import your group CSV. Preview is optional; no per-transaction review is required. All data stays local. The fictional export is `samples/splitwise/fictional-group.csv`.

## Automatic accounting

The CSV stores each member's net balance change, `paid − owed`, rather than separate paid and owed amounts. The importer uses the single-payer interpretation:

```text
net < 0: your share = −net, you paid nothing
net > 0: your share = total − net, you paid the total
net = 0: no identifiable personal payment or share; skip
```

Other people's expenses are skipped when both your identifiable payment and your share are zero. An expense paid entirely by you for somebody else still maps to your bank debit and has zero personal spending. Multiple payers with equal paid and owed amounts cannot be identified from this export; such zero-net rows are skipped. This is a limitation of the export's information.

Expenses you paid require an INR bank debit within ₹5 of the derived paid amount. The matcher searches the entry date and preceding 15 days, prioritizing exact amounts, then nearest dates, then smallest amount difference. It consumes each bank movement only once and protects excluded movements and paired internal transfers. Unmatched cash-backed entries are dropped, without changing bank accounting. Re-import the CSV after importing missing bank statements if you want to try those entries again.

Your shares of expenses paid by someone else appear as accountless expense rows in **Transactions**. They do not require a bank match and never create cash or change account balances. New Splitwise expenses start in **Others & Uncategorized** and use the same priority, transaction type, exception, field and pattern matching as normal category rules. **Settings → Reapply rules** updates them too. Categories, notes and tags can be edited in Transactions; manual edits survive rules and re-imports. Linked bank expenses preserve manually assigned bank categories and original cash details, and expose the Splitwise description alongside the original narration.

CSV Payment rows are settlements, never additional personal expenses or income. A ₹50 taxi share paid by Asha counts as ₹50 spending. A later ₹50 repayment to Asha changes cash but adds no spending.

## Automatic member transfers

The CSV headers populate group members. Complete member names in bank payee, narration or UPI address identify outgoing payments as transfers. Add an optional per-member regex under **Automatic member transfers**, then select **Save & apply** to classify existing matching outgoing payments. The same matching runs automatically inside future bank statement imports. Use `(?i)` for case-insensitive matching, for example `(?i)asha@fictional` for a UPI address. Invalid regexes are rejected without replacing the saved rule.

Only outgoing member payments are automatically self transfers. Incoming payments remain unchanged unless explicitly identified by a matching CSV Payment row. Repayments may happen long after the expense, so member transfer classification has no 15-day limit. Explicit Splitwise expense/payment matches take precedence over generic member patterns. A bank movement already linked to an expense or settlement is protected from additional classification. Name/regex matching reflects the user's instruction that otherwise-unlinked payments to group members are repayments; use specific patterns for the intended people.

## Mappings and removal

The dedicated **Splitwise** page shows all applied expenses and settlements, each matched bank date, account, narration and original amount, and your personal expense amount. Entries paid by others explicitly say they have no bank movement. Search and pagination make older mappings accessible.

Select **Remove** to discard an incorrect mapping or expense. Removing a bank match restores the original bank transaction's spending treatment. Removing an accountless expense removes that expense from totals. An ignored entry and its former bank ID remain as suppression metadata, so subsequent CSV imports, saved regexes and bank imports do not recreate the removed mapping automatically.

## Design

Splitwise is a separate ledger rather than a bank parser. Parsing validates the whole CSV, uses integer paise, and retains member identities. Fingerprints include group/member, date, description, category, cost and nonzero member balances; occurrence numbers preserve identical repeated expenses. Import is idempotent for unchanged retained entries and refreshes member metadata without changing confirmations, aliases or manual edits. Group metadata survives even when all transaction rows are skipped.

`db.ImportSplitwiseAutomatically` inserts and reconciles in one SQLite transaction. Bank imports reconcile through `StatementWriter` inside their existing transaction. Saving member regexes applies reconciliation atomically. Candidate ordering and paid amount validation are shared with the existing confirmation path; category matching is shared with the normal bank importer and rule reapplication.

`personal_transactions` projects confirmed personal spending for analytics, budgets, monthly evidence, merchants, cash-flow intelligence and Wrapped. `ledger_transactions` projects original bank cash amounts alongside accountless Splitwise expenses for the regular Transactions UI. Original bank hashes, amounts, balances, notes and tags remain in `transactions`. Bank CSV exports and bank records in JSON exports retain the original statement ledger. Card rewards and fee-waiver progress continue to use gross bank spending.

JSON exports and SQLite backups include the Splitwise ledger and member settings. Reset clears both. Embedded migrations 17 and 18 are reversible. Downgrading 18 removes member configuration and the extended UI projection; downgrading 17 removes Splitwise accounting data without removing bank transactions.

CSV exports lack stable expense IDs. Edited rows, renamed groups/members, and subsets of identical repeated expenses cannot be correlated reliably across exports. Remove an older applied entry before importing a changed replacement. The automatic share interpretation cannot recover multiple payer amounts absent from the source.

## Verification

Fictional fixtures and regression tests cover automatic paid-by-other expenses, exact/nearest matching, 15-day boundaries, ₹5 tolerance, zero personal spending, uninvolved rows, outgoing-only regex transfers, future bank imports, regular rules/manual editing, mapping visibility, persistent removal, bank preservation, reset and backup/restore. API tests cover protected upload/preview, automatic results, member settings, malformed regexes and local authentication. Frontend checks cover typechecking/build, lint and protected requests.
