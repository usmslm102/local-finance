package db

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"local-finance/internal/models"
)

func (d *DB) ListSplitwiseMembers() ([]models.SplitwiseMember, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.listSplitwiseMembers()
}

func (d *DB) listSplitwiseMembers() ([]models.SplitwiseMember, error) {
	return listSplitwiseMembers(d.conn)
}

func listSplitwiseMembers(conn statementConnection) ([]models.SplitwiseMember, error) {
	groupRows, err := conn.Query(`SELECT group_name,person,member_names FROM splitwise_groups ORDER BY group_name`)
	if err != nil {
		return nil, err
	}
	members := map[string]models.SplitwiseMember{}
	for groupRows.Next() {
		var group, person, names string
		if err := groupRows.Scan(&group, &person, &names); err != nil {
			groupRows.Close()
			return nil, err
		}
		var namesList []string
		if err := json.Unmarshal([]byte(names), &namesList); err != nil {
			groupRows.Close()
			return nil, err
		}
		for _, name := range namesList {
			if strings.EqualFold(name, person) {
				continue
			}
			key := group + "\x00" + name
			members[key] = models.SplitwiseMember{Group: group, Name: name, Person: person, Aliases: []string{}}
		}
	}
	err = groupRows.Err()
	groupRows.Close()
	if err != nil {
		return nil, err
	}
	rows, err := conn.Query(`SELECT group_name,member_name,aliases,pattern FROM splitwise_member_aliases`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var group, name, aliases, pattern string
		if err := rows.Scan(&group, &name, &aliases, &pattern); err != nil {
			return nil, err
		}
		key := group + "\x00" + name
		if m, ok := members[key]; ok {
			m.Pattern = pattern
			if err := json.Unmarshal([]byte(aliases), &m.Aliases); err != nil {
				return nil, err
			}
			members[key] = m
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	result := make([]models.SplitwiseMember, 0, len(members))
	for _, m := range members {
		result = append(result, m)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Group != result[j].Group {
			return result[i].Group < result[j].Group
		}
		return result[i].Name < result[j].Name
	})
	return result, nil
}

func findSplitwiseMember(members []models.SplitwiseMember, group, name string) (models.SplitwiseMember, error) {
	for _, m := range members {
		if m.Group == group && m.Name == name {
			return m, nil
		}
	}
	return models.SplitwiseMember{}, fmt.Errorf("import the group CSV and choose another group member first")
}

func (d *DB) SaveSplitwiseMemberAliases(req models.SplitwiseMember) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	members, err := d.listSplitwiseMembers()
	if err != nil {
		return err
	}
	if _, err := findSplitwiseMember(members, req.Group, req.Name); err != nil {
		return err
	}
	if len(req.Aliases) > 20 {
		return fmt.Errorf("at most 20 aliases per member")
	}
	if len(req.Pattern) > 1000 {
		return fmt.Errorf("regex is limited to 1000 characters")
	}
	if req.Pattern != "" {
		if _, err := regexp.Compile(req.Pattern); err != nil {
			return fmt.Errorf("invalid member regex: %w", err)
		}
	}
	aliases := []string{}
	seen := map[string]bool{}
	for _, alias := range req.Aliases {
		alias = strings.TrimSpace(alias)
		normalized := settlementName(alias)
		if len([]rune(normalized)) < 3 || len(alias) > 200 {
			return fmt.Errorf("aliases must contain 3 to 200 meaningful characters")
		}
		if !seen[normalized] {
			aliases = append(aliases, alias)
			seen[normalized] = true
		}
	}
	data, err := json.Marshal(aliases)
	if err != nil {
		return err
	}
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO splitwise_member_aliases(group_name,member_name,aliases,pattern) VALUES(?,?,?,?) ON CONFLICT(group_name,member_name) DO UPDATE SET aliases=excluded.aliases,pattern=excluded.pattern`, req.Group, req.Name, string(data), req.Pattern)
	if err != nil {
		return err
	}
	if _, err := reconcileSplitwise(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// Whole normalized token sequences avoid matching Ann inside Joanne or a partial VPA.
// This is a suggestion only: even a complete name does not establish a settlement.
func settlementName(value string) string {
	return strings.Join(strings.FieldsFunc(strings.ToLower(value), func(r rune) bool { return !unicode.IsLetter(r) && !unicode.IsDigit(r) }), " ")
}

func matchesSplitwiseMember(t models.Transaction, m models.SplitwiseMember) bool {
	names := append([]string{m.Name}, m.Aliases...)
	fields := []string{t.CleanedPayee, t.RawNarration}
	if t.UPIVPA != nil {
		fields = append(fields, *t.UPIVPA)
	}
	if m.Pattern != "" {
		if re, err := regexp.Compile(m.Pattern); err == nil {
			for _, field := range fields {
				if re.MatchString(field) {
					return true
				}
			}
		}
	}
	for _, name := range names {
		name = settlementName(name)
		if len([]rune(name)) < 3 {
			continue
		}
		for _, field := range fields {
			if strings.Contains(" "+settlementName(field)+" ", " "+name+" ") {
				return true
			}
		}
	}
	return false
}

// Settlements can occur long after an expense. The 15-day purchase matching window
// does not apply here, and a name match never changes spending by itself.
func (d *DB) SplitwiseSettlementCandidates() ([]models.SplitwiseSettlementCandidate, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	members, err := d.listSplitwiseMembers()
	if err != nil {
		return nil, err
	}
	rows, err := d.conn.Query(`SELECT t.id,t.tx_date,t.raw_narration,COALESCE(t.cleaned_payee,''),t.amount,t.tx_type,COALESCE(NULLIF(a.nickname,''),a.bank_name),t.upi_vpa
 FROM transactions t JOIN accounts a ON a.id=t.account_id
 WHERE a.currency='INR' AND t.amount>0 AND t.tx_type IN ('DEBIT','CREDIT') AND t.is_excluded=0 AND COALESCE(t.transfer_peer_id,'')=''
 AND NOT EXISTS(SELECT 1 FROM splitwise_entries s WHERE s.transaction_id=t.id)
 ORDER BY t.tx_date DESC,t.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []models.SplitwiseSettlementCandidate{}
	for rows.Next() {
		var t models.Transaction
		if err := rows.Scan(&t.ID, &t.TxDate, &t.RawNarration, &t.CleanedPayee, &t.Amount, &t.TxType, &t.AccountName, &t.UPIVPA); err != nil {
			return nil, err
		}
		for _, m := range members {
			if matchesSplitwiseMember(t, m) {
				result = append(result, models.SplitwiseSettlementCandidate{Transaction: t, Group: m.Group, Member: m.Name})
				if len(result) == 100 {
					break
				}
			}
		}
		if len(result) >= 100 {
			break
		}
	}
	return result, rows.Err()
}

