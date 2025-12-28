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
	users := table.NewDefaultTable(ctx, table.DefaultConfigWithDriver("postgresql"))

	info := users.GetInfo()
	info.SetTable("users").SetTitle("用户管理").SetDescription("Telegram 用户列表")

	info.AddField("ID", "id", db.UUID).FieldFilterable()
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
	info.AddField("注册时间", "created_at", db.Timestamptz).FieldSortable()
	info.AddField("更新时间", "updated_at", db.Timestamptz)

	info.SetFilterFormLayout(form.LayoutTwoCol)

	formList := users.GetForm()
	formList.SetTable("users")
	formList.AddField("ID", "id", db.UUID, form.Text).FieldNotAllowEdit().FieldNotAllowAdd()
	formList.AddField("Telegram ID", "tg_id", db.Bigint, form.Number).FieldNotAllowEdit()
	formList.AddField("用户名", "tg_username", db.Text, form.Text)
	formList.AddField("显示名", "name", db.Text, form.Text)
	formList.AddField("头像 URL", "photo_url", db.Text, form.Text)
	formList.AddField("时区", "timezone", db.Text, form.Text)
	formList.AddField("注册时间", "created_at", db.Timestamptz, form.Datetime).FieldNotAllowEdit()
	formList.AddField("更新时间", "updated_at", db.Timestamptz, form.Datetime).FieldNotAllowEdit()

	return users
}
