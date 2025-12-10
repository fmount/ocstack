"""
OpenStack MCP HTTP Server
A standalone HTTP-based Model Context Protocol server for OpenStack tools
"""

import os
import json
import logging
from typing import Dict, Any, Optional
from fastapi import FastAPI, HTTPException, Request
from fastapi.responses import JSONResponse
import uvicorn

from mcp_types import (
    JSONRPCRequest, JSONRPCResponse, JSONRPCError,
    InitializeRequest, InitializeResponse, ServerCapabilities, ServerInfo,
    ListToolsResponse, CallToolRequest, CallToolResponse, ToolResult,
    Tool, ToolSchema
)
from openstack_tools import OpenStackTools

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
MCP_HOST = os.environ.get('MCP_HOST', 'localhost')
MCP_PORT = int(os.environ.get('MCP_PORT', '8080'))
DEFAULT_NAMESPACE = os.environ.get('DEFAULT_NAMESPACE', 'openstack')

app = FastAPI(
    title="OpenStack MCP Server",
    description="Model Context Protocol server for OpenStack tools",
    version="1.0.0"
)

# Initialize OpenStack tools
try:
    openstack_tools = OpenStackTools(default_namespace=DEFAULT_NAMESPACE)
    logger.info(f"OpenStack tools initialized with default namespace: {DEFAULT_NAMESPACE}")
except Exception as e:
    logger.error(f"Failed to initialize OpenStack tools: {e}")
    openstack_tools = None


class MCPServer:
    def __init__(self):
        self.protocol_version = "2024-11-05"
        self.server_info = ServerInfo(
            name="openstack-mcp-server",
            version="1.0.0"
        )
        self.capabilities = ServerCapabilities(
            tools={"listChanged": True}
        )
    
    def handle_initialize(self, params: InitializeRequest) -> InitializeResponse:
        """Handle MCP initialize request"""
        logger.info(f"Initialize request from {params.clientInfo.name} v{params.clientInfo.version}")
        
        return InitializeResponse(
            protocolVersion=self.protocol_version,
            capabilities=self.capabilities,
            serverInfo=self.server_info
        )
    
    def handle_tools_list(self, params: Optional[Dict[str, Any]] = None) -> ListToolsResponse:
        """Handle tools/list request"""
        if not openstack_tools:
            return ListToolsResponse(tools=[])
        
        tools_data = openstack_tools.get_all_tools()
        tools = []
        
        for tool_data in tools_data:
            if tool_data:  # Skip empty tool definitions
                tool = Tool(
                    name=tool_data["name"],
                    description=tool_data["description"],
                    inputSchema=ToolSchema(**tool_data["inputSchema"])
                )
                tools.append(tool)
        
        logger.info(f"Returning {len(tools)} tools")
        return ListToolsResponse(tools=tools)
    
    def handle_tools_call(self, params: CallToolRequest) -> CallToolResponse:
        """Handle tools/call request"""
        if not openstack_tools:
            return CallToolResponse(
                content=[ToolResult(text="OpenStack tools not available")],
                isError=True
            )
        
        tool_name = params.name
        arguments = params.arguments or {}
        
        logger.info(f"Executing tool: {tool_name} with args: {arguments}")
        
        try:
            result_text = openstack_tools.execute_tool(tool_name, arguments)
            
            return CallToolResponse(
                content=[ToolResult(text=result_text)],
                isError=False
            )
        
        except Exception as e:
            error_msg = f"Error executing tool {tool_name}: {str(e)}"
            logger.error(error_msg)
            
            return CallToolResponse(
                content=[ToolResult(text=error_msg)],
                isError=True
            )
    
    def process_request(self, request: JSONRPCRequest) -> JSONRPCResponse:
        """Process MCP JSON-RPC request"""
        try:
            if request.method == "initialize":
                params = InitializeRequest(**request.params)
                result = self.handle_initialize(params)
                return JSONRPCResponse(
                    id=request.id,
                    result=result.model_dump()
                )
            
            elif request.method == "notifications/initialized":
                # Client notification that initialization is complete
                logger.info("Client initialization complete")
                return JSONRPCResponse(id=request.id, result={})
            
            elif request.method == "tools/list":
                result = self.handle_tools_list(request.params)
                return JSONRPCResponse(
                    id=request.id,
                    result=result.model_dump()
                )
            
            elif request.method == "tools/call":
                params = CallToolRequest(**request.params)
                result = self.handle_tools_call(params)
                return JSONRPCResponse(
                    id=request.id,
                    result=result.model_dump()
                )
            
            else:
                raise HTTPException(
                    status_code=400,
                    detail=f"Unknown method: {request.method}"
                )
        
        except Exception as e:
            logger.error(f"Error processing request: {e}")
            error = JSONRPCError(code=-32603, message="Internal error", data=str(e))
            return JSONRPCResponse(
                id=request.id,
                error=error.model_dump()
            )


# Create MCP server instance
mcp_server = MCPServer()


@app.post("/mcp")
async def mcp_endpoint(request: Request):
    """Main MCP protocol endpoint"""
    try:
        body = await request.json()
        logger.debug(f"Received request: {body}")
        
        # Parse JSON-RPC request
        rpc_request = JSONRPCRequest(**body)
        
        # Process request
        response = mcp_server.process_request(rpc_request)
        
        logger.debug(f"Sending response: {response}")
        return JSONResponse(content=response.model_dump(exclude_none=True))
    
    except Exception as e:
        logger.error(f"Error in MCP endpoint: {e}")
        error_response = JSONRPCResponse(
            error=JSONRPCError(code=-32700, message="Parse error").model_dump()
        )
        return JSONResponse(
            content=error_response.model_dump(exclude_none=True),
            status_code=400
        )


@app.get("/health")
async def health_check():
    """Health check endpoint"""
    return {
        "status": "healthy",
        "server": "openstack-mcp-server",
        "version": "1.0.0",
        "openstack_tools_available": openstack_tools is not None
    }


@app.get("/tools")
async def list_tools_debug():
    """Debug endpoint to list available tools"""
    if not openstack_tools:
        return {"error": "OpenStack tools not available"}
    
    tools = openstack_tools.get_all_tools()
    return {"tools": tools}


@app.get("/")
async def root():
    """Root endpoint with server information"""
    return {
        "name": "OpenStack MCP Server",
        "version": "1.0.0",
        "protocol": "MCP over HTTP",
        "endpoints": {
            "mcp": "/mcp",
            "health": "/health",
            "tools": "/tools"
        },
        "usage": "POST JSON-RPC requests to /mcp endpoint"
    }


if __name__ == "__main__":
    logger.info(f"Starting OpenStack MCP Server on {MCP_HOST}:{MCP_PORT}")
    logger.info(f"Default namespace: {DEFAULT_NAMESPACE}")
    logger.info(f"KUBECONFIG: {os.environ.get('KUBECONFIG', 'Not set')}")
    
    uvicorn.run(
        app,
        host=MCP_HOST,
        port=MCP_PORT,
        log_level="info"
    )