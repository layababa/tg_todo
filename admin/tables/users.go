package tables

import (
	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/modules/db"
	"github.com/GoAdminGroup/go-admin/plugins/admin/modules/table"
	"github.com/GoAdminGroup/go-admin/template/types"
	"github.com/GoAdminGroup/go-admin/template/types/form"
)

// GetUsersTable 返回 users 表的配置
func GetUsersTable(ctx *context.Context) table.Table {
	cfg := table.DefaultConfigWithDriver("postgresql")
	cfg.PrimaryKey = table.PrimaryKey{
		Type: db.Varchar,
		Name: "id",
	}

	users := table.NewDefaultTable(ctx, cfg)

	info := users.GetInfo()
	info.SetTable("users").SetTitle("用户管理").SetDescription("Telegram 用户列表")

	// 隐藏操作列（不允许编辑/删除）
	info.HideEditButton().HideDeleteButton().HideNewButton()

	info.AddField("ID", "id", db.Varchar).FieldFilterable()
	info.AddField("Telegram ID", "tg_id", db.Bigint).FieldSortable().FieldFilterable()
	info.AddField("用户名", "tg_username", db.Text).FieldFilterable()
	info.AddField("显示名", "name", db.Text).FieldFilterable()
	info.AddField("头像", "photo_url", db.Text).FieldDisplay(func(model types.FieldModel) interface{} {
		if model.Value == "" {
			return "-"
		}
		return "<img src='" + model.Value + "' width='32' height='32' style='border-radius:50%' />"
	})
	info.AddField("时区", "timezone", db.Text)
	info.AddField("Notion 已连接", "notion_connected", db.Bool).FieldDisplay(func(model types.FieldModel) interface{} {
		if model.Value == "true" || model.Value == "t" {
			return "<span class='label label-success'>是</span>"
		}
		return "<span class='label label-default'>否</span>"
	})
	info.AddField("注册时间", "created_at", db.Timestamptz).FieldSortable()
	info.AddField("更新时间", "updated_at", db.Timestamptz)

	info.SetFilterFormLayout(form.LayoutTwoCol)

	// 详情页配置
	formList := users.GetForm()
	formList.SetTable("users")
	formList.AddField("ID", "id", db.Varchar, form.Text).FieldNotAllowEdit().FieldNotAllowAdd()
	formList.AddField("Telegram ID", "tg_id", db.Bigint, form.Number).FieldNotAllowEdit()
	formList.AddField("用户名", "tg_username", db.Text, form.Text).FieldNotAllowEdit()
	formList.AddField("显示名", "name", db.Text, form.Text).FieldNotAllowEdit()
	formList.AddField("头像 URL", "photo_url", db.Text, form.Text).FieldNotAllowEdit()
	formList.AddField("时区", "timezone", db.Text, form.Text).FieldNotAllowEdit()
	formList.AddField("注册时间", "created_at", db.Timestamptz, form.Datetime).FieldNotAllowEdit()
	formList.AddField("更新时间", "updated_at", db.Timestamptz, form.Datetime).FieldNotAllowEdit()

	return users
}
