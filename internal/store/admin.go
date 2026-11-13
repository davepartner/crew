package store

import (
	"context"
	"time"
)

type DailyPoint struct {
	Date      string  `json:"date"`
	Signups   int     `json:"signups"`
	Companies int     `json:"companies"`
	Tasks     int     `json:"tasks"`
	Completed int     `json:"completed"`
	SpendUSD  float64 `json:"spend_usd"`
}

type AdminUser struct {
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Companies int       `json:"companies"`
	Tasks     int       `json:"tasks"`
}

type CompanySummary struct {
	Name     string  `json:"name"`
	Users    int     `json:"users"`
	Agents   int     `json:"agents"`
	Tasks    int     `json:"tasks"`
	SpendUSD float64 `json:"spend_usd"`
}

type AdminStats struct {
	Users         int              `json:"users"`
	Companies     int              `json:"companies"`
	Agents        int              `json:"agents"`
	Tasks         int              `json:"tasks"`
	TasksByStatus map[string]int   `json:"tasks_by_status"`
	ModelCalls    int              `json:"model_calls"`
	PromptTokens  int              `json:"prompt_tokens"`
	ReplyTokens   int              `json:"reply_tokens"`
	SpendUSD      float64          `json:"spend_usd"`
	ActiveUsers7d int              `json:"active_users_7d"`
	Daily         []DailyPoint     `json:"daily"`
	RecentUsers   []AdminUser      `json:"recent_users"`
	TopCompanies  []CompanySummary `json:"top_companies"`
}

func (s *Store) AdminStats(ctx context.Context, days int) (AdminStats, error) {
	if days <= 0 || days > 90 {
		days = 30
	}

	stats := AdminStats{TasksByStatus: map[string]int{}}
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.Users)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM companies`).Scan(&stats.Companies)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agents`).Scan(&stats.Agents)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks`).Scan(&stats.Tasks)
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(*), COALESCE(SUM(prompt_tokens),0), COALESCE(SUM(completion_tokens),0), COALESCE(SUM(cost_usd),0)
		FROM model_usage`).Scan(&stats.ModelCalls, &stats.PromptTokens, &stats.ReplyTokens, &stats.SpendUSD)

	rows, err := s.db.QueryContext(ctx, `SELECT status, COUNT(*) FROM tasks GROUP BY status`)
	if err == nil {
		for rows.Next() {
			var status string
			var count int
			if rows.Scan(&status, &count) == nil {
				stats.TasksByStatus[status] = count
			}
		}
		rows.Close()
	}

	since := time.Now().UTC().AddDate(0, 0, -7).Format(TimeFormat)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT user_id) FROM activity WHERE created_at >= ?`, since).Scan(&stats.ActiveUsers7d)

	start := time.Now().UTC().AddDate(0, 0, -(days - 1))
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	points := map[string]*DailyPoint{}
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		points[date] = &DailyPoint{Date: date}
	}

	collect := func(query string, apply func(point *DailyPoint, value float64)) {
		rows, err := s.db.QueryContext(ctx, query, start.Format(TimeFormat))
		if err != nil {
			return
		}
		defer rows.Close()
		for rows.Next() {
			var date string
			var value float64
			if rows.Scan(&date, &value) == nil {
				if point, ok := points[date]; ok {
					apply(point, value)
				}
			}
		}
	}

	collect(`SELECT date(created_at), COUNT(*) FROM users WHERE created_at >= ? GROUP BY date(created_at)`,
		func(point *DailyPoint, value float64) { point.Signups = int(value) })
	collect(`SELECT date(created_at), COUNT(*) FROM companies WHERE created_at >= ? GROUP BY date(created_at)`,
		func(point *DailyPoint, value float64) { point.Companies = int(value) })
	collect(`SELECT date(created_at), COUNT(*) FROM tasks WHERE created_at >= ? GROUP BY date(created_at)`,
		func(point *DailyPoint, value float64) { point.Tasks = int(value) })
	collect(`SELECT date(completed_at), COUNT(*) FROM tasks WHERE completed_at IS NOT NULL AND completed_at >= ? GROUP BY date(completed_at)`,
		func(point *DailyPoint, value float64) { point.Completed = int(value) })
	collect(`SELECT date(created_at), COALESCE(SUM(cost_usd),0) FROM model_usage WHERE created_at >= ? GROUP BY date(created_at)`,
		func(point *DailyPoint, value float64) { point.SpendUSD = value })

	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i).Format("2006-01-02")
		stats.Daily = append(stats.Daily, *points[date])
	}

	userRows, err := s.db.QueryContext(ctx, `
		SELECT u.email, u.name, u.created_at,
		       (SELECT COUNT(*) FROM companies c WHERE c.user_id = u.id),
		       (SELECT COUNT(*) FROM tasks t WHERE t.user_id = u.id)
		FROM users u ORDER BY u.created_at DESC LIMIT 20`)
	if err == nil {
		defer userRows.Close()
		for userRows.Next() {
			var user AdminUser
			var created string
			if userRows.Scan(&user.Email, &user.Name, &created, &user.Companies, &user.Tasks) == nil {
				user.CreatedAt = ParseTime(created)
				stats.RecentUsers = append(stats.RecentUsers, user)
			}
		}
	}

	companyRows, err := s.db.QueryContext(ctx, `
		SELECT c.name,
		       (SELECT COUNT(*) FROM company_members m WHERE m.company_id = c.id) + 1,
		       (SELECT COUNT(*) FROM agents a WHERE a.company_id = c.id),
		       (SELECT COUNT(*) FROM tasks t WHERE t.company_id = c.id),
		       (SELECT COALESCE(SUM(u.cost_usd),0) FROM model_usage u WHERE u.company_id = c.id)
		FROM companies c ORDER BY 5 DESC LIMIT 8`)
	if err == nil {
		defer companyRows.Close()
		for companyRows.Next() {
			var summary CompanySummary
			if companyRows.Scan(&summary.Name, &summary.Users, &summary.Agents, &summary.Tasks, &summary.SpendUSD) == nil {
				stats.TopCompanies = append(stats.TopCompanies, summary)
			}
		}
	}

	return stats, nil
}

type UserUsage struct {
	Exists     bool      `json:"exists"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	CreatedAt  time.Time `json:"created_at"`
	Companies  int       `json:"companies"`
	Agents     int       `json:"agents"`
	Tasks      int       `json:"tasks"`
	Completed  int       `json:"completed"`
	SpendUSD   float64   `json:"spend_usd"`
	LastActive time.Time `json:"last_active"`
}

func (s *Store) UserUsage(ctx context.Context, email string) (UserUsage, error) {
	usage := UserUsage{Email: email}

	var id, created string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, created_at FROM users WHERE lower(email) = lower(?)`, email).Scan(&id, &usage.Name, &created)
	if err != nil {
		return usage, nil
	}

	usage.Exists = true
	usage.CreatedAt = ParseTime(created)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM companies WHERE user_id = ?`, id).Scan(&usage.Companies)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM agents WHERE user_id = ?`, id).Scan(&usage.Agents)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE user_id = ?`, id).Scan(&usage.Tasks)
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM tasks WHERE user_id = ? AND status = 'done'`, id).Scan(&usage.Completed)
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(cost_usd),0) FROM model_usage WHERE user_id = ?`, id).Scan(&usage.SpendUSD)

	var last string
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(created_at),'') FROM activity WHERE user_id = ?`, id).Scan(&last); err == nil && last != "" {
		usage.LastActive = ParseTime(last)
	}

	return usage, nil
}
