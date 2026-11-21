package db

// A repeated upload is evidence of the same source version, not a newer amended
// statement. Keep every import's bill/ownership, but order distinct source hashes
// by their first arrival within the card's cycle. Legacy bills without a source
// hash remain independent versions. Compute this once, not once per archive row.
const billVersions = `WITH bill_sources AS (
 SELECT b.*,CASE WHEN COALESCE(s.file_hash,'')!='' THEN 'hash:'||s.file_hash ELSE 'bill:'||b.id END source_key
 FROM credit_card_bills b LEFT JOIN statement_imports s ON s.id=b.statement_import_id
), bill_versions AS (
 SELECT *,MIN(created_at) OVER (PARTITION BY account_id,statement_date,source_key) version_first_seen FROM bill_sources
), cycle_versions AS (
 SELECT *,ROW_NUMBER() OVER (PARTITION BY account_id,statement_date
 ORDER BY version_first_seen DESC,source_key DESC,created_at DESC,id DESC) cycle_rank FROM bill_versions
) `
