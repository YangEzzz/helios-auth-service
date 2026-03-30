package dashboard

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"helios-auth-service/internal/constant"
	"helios-auth-service/internal/models"

	"gorm.io/gorm"
)

type Service interface {
	GetDashboard(ctx context.Context, userID string) (*DashboardData, error)
}

type service struct {
	dao Dao
}

func NewService(dao Dao) Service {
	return &service{dao: dao}
}

type DashboardData struct {
	GeneratedAt string          `json:"generated_at"`
	RoleMode    string          `json:"role_mode"`
	Admin       *AdminDashboard `json:"admin,omitempty"`
	User        *UserDashboard  `json:"user,omitempty"`
}

type AdminDashboard struct {
	Stats          []DashboardStat `json:"stats"`
	ApprovalFeed   []ApprovalItem  `json:"approval_feed"`
	ActivityFeed   []ActivityItem  `json:"activity_feed"`
	ProjectTrend   []TrendPoint    `json:"project_trend"`
	SystemHeadline string          `json:"system_headline"`
}

type UserDashboard struct {
	WelcomeTitle  string          `json:"welcome_title"`
	WelcomeNote   string          `json:"welcome_note"`
	Summary       []DashboardStat `json:"summary"`
	Projects      []MyProjectItem `json:"projects"`
	ActivityFeed  []ActivityItem  `json:"activity_feed"`
	AccountStatus string          `json:"account_status"`
}

type DashboardStat struct {
	Label    string `json:"label"`
	Value    string `json:"value"`
	Trend    string `json:"trend"`
	Tone     string `json:"tone"`
	Emphasis string `json:"emphasis"`
}

type ApprovalItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	Status     string `json:"status"`
	Time       string `json:"time"`
	Operator   string `json:"operator"`
	OccurredAt string `json:"occurred_at"`
}

type ActivityItem struct {
	ID         string `json:"id"`
	Action     string `json:"action"`
	Detail     string `json:"detail"`
	Resource   string `json:"resource"`
	Time       string `json:"time"`
	Tone       string `json:"tone"`
	OccurredAt string `json:"occurred_at"`
}

type TrendPoint struct {
	Label string `json:"label"`
	Value int64  `json:"value"`
}

type MyProjectItem struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	ProjectKey    string `json:"project_key"`
	Description   string `json:"description"`
	Role          string `json:"role"`
	GrantedAt     string `json:"granted_at"`
	GrantedAtText string `json:"granted_at_text"`
}

func (s *service) GetDashboard(ctx context.Context, userID string) (*DashboardData, error) {
	user, err := s.dao.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	data := &DashboardData{
		GeneratedAt: time.Now().Format(time.RFC3339),
		RoleMode:    "user",
	}

	if user.Role == constant.UserRoleAdmin || user.Role == constant.UserRoleSuperAdmin {
		data.RoleMode = "admin"
		adminData, err := s.buildAdminDashboard(ctx)
		if err != nil {
			return nil, err
		}
		data.Admin = adminData
	}

	userData, err := s.buildUserDashboard(ctx, user)
	if err != nil {
		return nil, err
	}
	data.User = userData

	return data, nil
}

