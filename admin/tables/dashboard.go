package tables

import (
	"html/template"

	"github.com/GoAdminGroup/go-admin/context"
	"github.com/GoAdminGroup/go-admin/template/types"
)

// GetDashboardContent 返回仪表盘内容
func GetDashboardContent(ctx *context.Context) (types.Panel, error) {
	// 简化的仪表盘，使用纯 HTML
	content := template.HTML(`
<div class="row">
    <div class="col-md-3 col-sm-6 col-xs-12">
        <div class="info-box bg-aqua">
            <span class="info-box-icon"><i class="fa fa-users"></i></span>
            <div class="info-box-content">
                <span class="info-box-text">用户总数</span>
                <span class="info-box-number" id="user-count">-</span>
            </div>
        </div>
    </div>
    <div class="col-md-3 col-sm-6 col-xs-12">
        <div class="info-box bg-green">
            <span class="info-box-icon"><i class="fa fa-tasks"></i></span>
            <div class="info-box-content">
                <span class="info-box-text">任务总数</span>
                <span class="info-box-number" id="task-count">-</span>
            </div>
        </div>
    </div>
    <div class="col-md-3 col-sm-6 col-xs-12">
        <div class="info-box bg-yellow">
            <span class="info-box-icon"><i class="fa fa-group"></i></span>
            <div class="info-box-content">
                <span class="info-box-text">群组总数</span>
                <span class="info-box-number" id="group-count">-</span>
            </div>
        </div>
    </div>
    <div class="col-md-3 col-sm-6 col-xs-12">
        <div class="info-box bg-red">
            <span class="info-box-icon"><i class="fa fa-clock-o"></i></span>
            <div class="info-box-content">
                <span class="info-box-text">待办任务</span>
                <span class="info-box-number" id="todo-count">-</span>
            </div>
        </div>
    </div>
</div>

<div class="row">
    <div class="col-md-6">
        <div class="box box-primary">
            <div class="box-header with-border">
                <h3 class="box-title">任务状态分布</h3>
            </div>
            <div class="box-body">
                <canvas id="taskStatusChart" height="200"></canvas>
            </div>
        </div>
    </div>
    <div class="col-md-6">
        <div class="box box-success">
            <div class="box-header with-border">
                <h3 class="box-title">群组状态分布</h3>
            </div>
            <div class="box-body">
                <canvas id="groupStatusChart" height="200"></canvas>
            </div>
        </div>
    </div>
</div>

<script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
<script>
$(document).ready(function() {
    // 任务状态饼图
    new Chart(document.getElementById('taskStatusChart'), {
        type: 'doughnut',
        data: {
            labels: ['待办', '进行中', '已完成'],
            datasets: [{
                data: [30, 20, 50],
                backgroundColor: ['#f39c12', '#00c0ef', '#00a65a']
            }]
        },
        options: { responsive: true }
    });
    
    // 群组状态饼图
    new Chart(document.getElementById('groupStatusChart'), {
        type: 'doughnut',
        data: {
            labels: ['已连接', '未绑定', '失效'],
            datasets: [{
                data: [70, 20, 10],
                backgroundColor: ['#00a65a', '#f39c12', '#6c757d']
            }]
        },
        options: { responsive: true }
    });
});
</script>
`)

	return types.Panel{
		Content:     content,
		Title:       "仪表盘",
		Description: "TG TODO 系统概览",
	}, nil
}
