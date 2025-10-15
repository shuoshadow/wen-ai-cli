#!/usr/bin/env python3
"""
本地系统工具 MCP Server
提供查询本地服务状态、进程信息等功能
"""
import asyncio
import subprocess
import json
from mcp.server import Server
from mcp.types import Tool, TextContent

# 创建 MCP Server
server = Server("local-system")

@server.list_tools()
async def list_tools() -> list[Tool]:
    """列出所有可用工具"""
    return [
        Tool(
            name="check_service_status",
            description="查询系统服务状态（支持 systemctl）",
            inputSchema={
                "type": "object",
                "properties": {
                    "service_name": {
                        "type": "string",
                        "description": "服务名称，例如：nginx, docker, ssh"
                    }
                },
                "required": ["service_name"]
            }
        ),
        Tool(
            name="check_process",
            description="查询进程信息",
            inputSchema={
                "type": "object",
                "properties": {
                    "process_name": {
                        "type": "string",
                        "description": "进程名称或关键字"
                    }
                },
                "required": ["process_name"]
            }
        ),
        Tool(
            name="check_port",
            description="查询端口占用情况",
            inputSchema={
                "type": "object",
                "properties": {
                    "port": {
                        "type": "number",
                        "description": "端口号"
                    }
                },
                "required": ["port"]
            }
        ),
        Tool(
            name="disk_usage",
            description="查询磁盘使用情况",
            inputSchema={
                "type": "object",
                "properties": {
                    "path": {
                        "type": "string",
                        "description": "路径，默认为根目录"
                    }
                }
            }
        )
    ]

@server.call_tool()
async def call_tool(name: str, arguments: dict):
    """处理工具调用"""
    try:
        if name == "check_service_status":
            service_name = arguments.get("service_name")
            result = subprocess.run(
                ["systemctl", "status", service_name],
                capture_output=True,
                text=True,
                timeout=5
            )

            # 提取关键信息
            output_lines = result.stdout.split('\n')
            summary = []
            for line in output_lines[:10]:  # 只取前10行
                if line.strip():
                    summary.append(line)

            status_text = '\n'.join(summary)
            return [TextContent(
                type="text",
                text=f"服务 {service_name} 状态：\n{status_text}"
            )]

        elif name == "check_process":
            process_name = arguments.get("process_name")
            result = subprocess.run(
                ["ps", "aux"],
                capture_output=True,
                text=True,
                timeout=5
            )

            # 过滤包含进程名的行
            lines = result.stdout.split('\n')
            matched = [lines[0]]  # 保留表头
            for line in lines[1:]:
                if process_name.lower() in line.lower():
                    matched.append(line)

            if len(matched) <= 1:
                return [TextContent(
                    type="text",
                    text=f"未找到进程：{process_name}"
                )]

            return [TextContent(
                type="text",
                text=f"找到 {len(matched)-1} 个相关进程：\n" + '\n'.join(matched[:6])
            )]

        elif name == "check_port":
            port = arguments.get("port")
            result = subprocess.run(
                ["lsof", "-i", f":{port}"],
                capture_output=True,
                text=True,
                timeout=5
            )

            if result.returncode != 0 or not result.stdout.strip():
                return [TextContent(
                    type="text",
                    text=f"端口 {port} 未被占用"
                )]

            return [TextContent(
                type="text",
                text=f"端口 {port} 占用情况：\n{result.stdout}"
            )]

        elif name == "disk_usage":
            path = arguments.get("path", "/")
            result = subprocess.run(
                ["df", "-h", path],
                capture_output=True,
                text=True,
                timeout=5
            )

            return [TextContent(
                type="text",
                text=f"磁盘使用情况：\n{result.stdout}"
            )]

        else:
            return [TextContent(
                type="text",
                text=f"未知工具：{name}"
            )]

    except subprocess.TimeoutExpired:
        return [TextContent(
            type="text",
            text=f"执行超时"
        )]
    except Exception as e:
        return [TextContent(
            type="text",
            text=f"执行失败：{str(e)}"
        )]

async def main():
    """启动 MCP Server"""
    from mcp.server.stdio import stdio_server

    async with stdio_server() as (read_stream, write_stream):
        await server.run(
            read_stream,
            write_stream,
            server.create_initialization_options()
        )

if __name__ == "__main__":
    asyncio.run(main())