func (s *service) buildAdminDashboard(_ context.Context) (*AdminDashboard, error) {
	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)
	monthAgo := now.AddDate(0, 0, -30)

	totalUsers, err := s.dao.CountUsers()
	if err != nil {
		return nil, err
	}
	pendingUsers, err := s.dao.CountUsersByStatus(constant.UserStatusPending)
	if err != nil {
		return nil, err
	}
	totalProjects, err := s.dao.CountProjects()
	if err != nil {
		return nil, err
	}
	totalAudits, err := s.dao.CountAuditLogs()
	if err != nil {
		return nil, err
	}
	newUsersThisWeek, err := s.dao.CountUsersCreatedAfter(weekAgo)
	if err != nil {
		return nil, err
	}
	newProjectsThisWeek, err := s.dao.CountProjectsCreatedAfter(weekAgo)
	if err != nil {
		return nil, err
	}
	newAuditsThisMonth, err := s.dao.CountAuditLogsCreatedAfter(monthAgo)
	if err != nil {
		return nil, err
	}

	decisionLogs, err := s.dao.ListRecentDecisionLogs(5)
	if err != nil {
		return nil, err
	}
	recentLogs, err := s.dao.ListRecentAuditLogs(8)
	if err != nil {
		return nil, err
	}
	trendRows, err := s.dao.ListProjectCreationTrend(now.AddDate(0, 0, -6))
	if err != nil {
		return nil, err
	}

	trendMap := make(map[string]int64, len(trendRows))
	for _, row := range trendRows {
		dayKey := row.Day.Format("2006-01-02")
		trendMap[dayKey] = row.Count
	}

	trend := make([]TrendPoint, 0, 7)
	for i := 6; i >= 0; i-- {
		day := now.AddDate(0, 0, -i)
		dayKey := day.Format("2006-01-02")
		trend = append(trend, TrendPoint{
			Label: day.Format("01/02"),
			Value: trendMap[dayKey],
		})
	}

	approvalFeed := make([]ApprovalItem, 0, len(decisionLogs))
	for _, log := range decisionLogs {
		targetUser, _ := s.loadUserFromResource(log.Resource)
		status := "pending"
		if log.Action == constant.ActionApproveUser {
			status = "approved"
		} else if log.Action == constant.ActionRejectUser {
			status = "rejected"
		}

		approvalFeed = append(approvalFeed, ApprovalItem{
			ID:         log.ID.String(),
			Name:       userDisplayName(targetUser),
			Email:      userEmail(targetUser),
			Status:     status,
			Time:       humanizeSince(log.CreatedAt),
			Operator:   auditOperatorName(log),
			OccurredAt: log.CreatedAt.Format(time.RFC3339),
		})
	}

	activityFeed := make([]ActivityItem, 0, len(recentLogs))
	for _, log := range recentLogs {
		activityFeed = append(activityFeed, ActivityItem{
			ID:         log.ID.String(),
			Action:     displayAction(log),
			Detail:     displayDetail(log),
			Resource:   displayResource(log),
			Time:       humanizeSince(log.CreatedAt),
			Tone:       toneFromAction(log.Action),
			OccurredAt: log.CreatedAt.Format(time.RFC3339),
		})
	}

	return &AdminDashboard{
		Stats: []DashboardStat{
			{Label: "总用户数", Value: fmt.Sprintf("%d", totalUsers), Trend: fmt.Sprintf("+%d 本周新注册", newUsersThisWeek), Tone: "info", Emphasis: "users"},
			{Label: "待审批", Value: fmt.Sprintf("%d", pendingUsers), Trend: "需要尽快处理", Tone: "warning", Emphasis: "approvals"},
			{Label: "活跃项目", Value: fmt.Sprintf("%d", totalProjects), Trend: fmt.Sprintf("+%d 本周新增", newProjectsThisWeek), Tone: "success", Emphasis: "projects"},
			{Label: "授权记录", Value: fmt.Sprintf("%d", totalAudits), Trend: fmt.Sprintf("+%d 近 30 天变更", newAuditsThisMonth), Tone: "accent", Emphasis: "audit"},
		},
		ApprovalFeed:   approvalFeed,
		ActivityFeed:   activityFeed,
		ProjectTrend:   trend,
		SystemHeadline: fmt.Sprintf("过去 7 天新增 %d 个项目，当前有 %d 个待处理账号申请。", newProjectsThisWeek, pendingUsers),
	}, nil
}

