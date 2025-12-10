"""
MCP Protocol Types and Models
Based on Model Context Protocol specification
"""

from typing import Dict, List, Optional, Any, Union
from pydantic import BaseModel


class JSONRPCRequest(BaseModel):
    jsonrpc: str = "2.0"
    id: Optional[Union[str, int]] = None
    method: str
    params: Optional[Dict[str, Any]] = None


class JSONRPCResponse(BaseModel):
    jsonrpc: str = "2.0"
    id: Optional[Union[str, int]] = None
    result: Optional[Dict[str, Any]] = None
    error: Optional[Dict[str, Any]] = None


class JSONRPCError(BaseModel):
    code: int
    message: str
    data: Optional[Any] = None


class ClientCapabilities(BaseModel):
    roots: Optional[Dict[str, Any]] = None
    sampling: Optional[Dict[str, Any]] = None


class ClientInfo(BaseModel):
    name: str
    version: str


class InitializeRequest(BaseModel):
    protocolVersion: str
    capabilities: ClientCapabilities
    clientInfo: ClientInfo


class ServerCapabilities(BaseModel):
    tools: Optional[Dict[str, Any]] = None
    resources: Optional[Dict[str, Any]] = None
    prompts: Optional[Dict[str, Any]] = None
    logging: Optional[Dict[str, Any]] = None


class ServerInfo(BaseModel):
    name: str
    version: str


class InitializeResponse(BaseModel):
    protocolVersion: str
    capabilities: ServerCapabilities
    serverInfo: ServerInfo


class ToolSchema(BaseModel):
    type: str = "object"
    properties: Dict[str, Any] = {}
    required: List[str] = []
    description: Optional[str] = None


class Tool(BaseModel):
    name: str
    description: str
    inputSchema: ToolSchema


class ListToolsResponse(BaseModel):
    tools: List[Tool]


class CallToolRequest(BaseModel):
    name: str
    arguments: Optional[Dict[str, Any]] = None


class ToolResult(BaseModel):
    type: str = "text"
    text: str
    mimeType: Optional[str] = None


class CallToolResponse(BaseModel):
    content: List[ToolResult]
    isError: bool = False