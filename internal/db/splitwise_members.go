package db

import (
	"encoding/json"
	"fmt"
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
	rows.Close()
	// Suggestions come from actual CSV Payment links, never arbitrary bank names.
	paymentRows, err := conn.Query(`SELECT s.id,s.group_name,s.person,s.description,s.counterparty,t.raw_narration,COALESCE(t.cleaned_payee,''),COALESCE(t.upi_vpa,'') FROM splitwise_entries s JOIN transactions t ON t.id=s.transaction_id WHERE s.kind='PAYMENT' AND s.status='CONFIRMED' AND s.id NOT LIKE 'sw-settlement-%' ORDER BY t.tx_date DESC,t.id`)
	if err != nil {
		return nil, err
	}
	for paymentRows.Next() {
		var id, group, person, description, name, narration, payee, vpa string
		if err := paymentRows.Scan(&id, &group, &person, &description, &name, &narration, &payee, &vpa); err != nil {
			paymentRows.Close()
			return nil, err
		}
		if name == "" {
			memberList := make([]models.SplitwiseMember, 0, len(members))
			for _, member := range members {
				memberList = append(memberList, member)
			}
			other, err := paymentMember(models.SplitwiseEntry{ID: id, Group: group, Person: person, Description: description}, memberList)
			if err != nil {
				continue
			}
			name = other.Name
		}
		key := group + "\x00" + name
		m, ok := members[key]
		if !ok {
			continue
		}
		m.MatchedPayments++
		identifier := strings.TrimSpace(vpa)
		if identifier == "" {
			identifier = strings.TrimSpace(payee)
		}
		if identifier == "" {
			identifier = strings.TrimSpace(narration)
		}
		pattern := "(?i)^" + regexp.QuoteMeta(identifier) + "$"
		if m.SuggestedPattern == "" && identifier != "" && len(pattern) <= 1000 {
			m.SuggestedPattern = pattern
		}
		members[key] = m
	}
	err = paymentRows.Err()
	paymentRows.Close()
	if err != nil {
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
// A complete name identifies a participant; a CSV Payment must establish settlement evidence.
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

// Candidates are backed by CSV Payment entries with a known other participant.
func (d *DB) SplitwiseSettlementCandidates() ([]models.SplitwiseSettlementCandidate, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	members, err := d.listSplitwiseMembers()
	if err != nil {
		return nil, err
	}
	entries, err := d.listSplitwise()
	if err != nil {
		return nil, err
	}
	result := []models.SplitwiseSettlementCandidate{}
	for _, e := range entries {
		if e.Kind != "PAYMENT" || e.Status != "REVIEW" {
			continue
		}
		member, err := paymentMember(e, members)
		if err != nil {
			continue
		}
		candidates, err := splitwiseCandidates(d.conn, e.ID, models.SplitwiseMatchOptions{PaymentMember: &member})
		if err != nil {
			return nil, err
		}
		for _, candidate := range candidates {
			result = append(result, models.SplitwiseSettlementCandidate{Transaction: candidate, Group: e.Group, Member: member.Name})
			if len(result) == 100 {
				return result, nil
			}
		}
	}
	return result, nil
}

func (d *DB) ConfirmSplitwiseSettlement(req models.SplitwiseSettlementConfirmation) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	tx, err := d.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	members, err := listSplitwiseMembers(tx)
	if err != nil {
		return err
	}
	member, err := findSplitwiseMember(members, req.Group, req.Member)
	if err != nil {
		return err
	}
	entries, err := listSplitwise(tx)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.Kind != "PAYMENT" || e.Status != "REVIEW" || e.Group != req.Group {
			continue
		}
		other, err := paymentMember(e, members)
		if err != nil || other.Name != member.Name {
			continue
		}
		candidates, err := splitwiseCandidates(tx, e.ID, models.SplitwiseMatchOptions{PaymentMember: &member})
		if err != nil {
			return err
		}
		for _, candidate := range candidates {
			if candidate.ID != req.TransactionID {
				continue
			}
			if _, err := tx.Exec(`UPDATE splitwise_entries SET status='CONFIRMED',transaction_id=? WHERE id=?`, req.TransactionID, e.ID); err != nil {
				return err
			}
			return tx.Commit()
		}
	}
	return fmt.Errorf("a matching Splitwise Payment entry and bank amount are required")
}
