package email

const htmlTemplate = `<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
</head>
<body style="margin:0;padding:0;background-color:#f0f0f0;font-family:'Helvetica Neue',Helvetica,Arial,sans-serif;">
<table width="100%" cellpadding="0" cellspacing="0" style="background-color:#f0f0f0;padding:40px 20px;">
<tr>
<td align="center">
<table width="600" cellpadding="0" cellspacing="0" style="background-color:#ffffff;border-radius:8px;overflow:hidden;box-shadow:0 2px 8px rgba(0,0,0,0.05);">

<tr><td style="background:{{.LevelColor}};height:6px;"></td></tr>

<tr>
<td style="padding:32px 40px;text-align:center;border-bottom:1px solid #e9ecef;">
<h1 style="margin:0 0 8px 0;font-size:24px;font-weight:600;color:#333;">🔔 Alert Notification</h1>
<p style="margin:0;font-size:14px;color:#666;">{{.ServiceName}}</p>
</td>
</tr>

<tr>
<td style="background:{{.LevelColor}};padding:16px 40px;">
<table width="100%" cellpadding="0" cellspacing="0">
<tr>
<td>
<span style="color:#fff;font-size:12px;text-transform:uppercase;letter-spacing:1px;opacity:0.9;">Alert Level</span><br>
<span style="color:#fff;font-size:20px;font-weight:600;">{{.Level}}</span>
</td>
<td align="right">
<span style="color:#fff;font-size:12px;text-transform:uppercase;letter-spacing:1px;opacity:0.9;">Time</span><br>
<span style="color:#fff;font-size:14px;">{{.Timestamp}}</span>
</td>
</tr>
</table>
</td>
</tr>

<tr>
<td style="padding:32px 40px;border-bottom:1px solid #e9ecef;">
<p style="margin:0 0 12px 0;font-size:11px;text-transform:uppercase;letter-spacing:1px;color:#999;font-weight:600;">Error Message</p>
<p style="margin:0;font-size:16px;color:#dc3545;line-height:1.6;word-break:break-word;">{{.Error}}</p>
</td>
</tr>

<tr>
<td style="padding:32px 40px;border-bottom:1px solid #e9ecef;">
<p style="margin:0 0 20px 0;font-size:11px;text-transform:uppercase;letter-spacing:1px;color:#999;font-weight:600;">Request Details</p>
<table width="100%" cellpadding="0" cellspacing="0">
<tr>
<td width="50%" style="padding:12px 0;vertical-align:top;">
<p style="margin:0 0 4px 0;font-size:12px;color:#999;">Service</p>
<p style="margin:0;font-size:14px;color:#333;font-weight:500;">{{.ServiceName}}</p>
</td>
<td width="50%" style="padding:12px 0;vertical-align:top;">
<p style="margin:0 0 4px 0;font-size:12px;color:#999;">Method</p>
<span style="display:inline-block;background:{{.MethodColor}};color:#fff;padding:4px 12px;border-radius:4px;font-size:12px;font-weight:600;">{{.Method}}</span>
</td>
</tr>
<tr>
<td colspan="2" style="padding:12px 0;border-top:1px solid #f0f0f0;">
<p style="margin:0 0 4px 0;font-size:12px;color:#999;">Path</p>
<p style="margin:0;font-size:14px;color:#333;font-family:'Courier New',monospace;word-break:break-all;">{{.Path}}</p>
</td>
</tr>
<tr>
<td width="50%" style="padding:12px 0;border-top:1px solid #f0f0f0;vertical-align:top;">
<p style="margin:0 0 4px 0;font-size:12px;color:#999;">Client IP</p>
<p style="margin:0;font-size:14px;color:#333;font-family:'Courier New',monospace;">{{.IP}}</p>
</td>
<td width="50%" style="padding:12px 0;border-top:1px solid #f0f0f0;vertical-align:top;">
<p style="margin:0 0 4px 0;font-size:12px;color:#999;">Source</p>
<p style="margin:0;font-size:14px;color:#333;font-family:'Courier New',monospace;">{{.Source}}</p>
</td>
</tr>
<tr>
<td colspan="2" style="padding:12px 0;border-top:1px solid #f0f0f0;">
<p style="margin:0 0 4px 0;font-size:12px;color:#999;">Request ID</p>
<p style="margin:0;font-size:13px;color:#666;font-family:'Courier New',monospace;word-break:break-all;">{{.RequestID}}</p>
</td>
</tr>
<tr>
<td colspan="2" style="padding:12px 0;border-top:1px solid #f0f0f0;">
<p style="margin:0 0 4px 0;font-size:12px;color:#999;">User Agent</p>
<p style="margin:0;font-size:12px;color:#888;word-break:break-all;">{{.UserAgent}}</p>
</td>
</tr>
</table>
</td>
</tr>

<tr>
<td style="padding:32px 40px;">
<p style="margin:0 0 16px 0;font-size:11px;text-transform:uppercase;letter-spacing:1px;color:#999;font-weight:600;">Stack Trace</p>
<table width="100%" cellpadding="0" cellspacing="0" style="border:1px solid #e9ecef;border-radius:6px;overflow:hidden;">
{{range .Stack}}
<tr><td style="padding:8px 16px;font-family:'Courier New',monospace;font-size:13px;color:#555;background:#f8f9fa;border-bottom:1px solid #e9ecef;">{{.}}</td></tr>
{{else}}
<tr><td style="padding:16px;color:#888;text-align:center;background:#f8f9fa;">No stack trace available</td></tr>
{{end}}
</table>
</td>
</tr>

<tr>
<td style="background:#f8f9fa;padding:24px 40px;text-align:center;border-top:1px solid #e9ecef;">
<p style="margin:0 0 8px 0;font-size:13px;color:#666;">Sent by <strong>Go Logging Library</strong></p>
<p style="margin:0;font-size:11px;color:#999;">This is an automated alert notification.</p>
</td>
</tr>

</table>

<table width="600" cellpadding="0" cellspacing="0">
<tr>
<td style="padding:24px 40px;text-align:center;">
<p style="margin:0;font-size:11px;color:#999;">© {{.Year}} {{.ServiceName}} • Alert System</p>
</td>
</tr>
</table>

</td>
</tr>
</table>
</body>
</html>`

type templateData struct {
	LevelColor  string
	MethodColor string
	Level       string
	ServiceName string
	Timestamp   string
	Error       string
	Method      string
	Path        string
	IP          string
	Source      string
	RequestID   string
	UserAgent   string
	Stack       []string
	Year        int
}
