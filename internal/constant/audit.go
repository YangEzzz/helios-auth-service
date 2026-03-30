package constant

import "strings"

const (
	ActionUserLogin    = "user_login"
	ActionUserLogout   = "user_logout"
	ActionCreateProject = "create_project"
	ActionDeleteProject = "delete_project"
	ActionApproveUser   = "approve_user"
	ActionRejectUser    = "reject_user"
	ActionSetUserRole   = "set_user_role"
	ActionUpdateAvatar  = "update_avatar"
	ActionAddMember     = "add_member"
	ActionRemoveMember  = "remove_member"
	ActionUpdateMemberRole = "update_member_role"
	ActionAddRoleTemplate = "add_role_template"
	ActionUserRegister   = "user_register"
	ActionUserExternalLogin = "user_external_login"
)

var ActionMap = map[string]string{
	ActionUserLogin:    "用户登录",
	ActionUserLogout:   "用户登出",
	ActionCreateProject: "创建项目",
	ActionDeleteProject: "删除项目",
	ActionApproveUser:   "审批通过用户",
	ActionRejectUser:    "审批拒绝用户",
	ActionSetUserRole:   "设置用户角色",
	ActionUpdateAvatar:  "更新用户头像",
	ActionAddMember:     "添加项目成员",
	ActionRemoveMember:  "移除项目成员",
	ActionUpdateMemberRole: "更新成员角色",
	ActionAddRoleTemplate: "添加角色模板",
	ActionUserRegister:   "提交注册申请",
	ActionUserExternalLogin: "第三方系统登录",
}

func GetActionName(action string) string {
	if name, ok := ActionMap[action]; ok {
		return name
	}
	// 尝试不区分大小写匹配
	lowerAction := strings.ToLower(action)
	if name, ok := ActionMap[lowerAction]; ok {
		return name
	}
	return action
}

func GetResourceName(resource string) string {
	if strings.HasPrefix(resource, "user:") {
		return "用户: " + strings.TrimPrefix(resource, "user:")
	}
	if strings.HasPrefix(resource, "project:") {
		return "项目: " + strings.TrimPrefix(resource, "project:")
	}
	return resource
}

func GetDetailName(action, details string) string {
	// 根据行为和原始详情翻译一些通用的描述
	switch action {
	case ActionUserLogin:
		return "用户登录系统"
	case ActionUserRegister:
		return "用户主动提交注册申请，待审核"
	case ActionUpdateAvatar:
		return "用户更新了个人头像"
	case ActionApproveUser:
		return "管理员通过了该用户的注册申请"
	case ActionRejectUser:
		return "管理员拒绝或锁定了该用户"
	case "create_project":
		return "创建了新的项目"
	case "delete_project":
		return "删除了项目及其关联数据"
	case "add_member", "add_project_member":
		return "为项目添加了新成员"
	case "remove_member", "remove_project_member":
		return "从项目中移除了成员"
	case "set_user_role":
		return "修改了用户的全局权限角色"
	case "update_member_role", "update_project_member_role":
		return "修改了成员在项目中的工作角色"
	}
	
	// 简单替换一些关键词
	translated := details
	translated = strings.ReplaceAll(translated, "Approved user", "已通过用户审核")
	translated = strings.ReplaceAll(translated, "Rejected user", "已拒绝用户申请")
	translated = strings.ReplaceAll(translated, "Created project", "创建了项目")
	translated = strings.ReplaceAll(translated, "Set role to", "设置角色为")
	translated = strings.ReplaceAll(translated, "Updated profile avatar", "更新了个人头像")
	
	return translated
}
