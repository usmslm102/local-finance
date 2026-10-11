package db

import "fmt"

// MCPSettings never contains the credential itself. TokenHash is not serialized.
type MCPSettings struct {
	AllowStatementWrites      bool   `json:"allow_statement_writes"`
	Enabled                   bool   `json:"enabled"`
	Port                      int    `json:"port"`
	TokenHash                 string `json:"-"`
	AllowCategorizationWrites bool   `json:"allow_categorization_writes"`
}

func (d *DB) GetMCPSettings() (MCPSettings, error) {
	d.mu.RLock()
	defer d.mu.RUnlock()
	var s MCPSettings
	err := d.conn.QueryRow("SELECT enabled, port, token_hash, allow_categorization_writes, allow_statement_writes FROM mcp_settings WHERE id = 1").Scan(&s.Enabled, &s.Port, &s.TokenHash, &s.AllowCategorizationWrites, &s.AllowStatementWrites)
	return s, err
}

func (d *DB) SetMCPSettings(s MCPSettings) error {
	if s.Port < 1024 || s.Port > 65535 {
		return fmt.Errorf("MCP port must be between 1024 and 65535")
	}
	if s.Enabled && s.TokenHash == "" {
		return fmt.Errorf("create an MCP token before enabling access")
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	_, err := d.conn.Exec("UPDATE mcp_settings SET enabled = ?, port = ?, token_hash = ?, allow_categorization_writes = ?, allow_statement_writes = ? WHERE id = 1", s.Enabled, s.Port, s.TokenHash, s.AllowCategorizationWrites, s.AllowStatementWrites)
	return err
}

// clearMCPAccess is called with d.mu held before destructive database operations.
func (d *DB) clearMCPAccess() error {
	_, err := d.conn.Exec("UPDATE mcp_settings SET enabled = 0, token_hash = '', allow_categorization_writes = 0, allow_statement_writes = 0 WHERE id = 1")
	return err
}
