# Splitwise imports and personal spending

Use **Statement Import → Splitwise**. Enter a stable group name and your member name (defaults to Sanjay), choose a group CSV export, and preview before importing. First names resolve only when exactly one member matches; use the full CSV column name if necessary. All data stays local. The fictional example is `samples/splitwise/fictional-group.csv`.

Imported rows initially need review. Select **Review**, check your expense share, find statement candidates if you paid, and confirm the correct match. When someone else paid, confirm your share without a statement link and choose an expense category. Linked expenses keep the bank transaction's category, notes and tags. **Ignore entry** and **Undo confirmation** restore the original statement contribution. Import statements at any time and return to review unmatched entries.

## Accounting rules

A member's CSV column is their net balance change: `paid − owed`. It is not their expense share. The group export omits actual paid amounts, so it cannot uniquely describe multiple payers, including a member whose net balance is zero. The initial suggestions assume a single payer and always require explicit confirmation:

```text
net < 0: suggested share = −net (someone else paid)
net > 0: suggested share = group cost − net (you paid the full cost)
net = 0: suggested share = 0; review for equal paid and owed
confirmed paid = confirmed share + net
```

An expense with a nonzero paid amount requires a bank debit of that amount. Candidate amounts must match in paise, use INR accounts, fall within three days, and be eligible for spending. Equal amounts and nearby dates only suggest a match; they never confirm one automatically. One statement transaction can match one Splitwise row. Cash payments, combined bank payments covering multiple rows, refunds/negative-cost rows and non-INR exports currently require separate handling and are not automatically reconciled.

Rows with category `Payment` are settlements: a positive net means you paid (bank debit), and a negative net means you received (bank credit). Once linked, neither contributes to personal income or expense. The original expense stays on its original date; repayment does not create a second expense. The total-balance footer and blank lines are not transactions.

## Low-level design

`parser.ParseSplitwise(reader, group, person)` is a pure parser independent of bank account identity and the statement registry. A Splitwise export is a debt ledger, so registering it as a bank statement would invent liquidity and corrupt bank balances. The parser validates the entire export before persistence, uses integer paise and returns reviewable entries.

`db.ImportSplitwise` atomically inserts entries without overwriting confirmed shares, categories or links. Content fingerprints include group, member, date, description, category, cost and member balances. An occurrence counter preserves identical repeated expenses within an export. Use the same group name for overlapping reimports. An edited Splitwise row is a new review entry; ignore its older confirmed version before confirming the replacement. Renaming groups/members or exporting only a subset of identical repeated expenses cannot establish stable identity because this CSV format has no transaction IDs.

`db.ConfirmSplitwise` validates shares, amount, direction, account currency, date, eligibility and link uniqueness inside one SQLite transaction. `personal_transactions` is a read-only SQL projection used for overview, budgets, monthly review/evidence, merchants, cash-flow intelligence and annual wrapped spending. It replaces the amount of a confirmed linked expense with your share, excludes linked settlements, and adds confirmed expenses paid by others without creating bank accounts or bank transactions. Card rewards and fee-waiver progress continue to use gross statement spending.

The cash ledger remains the source for balances, transaction lists, hashes, statement reimports and CSV cash exports. The Splitwise screen exposes its confirmation state and links; JSON exports and SQLite backups include the separate ledger. Reset clears it. Schema migration 17 is embedded and reversible; undoing it removes Splitwise data and restores original statement analytics.

## Verification

Fictional fixtures exercise both payer directions, settlements, zero net and multiple payers. Parser tests cover validation and duplicate identity; database tests cover personal totals, monthly evidence, cash-flow totals, preservation of bank values/manual edits, reimports, undo, uniqueness, reset and backup/restore. API tests cover read-only preview, duplicate import, categories, request limits and local authentication; frontend tests cover protected multipart uploads and confirmation responses.
