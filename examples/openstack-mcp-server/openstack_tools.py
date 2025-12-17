"""
OpenStack Tools Implementation
Implements the same tools as the ocstack Go project
"""

import subprocess
import os
import json
from typing import Dict, Any, Optional, List


class OpenStackTools:
    def __init__(self, default_namespace: str = "openstack"):
        self.default_namespace = default_namespace
        self.kubeconfig = os.environ.get('KUBECONFIG')
        if not self.kubeconfig:
            raise RuntimeError("KUBECONFIG environment variable is not set")
    
    def _run_oc_command(self, command: str, namespace: Optional[str] = None) -> Dict[str, Any]:
        """Execute oc command and return result"""
        ns = namespace or self.default_namespace
        cmd = f"oc -n {ns} {command}"
        
        try:
            result = subprocess.run(
                cmd.split(),
                capture_output=True,
                text=True,
                timeout=30
            )
            
            return {
                "stdout": result.stdout.strip(),
                "stderr": result.stderr.strip(),
                "exitcode": result.returncode
            }
        except subprocess.TimeoutExpired:
            return {
                "stdout": "",
                "stderr": "Command timed out after 30 seconds",
                "exitcode": 124
            }
        except Exception as e:
            return {
                "stdout": "",
                "stderr": str(e),
                "exitcode": 1
            }
    
    def hello(self, arguments: Dict[str, Any]) -> str:
        """Simple hello function for testing"""
        name = arguments.get("name", "World")
        return f"Hello {name}!"
    
    def oc(self, arguments: Dict[str, Any]) -> str:
        """Execute oc command"""
        command = arguments.get("command", "")
        if not command:
            return "Error: command argument is required"
        
        result = self._run_oc_command(command)
        return f"out: {result['stdout']}\nerr: {result['stderr']}\nexit: {result['exitcode']}"
    
    def get_openstack_control_plane(self, arguments: Dict[str, Any]) -> str:
        """Get OpenStack control plane information"""
        namespace = arguments.get("namespace", self.default_namespace)
        result = self._run_oc_command("get oscp", namespace)
        return f"out: {result['stdout']}\nerr: {result['stderr']}"
    
    def check_openstack_svc(self, arguments: Dict[str, Any]) -> str:
        """Check OpenStack service status"""
        service = arguments.get("service", "")
        namespace = arguments.get("namespace", self.default_namespace)
        
        if not service:
            return "Error: service argument is required"
        
        result = self._run_oc_command(f"get {service}", namespace)
        return f"out: {result['stdout']}\nerr: {result['stderr']}"
    
    def get_deployed_version(self, arguments: Dict[str, Any]) -> str:
        """Get deployed OpenStack version"""
        namespace = arguments.get("namespace", self.default_namespace)
        options = "-o custom-columns=VERSION:.status.deployedVersion --no-headers"
        result = self._run_oc_command(f"get openstackversion {options}", namespace)
        return f"out: {result['stdout']}\nerr: {result['stderr']}"
    
    def get_available_version(self, arguments: Dict[str, Any]) -> str:
        """Get available OpenStack version"""
        namespace = arguments.get("namespace", self.default_namespace)
        options = "-o custom-columns=VERSION:.status.availableVersion --no-headers"
        result = self._run_oc_command(f"get openstackversion {options}", namespace)
        return f"out: {result['stdout']}\nerr: {result['stderr']}"
    
    def needs_minor_update(self, arguments: Dict[str, Any]) -> str:
        """Check if OpenStack needs minor update"""
        namespace = arguments.get("namespace", self.default_namespace)
        
        # Get both versions
        av_result = self._run_oc_command(
            "get openstackversion -o custom-columns=VERSION:.status.availableVersion --no-headers", 
            namespace
        )
        dv_result = self._run_oc_command(
            "get openstackversion -o custom-columns=VERSION:.status.deployedVersion --no-headers", 
            namespace
        )
        
        if av_result['exitcode'] != 0 or dv_result['exitcode'] != 0:
            return "Error: Could not retrieve version information"
        
        available = av_result['stdout'].strip()
        deployed = dv_result['stdout'].strip()
        
        if available == deployed:
            return "OpenStack is up to date"
        else:
            return "OpenStack control plane update available!"
    
    def trigger_minor_update(self, arguments: Dict[str, Any]) -> str:
        """Trigger OpenStack control plane minor update"""
        namespace = arguments.get("namespace", self.default_namespace)
        target_version = arguments.get("targetVersion", "")
        openstack_version = arguments.get("openstackVersion", "")
        
        if not target_version:
            return "Error: targetVersion is required"
        
        if not openstack_version:
            return "Error: openstackVersion is required"
        
        # Build the patch JSON
        patch_data = {
            "spec": {
                "openstackVersion": openstack_version,
                "targetVersion": target_version
            }
        }
        
        patch_json = json.dumps(patch_data)
        
        # Execute oc patch command
        patch_command = f'patch openstackcontrolplane --type=merge --patch=\'{patch_json}\''
        result = self._run_oc_command(patch_command, namespace)
        
        if result['exitcode'] == 0:
            return f"Successfully triggered minor update to version {target_version} in namespace {namespace}. Output: {result['stdout']}"
        else:
            return f"Error triggering update: {result['stderr']}"
    
    def get_tool_definition(self, tool_name: str) -> Dict[str, Any]:
        """Get tool definition for MCP protocol"""
        tools = {
            "hello": {
                "name": "hello",
                "description": "Say hello to a given person with their name",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "name": {
                            "type": "string",
                            "description": "The name of the person"
                        }
                    },
                    "required": ["name"]
                }
            },
            "oc": {
                "name": "oc",
                "description": "Runs the OpenShift client (oc) to interact with an OpenShift environment",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "command": {
                            "type": "string",
                            "description": "The oc command to execute"
                        }
                    },
                    "required": ["command"]
                }
            },
            "get_openstack_control_plane": {
                "name": "get_openstack_control_plane",
                "description": "Get OpenStack control plane information",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "namespace": {
                            "type": "string",
                            "description": "The OpenStack namespace (optional)"
                        }
                    },
                    "required": []
                }
            },
            "check_openstack_svc": {
                "name": "check_openstack_svc",
                "description": "Check the status of an OpenStack service",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "service": {
                            "type": "string",
                            "description": "The service name to check"
                        },
                        "namespace": {
                            "type": "string",
                            "description": "The OpenStack namespace (optional)"
                        }
                    },
                    "required": ["service"]
                }
            },
            "get_deployed_version": {
                "name": "get_deployed_version",
                "description": "Get the currently deployed OpenStack version",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "namespace": {
                            "type": "string",
                            "description": "The OpenStack namespace (optional)"
                        }
                    },
                    "required": []
                }
            },
            "get_available_version": {
                "name": "get_available_version",
                "description": "Get the available OpenStack version for update",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "namespace": {
                            "type": "string",
                            "description": "The OpenStack namespace (optional)"
                        }
                    },
                    "required": []
                }
            },
            "needs_minor_update": {
                "name": "needs_minor_update",
                "description": "Check if OpenStack needs a minor update",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "namespace": {
                            "type": "string",
                            "description": "The OpenStack namespace (optional)"
                        }
                    },
                    "required": []
                }
            },
            "trigger_minor_update": {
                "name": "trigger_minor_update",
                "description": "Runs the openshift client (oc) to patch the control plane and trigger the openstack control plane minor update",
                "inputSchema": {
                    "type": "object",
                    "properties": {
                        "namespace": {
                            "type": "string",
                            "description": "The namespace of the openstack control plane"
                        },
                        "targetVersion": {
                            "type": "string",
                            "description": "The targetVersion that we need to update to"
                        },
                        "openstackVersion": {
                            "type": "string",
                            "description": "The name of the openstackVersion CR to patch"
                        }
                    },
                    "required": ["namespace", "targetVersion", "openstackVersion"]
                }
            }
        }
        return tools.get(tool_name, {})
    
    def get_all_tools(self) -> List[Dict[str, Any]]:
        """Get all available tools"""
        tool_names = [
            "hello", "oc", "get_openstack_control_plane", "check_openstack_svc",
            "get_deployed_version", "get_available_version", "needs_minor_update", "trigger_minor_update"
        ]
        return [self.get_tool_definition(name) for name in tool_names]
    
    def execute_tool(self, tool_name: str, arguments: Dict[str, Any]) -> str:
        """Execute a tool by name"""
        method = getattr(self, tool_name, None)
        if not method:
            return f"Error: Unknown tool '{tool_name}'"
        
        try:
            return method(arguments)
        except Exception as e:
            return f"Error executing tool '{tool_name}': {str(e)}"