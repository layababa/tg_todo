package tables

import "github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"

// Generators 是所有数据表生成器的集合
var Generators = map[string]table.Generator{
	"users":  GetUsersTable,
	"tasks":  GetTasksTable,
	"groups": GetGroupsTable,
}
