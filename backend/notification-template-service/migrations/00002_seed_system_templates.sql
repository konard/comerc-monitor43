-- +goose Up
-- SQL в этой секции выполняется при миграции вверх.

-- Вставляем системные шаблоны для Email уведомлений

-- Monitor Down - Email
INSERT INTO templates (
	id, user_id, name, description, channel, type, engine,
	subject, body, format, is_default, is_system, version, created_at, updated_at
) VALUES (
	'00000000-0000-0000-0000-000000000001',
	'00000000-0000-0000-0000-000000000000',
	'Default Monitor Down (Email)',
	'System template for monitor down alerts',
	'email',
	'monitor_down',
	'gotemplate',
	'🚨 Alert: {{ .monitor_name }} is DOWN',
	'<h2>Monitor Alert</h2>
<p><strong>Monitor:</strong> {{ .monitor_name }}</p>
<p><strong>Status:</strong> {{ .status }}</p>
<p><strong>URL:</strong> <a href="{{ .monitor_url }}">{{ .monitor_url }}</a></p>
<p><strong>Error:</strong> {{ .error_message }}</p>
<p><strong>Time:</strong> {{ .timestamp }}</p>',
	'html',
	true,
	true,
	1,
	NOW(),
	NOW()
) ON CONFLICT DO NOTHING;

-- Monitor Up - Email
INSERT INTO templates (
	id, user_id, name, description, channel, type, engine,
	subject, body, format, is_default, is_system, version, created_at, updated_at
) VALUES (
	'00000000-0000-0000-0000-000000000002',
	'00000000-0000-0000-0000-000000000000',
	'Default Monitor Up (Email)',
	'System template for monitor recovery notifications',
	'email',
	'monitor_up',
	'gotemplate',
	'✅ {{ .monitor_name }} is back UP',
	'<h2>Monitor Recovered</h2>
<p><strong>Monitor:</strong> {{ .monitor_name }}</p>
<p><strong>Status:</strong> {{ .status }}</p>
<p><strong>URL:</strong> <a href="{{ .monitor_url }}">{{ .monitor_url }}</a></p>
<p><strong>Time:</strong> {{ .timestamp }}</p>',
	'html',
	true,
	true,
	1,
	NOW(),
	NOW()
) ON CONFLICT DO NOTHING;

-- Monitor Degraded - Email
INSERT INTO templates (
	id, user_id, name, description, channel, type, engine,
	subject, body, format, is_default, is_system, version, created_at, updated_at
) VALUES (
	'00000000-0000-0000-0000-000000000003',
	'00000000-0000-0000-0000-000000000000',
	'Default Monitor Degraded (Email)',
	'System template for degraded performance alerts',
	'email',
	'monitor_degraded',
	'gotemplate',
	'⚠️ Alert: {{ .monitor_name }} is DEGRADED',
	'<h2>Performance Degraded</h2>
<p><strong>Monitor:</strong> {{ .monitor_name }}</p>
<p><strong>Status:</strong> {{ .status }}</p>
<p><strong>Response Time:</strong> {{ .response_time_ms }}ms</p>
<p><strong>Threshold:</strong> {{ .threshold_ms }}ms</p>
<p><strong>URL:</strong> <a href="{{ .monitor_url }}">{{ .monitor_url }}</a></p>
<p><strong>Time:</strong> {{ .timestamp }}</p>',
	'html',
	true,
	true,
	1,
	NOW(),
	NOW()
) ON CONFLICT DO NOTHING;

-- Certificate Expiry - Email
INSERT INTO templates (
	id, user_id, name, description, channel, type, engine,
	subject, body, format, is_default, is_system, version, created_at, updated_at
) VALUES (
	'00000000-0000-0000-0000-000000000004',
	'00000000-0000-0000-0000-000000000000',
	'Default Certificate Expiry (Email)',
	'System template for SSL certificate expiry warnings',
	'email',
	'certificate_expiry',
	'gotemplate',
	'🔐 SSL Certificate Expiring Soon for {{ .monitor_name }}',
	'<h2>SSL Certificate Expiring</h2>
<p><strong>Monitor:</strong> {{ .monitor_name }}</p>
<p><strong>URL:</strong> <a href="{{ .monitor_url }}">{{ .monitor_url }}</a></p>
<p><strong>Days Until Expiry:</strong> {{ .days_until_expiry }}</p>
<p><strong>Expiry Date:</strong> {{ .expiry_date }}</p>
<p><strong>Time:</strong> {{ .timestamp }}</p>',
	'html',
	true,
	true,
	1,
	NOW(),
	NOW()
) ON CONFLICT DO NOTHING;

-- System templates для Telegram

-- Monitor Down - Telegram
INSERT INTO templates (
	id, user_id, name, description, channel, type, engine,
	subject, body, format, is_default, is_system, version, created_at, updated_at
) VALUES (
	'00000000-0000-0000-0000-000000000005',
	'00000000-0000-0000-0000-000000000000',
	'Default Monitor Down (Telegram)',
	'System template for monitor down alerts',
	'telegram',
	'monitor_down',
	'gotemplate',
	'',
	'🚨 *{{ .monitor_name }} is DOWN*

*Status:* {{ .status }}
*URL:* {{ .monitor_url }}
*Error:* {{ .error_message }}
*Time:* {{ .timestamp }}',
	'text',
	true,
	true,
	1,
	NOW(),
	NOW()
) ON CONFLICT DO NOTHING;

-- Monitor Up - Telegram
INSERT INTO templates (
	id, user_id, name, description, channel, type, engine,
	subject, body, format, is_default, is_system, version, created_at, updated_at
) VALUES (
	'00000000-0000-0000-0000-000000000006',
	'00000000-0000-0000-0000-000000000000',
	'Default Monitor Up (Telegram)',
	'System template for monitor recovery notifications',
	'telegram',
	'monitor_up',
	'gotemplate',
	'',
	'✅ *{{ .monitor_name }} recovered*

*Status:* {{ .status }}
*URL:* {{ .monitor_url }}
*Time:* {{ .timestamp }}',
	'text',
	true,
	true,
	1,
	NOW(),
	NOW()
) ON CONFLICT DO NOTHING;

-- +goose Down
-- SQL в этой секции выполняется при откате миграции.

DELETE FROM templates WHERE user_id = '00000000-0000-0000-0000-000000000000';
