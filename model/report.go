package model

type BillingReportRow struct {
	Date             string `json:"date"`
	TenantId         int    `json:"tenant_id,omitempty"`
	TenantName       string `json:"tenant_name,omitempty"`
	UserId           int    `json:"user_id,omitempty"`
	Username         string `json:"username,omitempty"`
	ChannelId        int    `json:"channel_id,omitempty"`
	ChannelName      string `json:"channel_name,omitempty"`
	ModelName        string `json:"model_name,omitempty"`
	Quota            int64  `json:"quota"`
	RequestCount     int64  `json:"request_count"`
	PromptTokens     int64  `json:"prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens"`
}

func GetPlatformBillingReport(startTimestamp, endTimestamp int64) ([]*BillingReportRow, error) {
	var rows []*BillingReportRow
	// Postgres and MySQL have different date formatting. Since this is an abstraction, let's just group by day using seconds division, or let database handle it.
	// A simpler cross-db approach for epoch seconds to date string:
	// MySQL: DATE_FORMAT(FROM_UNIXTIME(created_at), '%Y-%m-%d')
	// SQLite: date(created_at, 'unixepoch')
	// PostgreSQL: to_char(to_timestamp(created_at), 'YYYY-MM-DD')
	
	// Given we support multiple databases, doing grouping in memory might be safer if data is small, but logs can be large.
	// Let's use the DB's capabilities conditionally or just fetch raw records within time range and group them if the db dialect is unknown, but Gorm can do it if we are careful.
	// Actually, an easier way for cross-db is to retrieve day buckets by time math: (created_at + timezone_offset) / 86400
	// But let's just retrieve the records and group in memory if we want simple cross-db, or use basic sql.
	// For simplicity, let's use a query that gets the basic stats per tenant per day.

	query := `
		SELECT 
			u.tenant_id,
			t.name as tenant_name,
			SUM(l.quota) as quota,
			COUNT(l.id) as request_count,
			SUM(l.prompt_tokens) as prompt_tokens,
			SUM(l.completion_tokens) as completion_tokens
		FROM logs l
		LEFT JOIN users u ON l.user_id = u.id
		LEFT JOIN tenants t ON u.tenant_id = t.id
		WHERE l.created_at >= ? AND l.created_at <= ? AND l.type = 2
		GROUP BY u.tenant_id, t.name
	`
	
	err := LOG_DB.Raw(query, startTimestamp, endTimestamp).Scan(&rows).Error
	return rows, err
}

func GetTenantBillingReport(tenantId int, startTimestamp, endTimestamp int64) ([]*BillingReportRow, error) {
	var rows []*BillingReportRow
	
	query := `
		SELECT 
			l.user_id,
			u.username,
			l.channel_id,
			c.name as channel_name,
			l.model_name,
			SUM(l.quota) as quota,
			COUNT(l.id) as request_count,
			SUM(l.prompt_tokens) as prompt_tokens,
			SUM(l.completion_tokens) as completion_tokens
		FROM logs l
		JOIN users u ON l.user_id = u.id
		LEFT JOIN channels c ON l.channel_id = c.id
		WHERE u.tenant_id = ? AND l.created_at >= ? AND l.created_at <= ? AND l.type = 2
		GROUP BY l.user_id, u.username, l.channel_id, c.name, l.model_name
	`
	
	err := LOG_DB.Raw(query, tenantId, startTimestamp, endTimestamp).Scan(&rows).Error
	return rows, err
}

func GetPlatformBillingExport(startTimestamp, endTimestamp int64) ([]*BillingReportRow, error) {
	// For export we might want detailed records or just the aggregated report.
	// Usually export is either raw logs or detailed aggregations. We'll return the same for now.
	return GetPlatformBillingReport(startTimestamp, endTimestamp)
}

func GetTenantBillingExport(tenantId int, startTimestamp, endTimestamp int64) ([]*BillingReportRow, error) {
	return GetTenantBillingReport(tenantId, startTimestamp, endTimestamp)
}