// Record a manually confirmed settlement in the same ledger as CSV Payment rows.
// Reusing that projection keeps all spending surfaces and bank badges consistent.
func (d *DB) ConfirmSplitwiseSettlement(req models.SplitwiseSettlementConfirmation) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	members, err := d.listSplitwiseMembers()
	if err != nil {
		return err
	}
	member, err := findSplitwiseMember(members, req.Group, req.Member)
	if err != nil {
		return err
	}
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := recordSplitwiseSettlement(tx, req, member); err != nil {
		return err
	}
	return tx.Commit()
}

func recordSplitwiseSettlement(conn statementConnection, req models.SplitwiseSettlementConfirmation, member models.SplitwiseMember) error {
	var amount float64
	var date, narration, direction string
	var eligible bool
	err := conn.QueryRow(`SELECT t.amount,t.tx_date,t.raw_narration,t.tx_type,(a.currency='INR' AND t.is_excluded=0 AND COALESCE(t.transfer_peer_id,'')='') FROM transactions t JOIN accounts a ON a.id=t.account_id WHERE t.id=?`, req.TransactionID).Scan(&amount, &date, &narration, &direction, &eligible)
	if err != nil {
		return fmt.Errorf("statement transaction not found: %w", err)
	}
	if !eligible || amount <= 0 || amount > 1_000_000_000 || (direction != "DEBIT" && direction != "CREDIT") {
		return fmt.Errorf("choose an INR bank payment that is not excluded or paired as an internal transfer")
	}
	var linked bool
	if err := conn.QueryRow(`SELECT EXISTS(SELECT 1 FROM splitwise_entries WHERE transaction_id=?)`, req.TransactionID).Scan(&linked); err != nil {
		return err
	}
	if linked {
		return fmt.Errorf("statement transaction is already linked to Splitwise")
	}
	cents := int64(math.Round(amount * 100))
	if cents <= 0 {
		return fmt.Errorf("settlement amount must be at least one paise")
	}
	net := cents
	if direction == "CREDIT" {
		net = -net
	}
	id := "sw-settlement-" + req.TransactionID
	_, err = conn.Exec(`INSERT INTO splitwise_entries(id,group_name,person,tx_date,description,category,cost_cents,net_cents,share_cents,kind,status,transaction_id)
 VALUES(?,?,?,?,?,'Payment',?,?,0,'PAYMENT','CONFIRMED',?)
 ON CONFLICT(id) DO UPDATE SET group_name=excluded.group_name,person=excluded.person,tx_date=excluded.tx_date,description=excluded.description,cost_cents=excluded.cost_cents,net_cents=excluded.net_cents,status='CONFIRMED',transaction_id=excluded.transaction_id`, id, req.Group, member.Person, strings.Split(date, "T")[0], "Settlement with "+member.Name+": "+narration, cents, net, req.TransactionID)
	if err != nil {
		return err
	}
	return nil
}
