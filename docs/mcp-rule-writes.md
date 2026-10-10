# MCP category and rule writes

## Scope

Extend the existing local MCP with `save_category` to create/update custom category
definitions and `save_categorization_rule` to create/update auto-categorization
rules targeting a category. Existing read tools retain their behavior.
Rule writes do not re-apply rules or edit existing transactions. Users apply rules
to the ledger through the app. No delete, import, transaction-edit, or security
management tool is added.

## Design

- Keep the existing Manager, Streamable HTTP transport, shared bearer token,
  local management protection, database writer lock, and rule matching engine.
- Add `allow_categorization_writes` to MCP settings with sequential migration 00016. Default
  false on fresh and upgraded databases. The Settings panel exposes this opt-in
  independently of MCP enablement. It applies to every client sharing the token.
  Omitted permission in older settings requests preserves the saved preference.
- Select the read-only or categorization-write MCP server using the access policy captured
  by Manager after authentication. Permission changes drain active requests using
  the existing lifecycle. Read-only clients cannot discover or invoke the write
  tools. App info describes the granted access accurately. Reset/restore clears
  the permission alongside MCP credentials.
- `save_category` accepts required `name`, optional `id` (create if absent),
  `color_hex` (#RRGGBB), and `icon`. Reuse DB.CreateCategory and add DB.UpdateCategory.
  Creation defaults to #64748B and tag; updates preserve omitted fields, identity,
  parent, and references from rules and transactions. Write only supplied columns
  under the database writer lock, preserving concurrent disjoint patches. System categories are
  protected, consistent with the existing custom-category management UI. Category
  hierarchy editing and deletion remain outside this feature's scope.
- The tool interface accepts optional `id` (absent creates, present updates),
  required `match_pattern` and `target_category_id`, and existing rule fields:
  `match_field`, `match_type`, `exclude_pattern`, `tx_type`, `priority`,
  `assign_tags`, and `is_active`. Defaults on creation match the app: cleaned
  payee, CONTAINS, ALL, priority 50, active true. Updates patch supplied fields
  while preserving omitted ones, including disabled rules.
- Validate a closed, bounded schema, supported matcher and transaction types,
  nonempty patterns and identifiers, regex syntax, category existence, and update
  target existence. Share rule insertion with DB.CreateRule via DB.InsertRule,
  preserving explicit zero priority while keeping the app's historical defaults.
  DB.PatchRule atomically writes only supplied fields and checks target existence.
  Add DB.GetRule to load a rule by id regardless of active state. Errors contain
  no SQL or credentials.
- Return the saved rule in the existing data envelope with a text fallback.
  Advertise the tool as a local write, non-destructive, and non-idempotent because
  creation generates a new id. Document that retrying creation can duplicate a
  rule, and that updates affect future imports without rewriting past data.

## Verification and review

Use disposable databases and fictional fixtures. Exercise real SDK discovery and
calls for permission enforcement, validation, create/update/disabled-rule parity,
ledger immutability, persistence, token rotation, and reset/restore revocation.
Cover backward-compatible settings updates and migration from the prior schema.
Run CGO-disabled Go tests and portable builds, plus pnpm frontend checks/build.
Review the committed diff against main with independent Standards and Spec agents,
fix warranted findings, then open a PR. This document and the user's stated scope
are the review spec; AGENTS.md and CONTRIBUTING.md are the repo standards.
