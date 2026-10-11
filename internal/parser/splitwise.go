package parser

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"local-finance/internal/models"
)

const MaxSplitwiseFileSize int64 = 10 << 20

// ParseSplitwise reads group-export net balances, not bank transactions.
// Exact member names win; a first name is accepted only when unambiguous.
// Net balances determine personal shares; bank-backed movements require matching evidence.
func ParseSplitwise(r io.Reader, group, person string) ([]models.SplitwiseEntry, error) {
	group, person = strings.TrimSpace(group), strings.TrimSpace(person)
	if group == "" || person == "" {
		return nil, fmt.Errorf("group and member name are required")
	}
	if len(group) > 200 || len(person) > 200 {
		return nil, fmt.Errorf("group and member names are limited to 200 bytes")
	}
	data, err := io.ReadAll(io.LimitReader(r, MaxSplitwiseFileSize+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > MaxSplitwiseFileSize {
		return nil, fmt.Errorf("Splitwise CSV exceeds 10 MB")
	}
	reader := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(data), "\ufeff")))
	header, err := reader.Read()
	if err != nil || len(header) < 6 {
		return nil, fmt.Errorf("invalid Splitwise CSV header")
	}
	if len(header)-5 > 100 {
		return nil, fmt.Errorf("Splitwise exports are limited to 100 members")
	}
	seenNames := map[string]bool{}
	for _, name := range header[5:] {
		name = strings.TrimSpace(name)
		key := strings.ToLower(name)
		if name == "" || len(name) > 200 {
			return nil, fmt.Errorf("member names must contain 1 to 200 bytes")
		}
		if seenNames[key] {
			return nil, fmt.Errorf("member names must be unique, ignoring case")
		}
		seenNames[key] = true
	}
	for i, name := range []string{"Date", "Description", "Category", "Cost", "Currency"} {
		if strings.TrimSpace(header[i]) != name {
			return nil, fmt.Errorf("expected Splitwise column %s", name)
		}
	}
	member := -1
	for i := 5; i < len(header); i++ {
		if strings.EqualFold(strings.TrimSpace(header[i]), person) {
			if member != -1 {
				return nil, fmt.Errorf("ambiguous member name; use a unique CSV member column")
			}
			member = i
		}
	}
	if member == -1 {
		for i := 5; i < len(header); i++ {
			fields := strings.Fields(header[i])
			if len(fields) > 0 && strings.EqualFold(fields[0], person) {
				if member != -1 {
					return nil, fmt.Errorf("ambiguous member name; use the full CSV member name")
				}
				member = i
			}
		}
	}
	if member == -1 {
		return nil, fmt.Errorf("member %q is not in this export", person)
	}
	person = strings.TrimSpace(header[member])
	members := make([]string, 0, len(header)-5)
	for _, name := range header[5:] {
		name = strings.TrimSpace(name)
		members = append(members, name)
	}
	// Preview responses repeat membership per row. Bound serialized expansion as
	// well as upload bytes, including JSON escaping of untrusted header names.
	membershipJSON, err := json.Marshal(members)
	if err != nil {
		return nil, err
	}
	result := []models.SplitwiseEntry{}
	occurrences := map[string]int{}
	for row := 2; ; row++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("row %d: %w", row, err)
		}
		if len(record[1]) > 2000 || len(record[2]) > 200 {
			return nil, fmt.Errorf("row %d: description or category exceeds its length limit", row)
		}
		if strings.EqualFold(strings.TrimSpace(record[1]), "Total balance") && strings.TrimSpace(record[3]) == "" {
			continue
		}
		if len(result) >= 50000 {
			return nil, fmt.Errorf("Splitwise exports are limited to 50000 transactions")
		}
		if (len(result)+1)*len(membershipJSON) > 16<<20 {
			return nil, fmt.Errorf("Splitwise export exceeds the 16 MB expanded membership limit")
		}
		date := strings.TrimSpace(record[0])
		if _, err := time.Parse(time.DateOnly, date); err != nil {
			return nil, fmt.Errorf("row %d: invalid date", row)
		}
		if strings.TrimSpace(record[4]) != "INR" {
			return nil, fmt.Errorf("row %d: only INR exports are supported", row)
		}
		cost, err := splitwiseCents(record[3])
		if err != nil || cost <= 0 {
			return nil, fmt.Errorf("row %d: invalid cost", row)
		}
		var net, sum int64
		for i := 5; i < len(record); i++ {
			value, err := splitwiseCents(record[i])
			if err != nil || value < -cost || value > cost {
				return nil, fmt.Errorf("row %d: invalid member balance", row)
			}
			sum += value
			if i == member {
				net = value
			}
		}
		if sum != 0 {
			return nil, fmt.Errorf("row %d: member balances must sum to zero", row)
		}
		entry := models.SplitwiseEntry{Group: group, Person: person, Date: date, Description: strings.TrimSpace(record[1]), Category: strings.TrimSpace(record[2]), Cost: cost, Net: net, Kind: "EXPENSE", Status: "REVIEW"}
		entry.Members = members
		if entry.Description == "" {
			return nil, fmt.Errorf("row %d: description is required", row)
		}
		if strings.EqualFold(entry.Category, "Payment") {
			entry.Kind = "PAYMENT"
			for i := 5; i < len(record); i++ {
				value, _ := splitwiseCents(record[i])
				if net != 0 && i != member && value != 0 {
					if entry.Counterparty != "" || value != -net {
						return nil, fmt.Errorf("row %d: payment must identify two members", row)
					}
					entry.Counterparty = strings.TrimSpace(header[i])
				}
			}
		} else if net < 0 {
			entry.Share = -net
		} else if net > 0 {
			entry.Share = cost - net
		}
		// Include all member balances and an occurrence number: identical real expenses
		// in one export remain distinct, overlapping unchanged exports stay idempotent.
		canonical := []string{strings.ToLower(group), strings.ToLower(person), date, entry.Description, entry.Category, strconv.FormatInt(cost, 10)}
		balances := []string{}
		for i := 5; i < len(record); i++ {
			value, _ := splitwiseCents(record[i])
			if value != 0 {
				balances = append(balances, fmt.Sprintf("%q:%d", strings.ToLower(strings.TrimSpace(header[i])), value))
			}
		}
		sort.Strings(balances)
		canonical = append(canonical, balances...)
		key := fmt.Sprintf("%q", canonical)
		occurrences[key]++
		entry.ID = fmt.Sprintf("sw-%x", sha256.Sum256([]byte(fmt.Sprintf("%s|%d", key, occurrences[key]))))
		result = append(result, entry)
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no Splitwise transactions found")
	}
	return result, nil
}

func splitwiseCents(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	negative := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")
	parts := strings.Split(s, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, fmt.Errorf("invalid money")
	}
	for _, part := range parts {
		for _, c := range part {
			if c < '0' || c > '9' {
				return 0, fmt.Errorf("invalid money")
			}
		}
	}
	whole, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || whole > 1_000_000_000 {
		return 0, fmt.Errorf("invalid money")
	}
	fraction := "00"
	if len(parts) == 2 {
		if len(parts[1]) > 2 || parts[1] == "" {
			return 0, fmt.Errorf("invalid money")
		}
		fraction = (parts[1] + "00")[:2]
	}
	paise, _ := strconv.ParseInt(fraction, 10, 64)
	value := whole*100 + paise
	if negative {
		value = -value
	}
	return value, nil
}
