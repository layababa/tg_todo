-- Go-Admin 自定义菜单初始化
-- 添加数据管理菜单及子菜单

-- 插入父菜单：数据管理
INSERT INTO goadmin_menu (parent_id, type, "order", title, plugin_name, icon, uri, created_at, updated_at) VALUES 
(0, 0, 1, '数据管理', '', 'fa-database', '', NOW(), NOW());

-- 插入子菜单：用户列表、任务列表、群组列表
INSERT INTO goadmin_menu (parent_id, type, "order", title, plugin_name, icon, uri, created_at, updated_at) VALUES 
((SELECT id FROM goadmin_menu WHERE title='数据管理'), 0, 1, '用户列表', '', 'fa-users', '/info/users', NOW(), NOW()),
((SELECT id FROM goadmin_menu WHERE title='数据管理'), 0, 2, '任务列表', '', 'fa-tasks', '/info/tasks', NOW(), NOW()),
((SELECT id FROM goadmin_menu WHERE title='数据管理'), 0, 3, '群组列表', '', 'fa-group', '/info/groups', NOW(), NOW());