func (s *service) buildUserDashboard(_ context.Context, user *models.User) (*UserDashboard, error) {
	userUUID := user.ID

	memberships, err := s.dao.ListProjectMembershipsForUser(userUUID)
	if err != nil {
		return nil, err
	}
	recentLogs, err := s.dao.ListRecentAuditLogsForUser(userUUID, 8)
	if err != nil {
		return nil, err
	}
	lastLogin, err := s.dao.GetLastLoginLogForUser(userUUID)
	if err != nil && !errorsIsRecordNotFound(err) {
		return nil, err
	}

	distinctRoles := make(map[string]struct{})
	projects := make([]MyProjectItem, 0, len(memberships))
	for _, membership := range memberships {
		if membership.Project == nil {
			continue
		}
		distinctRoles[membership.RoleInProject] = struct{}{}
		projects = append(projects, MyProjectItem{
			ID:            membership.Project.ID.String(),
			Name:          membership.Project.ProjectName,
			ProjectKey:    membership.Project.ProjectIDString,
			Description:   membership.Project.Description,
			Role:          membership.RoleInProject,
			GrantedAt:     membership.CreatedAt.Format("2006-01-02"),
			GrantedAtText: humanizeSince(membership.CreatedAt),
		})
	}

	lastAccessText := "暂无记录"
	if lastLogin != nil {
		lastAccessText = humanizeSince(lastLogin.CreatedAt)
	}

	activityFeed := make([]ActivityItem, 0, len(recentLogs))
	for _, log := range recentLogs {
		activityFeed = append(activityFeed, ActivityItem{
			ID:         log.ID.String(),
			Action:     displayAction(log),
			Detail:     displayDetail(log),
			Resource:   displayResource(log),
			Time:       humanizeSince(log.CreatedAt),
			Tone:       toneFromAction(log.Action),
			OccurredAt: log.CreatedAt.Format(time.RFC3339),
		})
	}

	return &UserDashboard{
		WelcomeTitle:  fmt.Sprintf("你好，%s", userDisplayName(user)),
		WelcomeNote:   fmt.Sprintf("当前账号状态为 %s，可直接查看已授权项目和最近访问动态。", user.Status),
		AccountStatus: string(user.Status),
		Summary: []DashboardStat{
			{Label: "我的项目", Value: fmt.Sprintf("%d", len(projects)), Trend: "已开通访问", Tone: "info", Emphasis: "projects"},
			{Label: "项目角色", Value: fmt.Sprintf("%d", len(distinctRoles)), Trend: "覆盖当前职责", Tone: "success", Emphasis: "roles"},
			{Label: "最近访问", Value: lastAccessText, Trend: "基于登录记录", Tone: "muted", Emphasis: "access"},
		},
		Projects:     projects,
		ActivityFeed: activityFeed,
	}, nil
}

func (s *service) loadUserFromResource(resource string) (*models.User, error) {
	if !strings.HasPrefix(resource, "user:") {
		return nil, fmt.Errorf("not user resource")
	}
	return s.dao.GetUserByID(strings.TrimPrefix(resource, "user:"))
}

func displayAction(log models.AuditLog) string {
	if log.ActionName != "" {
		return log.ActionName
	}
	return constant.GetActionName(log.Action)
}

func displayDetail(log models.AuditLog) string {
	if log.DetailsName != "" {
		return log.DetailsName
	}
	return constant.GetDetailName(log.Action, log.Details)
}

func displayResource(log models.AuditLog) string {
	if log.ResourceName != "" {
		return log.ResourceName
	}
	return constant.GetResourceName(log.Resource)
}

func auditOperatorName(log models.AuditLog) string {
	if log.User != nil && log.User.Username != "" {
		return log.User.Username
	}
	return "系统"
}

func userDisplayName(user *models.User) string {
	if user == nil {
		return "未知用户"
	}
	if user.Username != "" {
		return user.Username
	}
	if user.Email != "" {
		return user.Email
	}
	return "未知用户"
}

func userEmail(user *models.User) string {
	if user == nil {
		return "-"
	}
	if user.Email == "" {
		return "-"
	}
	return user.Email
}

func toneFromAction(action string) string {
	switch action {
	case constant.ActionApproveUser:
		return "success"
	case constant.ActionRejectUser, constant.ActionDeleteProject, constant.ActionRemoveMember:
		return "danger"
	case constant.ActionCreateProject, constant.ActionAddMember, constant.ActionAddRoleTemplate:
		return "info"
	default:
		return "neutral"
	}
}

func humanizeSince(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	diff := time.Since(t)
	switch {
	case diff < time.Minute:
		return "刚刚"
	case diff < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(diff.Minutes()))
	case diff < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(diff.Hours()))
	case diff < 7*24*time.Hour:
		return fmt.Sprintf("%d 天前", int(diff.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

func errorsIsRecordNotFound(err error) bool {
	return errors.Is(err, gorm.ErrRecordNotFound)
}
