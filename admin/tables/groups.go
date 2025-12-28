package tables

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
)

// GetGroupsTable 返回 groups 表的配置
func GetGroupsTable(ctx *context.Context) table.Table {
	groups := table.NewDefaultTable(ctx, table.DefaultConfigWithDriver("postgresql"))

	info := groups.GetInfo()
	info.SetTable("groups").SetTitle("群组管理").SetDescription("Telegram 群组列表")

	info.AddField("ID", "id", db.UUID).FieldFilterable()
	info.AddField("群名称", "title", db.Text).FieldFilterable()
	info.AddField("TG Chat ID", "tg_chat_id", db.Bigint).FieldSortable()
	info.AddField("状态", "status", db.Text).FieldDisplay(func(model types.FieldModel) interface{} {
		switch model.Value {
		case "Connected":
			return "<span class='label label-success'>已连接</span>"
		case "Unbound":
			return "<span class='label label-warning'>未绑定</span>"
		case "Inactive":
			return "<span class='label label-default'>失效</span>"
		default:
			return model.Value
		}
	}).FieldFilterable(types.FilterType{FormType: form.Select}).FieldFilterOptions(types.FieldOptions{
		{Value: "Connected", Text: "已连接"},
		{Value: "Unbound", Text: "未绑定"},
		{Value: "Inactive", Text: "失效"},
	})
	info.AddField("创建时间", "created_at", db.Timestamptz).FieldSortable()
	info.AddField("更新时间", "updated_at", db.Timestamptz)

	info.SetFilterFormLayout(form.LayoutTwoCol)

	formList := groups.GetForm()
	formList.SetTable("groups")
	formList.AddField("ID", "id", db.UUID, form.Text).FieldNotAllowEdit().FieldNotAllowAdd()
	formList.AddField("群名称", "title", db.Text, form.Text)
	formList.AddField("TG Chat ID", "tg_chat_id", db.Bigint, form.Number).FieldNotAllowEdit()
	formList.AddField("状态", "status", db.Text, form.Select).FieldOptions(types.FieldOptions{
		{Value: "Connected", Text: "已连接"},
		{Value: "Unbound", Text: "未绑定"},
		{Value: "Inactive", Text: "失效"},
	})
	formList.AddField("创建时间", "created_at", db.Timestamptz, form.Datetime).FieldNotAllowEdit()
	formList.AddField("更新时间", "updated_at", db.Timestamptz, form.Datetime).FieldNotAllowEdit()

	return groups
}
