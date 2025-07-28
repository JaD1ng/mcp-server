package multiplexer

import "html/template"

// htmlTemplate 预编译的HTML模板
var htmlTemplate = template.Must(template.New("index").Parse(`<!DOCTYPE html>
<html>
<head>
    <title>MCP服务器</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .endpoint { background: #f5f5f5; padding: 20px; margin: 20px 0; border-radius: 5px; }
        .endpoint h3 { margin-top: 0; color: #333; }
        .endpoint a { color: #007bff; text-decoration: none; }
        .endpoint a:hover { text-decoration: underline; }
        .status { padding: 5px 10px; border-radius: 3px; font-size: 12px; font-weight: bold; }
        .status.available { background: #d4edda; color: #155724; }
        .status.unavailable { background: #f8d7da; color: #721c24; }
        .server-addresses { background: #e9ecef; padding: 10px; margin: 10px 0; border-radius: 3px; font-family: monospace; }
        .service-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 20px; }
        .tools-list { margin: 10px 0; }
        .tools-list li { margin: 5px 0; font-size: 14px; }
    </style>
</head>
<body>
    <h1>MCP服务器</h1>
    <p>欢迎使用MCP服务器。以下是可用的端点：</p>
    
    <div class="server-addresses">
        <strong>服务器地址:</strong><br>
        {{range .ServerAddresses}}• http://{{.}}:{{$.Port}}<br>{{end}}
    </div>

    {{if .Services}}
    <div class="service-grid">
        {{range .Services}}
        <div class="endpoint">
            <h3>{{.Type}} MCP服务器 
                {{if .Available}}
                <span class="status available">可用</span>
                {{else}}
                <span class="status unavailable">不可用</span>
                {{end}}
            </h3>
            <p>{{.Description}}</p>
            <p><strong>端点:</strong> <a href="{{.Endpoint}}">{{.Endpoint}}</a></p>
            {{if .Tools}}
            <p><strong>可用工具:</strong></p>
            <ul class="tools-list">
                {{range .Tools}}
                <li>{{.}}</li>
                {{end}}
            </ul>
            {{end}}
        </div>
        {{end}}
    </div>
    {{else}}
    <div class="endpoint">
        <h3>无可用服务 <span class="status unavailable">暂无服务</span></h3>
        <p>当前没有注册任何MCP服务</p>
    </div>
    {{end}}

    <div style="margin-top: 40px; padding: 20px; background: #f8f9fa; border-radius: 5px;">
        <h4>关于MCP服务器</h4>
        <p>此服务器支持多个MCP（Model Context Protocol）服务的动态注册和路由。</p>
        <p>每个服务都提供特定的工具和功能，可以通过对应的端点访问。</p>
        <p>服务状态和连接信息会实时更新。</p>
    </div>
</body>
</html>`))
