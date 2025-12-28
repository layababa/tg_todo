package tables

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
)

// GetTasksTable 返回 tasks 表的配置
func GetTasksTable(ctx *context.Context) table.Table {
	cfg := table.DefaultConfigWithDriver("postgresql")
	cfg.PrimaryKey = table.PrimaryKey{
		Type: db.Varchar,
		Name: "id",
	}

	tasks := table.NewDefaultTable(ctx, cfg)

	info := tasks.GetInfo()
	info.SetTable("tasks").SetTitle("任务管理").SetDescription("待办任务列表")

	// 隐藏操作列（只读模式）
	info.HideEditButton().HideDeleteButton().HideNewButton()

	info.AddField("ID", "id", db.Varchar).FieldHide()
	info.AddField("标题", "title", db.Text).FieldFilterable()
	info.AddField("状态", "status", db.Text).FieldDisplay(func(model types.FieldModel) interface{} {
		switch model.Value {
		case "To Do":
			return "<span class='label label-warning'>待办</span>"
		case "In Progress":
			return "<span class='label label-info'>进行中</span>"
		case "Done":
			return "<span class='label label-success'>已完成</span>"
		default:
			return model.Value
		}
	}).FieldFilterable(types.FilterType{FormType: form.Select}).FieldFilterOptions(types.FieldOptions{
		{Value: "To Do", Text: "待办"},
		{Value: "In Progress", Text: "进行中"},
		{Value: "Done", Text: "已完成"},
	})

	info.AddField("同步状态", "sync_status", db.Text).FieldDisplay(func(model types.FieldModel) interface{} {
		switch model.Value {
		case "Synced":
			return "<span class='label label-success'>已同步</span>"
		case "Pending":
			return "<span class='label label-warning'>待同步</span>"
		case "Failed":
			return "<span class='label label-danger'>失败</span>"
		default:
			return model.Value
		}
	})
	info.AddField("截止时间", "due_at", db.Timestamptz).FieldSortable()
	info.AddField("创建时间", "created_at", db.Timestamptz).FieldSortable()
	info.AddField("归档", "archived", db.Bool).FieldDisplay(func(model types.FieldModel) interface{} {
		if model.Value == "true" || model.Value == "t" {
			return "<span class='label label-default'>已归档</span>"
		}
		return "-"
	})

	info.SetFilterFormLayout(form.LayoutTwoCol)

	// 详情页配置
	formList := tasks.GetForm()
	formList.SetTable("tasks")
	formList.AddField("ID", "id", db.Varchar, form.Text).FieldNotAllowEdit().FieldNotAllowAdd()
	formList.AddField("标题", "title", db.Text, form.Text).FieldNotAllowEdit()
	formList.AddField("状态", "status", db.Text, form.Text).FieldNotAllowEdit()
	formList.AddField("同步状态", "sync_status", db.Text, form.Text).FieldNotAllowEdit()
	formList.AddField("截止时间", "due_at", db.Timestamptz, form.Datetime).FieldNotAllowEdit()
	formList.AddField("创建时间", "created_at", db.Timestamptz, form.Datetime).FieldNotAllowEdit()
	formList.AddField("更新时间", "updated_at", db.Timestamptz, form.Datetime).FieldNotAllowEdit()

	return tasks
}
