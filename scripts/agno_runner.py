#!/usr/bin/env python
# -*- coding: utf-8 -*-
"""
Agno Runner - Python bridge for Xgent-go to use Agno SDK
Supports: Bot, Team, MCP Tools, Session management, Task cancellation,
          Session reuse, ThinkingStep, Non-streaming, Git download, Debug mode
"""

import json
import os
import sys
import asyncio
import signal
import subprocess
import tempfile
from typing import Optional, Dict, Any, List, Union
from dataclasses import dataclass, field, asdict
from datetime import datetime

# Ensure agno is importable
try:
    from agno.agent import Agent, RunEvent
    from agno.team import Team
    from agno.team.team import TeamRunEvent
    from agno.models.openai import OpenAIChat
    from agno.models.anthropic import Claude
    from agno.db.sqlite import SqliteDb
except ImportError as e:
    print(json.dumps({"type": "error", "content": f"agno module not found: {e}. Please install agno."}), flush=True)
    sys.exit(1)

# Optional: Gemini support
try:
    from agno.models.google import Gemini
    from google.genai import Client
    from google.genai.types import HttpOptions
    GEMINI_AVAILABLE = True
except ImportError:
    GEMINI_AVAILABLE = False

# Optional: MCP Tools support
try:
    from agno.tools.mcp import MCPTools
    from agno.tools.mcp import StreamableHTTPClientParams, SSEClientParams, StdioServerParameters
    MCP_AVAILABLE = True
except ImportError:
    MCP_AVAILABLE = False

# Global state
cancelled = False
current_run_id = None
db = SqliteDb(db_file="/tmp/agno_xgent.db")

# Session cache for reusing Agent/Team instances
_clients: Dict[str, Union[Agent, Team]] = {}


# ============================================================================
# ThinkingStep System
# ============================================================================

@dataclass
class ThinkingStep:
    """Represents a thinking step in the execution process"""
    title: str
    timestamp: str = field(default_factory=lambda: datetime.now().isoformat())
    progress: int = 0
    details: Dict[str, Any] = field(default_factory=dict)
    
    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)


class ThinkingStepManager:
    """Manages thinking steps for execution progress reporting"""
    
    def __init__(self):
        self._steps: List[ThinkingStep] = []
        self._current_progress: int = 0
    
    def add_step(self, title: str, details: Dict[str, Any] = None, report: bool = True) -> ThinkingStep:
        """Add a thinking step"""
        step = ThinkingStep(
            title=title,
            progress=self._current_progress,
            details=details or {}
        )
        self._steps.append(step)
        if report:
            emit_event("thinking_step", title, {"step": step.to_dict()})
        return step
    
    def update_progress(self, progress: int):
        """Update current progress"""
        self._current_progress = progress
    
    def get_steps(self) -> List[Dict[str, Any]]:
        """Get all steps as dictionaries"""
        return [s.to_dict() for s in self._steps]
    
    def clear(self):
        """Clear all steps"""
        self._steps.clear()


# Global thinking step manager
thinking_manager = ThinkingStepManager()


def emit_event(event_type: str, content: str = "", details: Dict[str, Any] = None):
    """Emit a JSON event to stdout"""
    event = {"type": event_type, "content": content}
    if details:
        event["details"] = details
    print(json.dumps(event, ensure_ascii=False), flush=True)


def handle_signal(signum, frame):
    """Handle cancellation signal"""
    global cancelled
    cancelled = True
    emit_event("cancelled", "Task cancelled by signal")


# Register signal handlers
signal.signal(signal.SIGTERM, handle_signal)
signal.signal(signal.SIGINT, handle_signal)


# ============================================================================
# Git Download Support
# ============================================================================

def download_code(git_url: str, branch: str = None) -> Optional[str]:
    """
    Download code from git repository
    
    Args:
        git_url: Git repository URL
        branch: Optional branch name
        
    Returns:
        str: Path to downloaded code, or None if failed
    """
    if not git_url:
        return None
    
    try:
        thinking_manager.add_step("Downloading code from repository", {"git_url": git_url})
        
        # Create temp directory
        temp_dir = tempfile.mkdtemp(prefix="xgent_code_")
        
        # Build git clone command
        cmd = ["git", "clone", "--depth", "1"]
        if branch:
            cmd.extend(["-b", branch])
        cmd.extend([git_url, temp_dir])
        
        # Execute git clone
        result = subprocess.run(
            cmd,
            capture_output=True,
            text=True,
            timeout=300  # 5 minute timeout
        )
        
        if result.returncode != 0:
            emit_event("warning", f"Git clone failed: {result.stderr}")
            return None
        
        thinking_manager.add_step("Code downloaded successfully", {"path": temp_dir})
        emit_event("git_downloaded", f"Code downloaded to {temp_dir}", {"path": temp_dir})
        return temp_dir
        
    except subprocess.TimeoutExpired:
        emit_event("warning", "Git clone timed out")
        return None
    except Exception as e:
        emit_event("warning", f"Failed to download code: {str(e)}")
        return None


def create_model(model_config: Dict[str, Any]):
    """Create a model instance from config"""
    provider = model_config.get("provider", "openai")
    model_id = model_config.get("model_id", "gpt-4")
    api_key = model_config.get("api_key")
    base_url = model_config.get("base_url")
    
    # Log model configuration (mask API key)
    masked_key = f"{api_key[:10]}...{api_key[-4:]}" if api_key and len(api_key) > 14 else "<not set>"
    emit_event("debug", f"Creating model: provider={provider}, model={model_id}, base_url={base_url}, api_key={masked_key}")
    
    if provider == "openai":
        # Configure proxy for OpenAI client
        import httpx
        http_client = None
        
        proxy_url = os.environ.get("HTTPS_PROXY") or os.environ.get("HTTP_PROXY") or os.environ.get("https_proxy") or os.environ.get("http_proxy")
        if proxy_url:
            emit_event("debug", f"Configuring OpenAI with proxy: {proxy_url}")
            http_client = httpx.AsyncClient(proxy=proxy_url, timeout=60.0)
        
        return OpenAIChat(
            id=model_id, 
            api_key=api_key, 
            base_url=base_url,
            max_tokens=4096,
            http_client=http_client,
        )
    elif provider == "anthropic" or provider == "claude":
        # Configure proxy for Anthropic
        import httpx
        http_client = None
        
        if base_url:
            os.environ["ANTHROPIC_BASE_URL"] = base_url
        
        proxy_url = os.environ.get("HTTPS_PROXY") or os.environ.get("HTTP_PROXY") or os.environ.get("https_proxy") or os.environ.get("http_proxy")
        if proxy_url:
            emit_event("debug", f"Configuring Anthropic with proxy: {proxy_url}")
            http_client = httpx.AsyncClient(proxy=proxy_url, timeout=60.0)
        
        return Claude(
            id=model_id, 
            api_key=api_key,
            max_tokens=32768,
            http_client=http_client,
        )
    elif provider == "gemini" or provider == "google":
        if not GEMINI_AVAILABLE:
            raise ValueError("Gemini support not available. Install google-genai package.")
        
        if base_url:
            # Custom base URL for Gemini
            headers = {"x-goog-api-key": api_key} if api_key else {}
            base_url_stripped = base_url.rstrip("/")
            api_version = "v1beta"
            
            if "/v1beta" in base_url_stripped:
                base_url_stripped = base_url_stripped.replace("/v1beta", "")
            elif "/v1" in base_url_stripped:
                base_url_stripped = base_url_stripped.replace("/v1", "")
                api_version = "v1"
            
            http_options = HttpOptions(
                base_url=base_url_stripped,
                api_version=api_version,
                headers=headers,
            )
            client = Client(api_key=api_key, http_options=http_options)
            return Gemini(id=model_id, client=client)
        else:
            return Gemini(id=model_id, api_key=api_key)
    else:
        raise ValueError(f"Unsupported provider: {provider}")


async def setup_mcp_tools(mcp_config: List[Dict[str, Any]]) -> List:
    """Setup MCP tools from configuration"""
    if not MCP_AVAILABLE:
        emit_event("warning", "MCP tools not available. Install agno[mcp] package.")
        return []
    
    if not mcp_config:
        return []
    
    tools = []
    for server_config in mcp_config:
        try:
            mcp_type = server_config.get("type", "stdio")
            timeout_seconds = server_config.get("timeout", 300)
            
            if mcp_type == "streamable-http" or mcp_type == "streamable_http":
                url = server_config.get("url")
                if not url:
                    continue
                from datetime import timedelta
                server_params = StreamableHTTPClientParams(
                    url=url,
                    headers=server_config.get("headers", {}),
                    timeout=timedelta(seconds=timeout_seconds),
                    sse_read_timeout=timedelta(seconds=server_config.get("sse_read_timeout", 300)),
                )
                mcp_tool = MCPTools(transport="streamable-http", server_params=server_params, timeout_seconds=timeout_seconds)
                
            elif mcp_type == "sse":
                url = server_config.get("url")
                if not url:
                    continue
                server_params = SSEClientParams(
                    url=url,
                    headers=server_config.get("headers", {}),
                    timeout=timeout_seconds,
                    sse_read_timeout=server_config.get("sse_read_timeout", 300),
                )
                mcp_tool = MCPTools(transport="sse", server_params=server_params, timeout_seconds=timeout_seconds)
                
            elif mcp_type == "stdio":
                command = server_config.get("command")
                if not command:
                    continue
                server_params = StdioServerParameters(
                    env=server_config.get("env", {}),
                    args=server_config.get("args", []),
                    command=command,
                )
                mcp_tool = MCPTools(transport="stdio", server_params=server_params, timeout_seconds=timeout_seconds)
            else:
                emit_event("warning", f"Unsupported MCP type: {mcp_type}")
                continue
            
            await mcp_tool.connect()
            tools.append(mcp_tool)
            emit_event("mcp_connected", f"Connected to MCP server: {server_config.get('name', mcp_type)}")
            
        except Exception as e:
            emit_event("warning", f"Failed to setup MCP tool: {str(e)}")
    
    return tools


async def cleanup_mcp_tools(tools: List):
    """Cleanup MCP tools"""
    for tool in tools:
        try:
            if hasattr(tool, 'disconnect'):
                await tool.disconnect()
        except Exception:
            pass


def get_mode_config(mode: str) -> Dict[str, Any]:
    """Get team mode configuration"""
    if mode == "coordinate":
        return {"reasoning": True}
    elif mode == "collaborate":
        return {"delegate_task_to_all_members": True, "reasoning": True}
    elif mode == "route":
        return {"respond_directly": True}
    else:
        return {}


def build_prompt_with_context(prompt: str, context: Dict[str, Any]) -> str:
    """Build prompt with additional context (cwd, git_url, etc.)"""
    parts = [prompt]
    
    if context.get("cwd"):
        parts.append(f"\nCurrent working directory: {context['cwd']}")
    if context.get("git_url"):
        parts.append(f"\nProject URL: {context['git_url']}")
    if context.get("project_path"):
        parts.append(f"\nProject path: {context['project_path']}")
    
    return "".join(parts)


def create_agent_from_member(member_config: Dict[str, Any], mcp_tools: List = None) -> Agent:
    """Create an Agent from member config"""
    model = create_model(member_config.get("model", {}))
    
    tools = mcp_tools if mcp_tools else []
    
    return Agent(
        name=member_config.get("name", "Agent"),
        model=model,
        instructions=member_config.get("personality", ""),
        description=member_config.get("description", member_config.get("personality", "")),
        tools=tools,
        markdown=True,
    )


async def handle_agent_event(event, result_content: str) -> str:
    """Handle agent streaming events"""
    global current_run_id
    
    if event.event == RunEvent.run_started:
        if hasattr(event, 'run_id'):
            current_run_id = event.run_id
        emit_event("run_started", f"Agent run started: {event.agent_id}")
    
    elif event.event == RunEvent.run_completed:
        emit_event("run_completed", f"Agent run completed: {event.agent_id}")
    
    elif event.event == RunEvent.tool_call_started:
        emit_event("tool_call_started", "", {
            "tool_name": event.tool.tool_name,
            "tool_args": event.tool.tool_args,
        })
    
    elif event.event == RunEvent.tool_call_completed:
        result_preview = event.tool.result[:200] if event.tool.result else ""
        emit_event("tool_call_completed", "", {
            "tool_name": event.tool.tool_name,
            "result": result_preview,
        })
    
    elif event.event == RunEvent.run_content:
        if event.content:
            result_content += str(event.content)
            emit_event("content", str(event.content))
    
    return result_content


async def handle_team_event(event, result_content: str) -> str:
    """Handle team streaming events"""
    global current_run_id
    
    # Team-level events
    if hasattr(TeamRunEvent, 'run_started') and event.event == TeamRunEvent.run_started:
        if hasattr(event, 'run_id'):
            current_run_id = event.run_id
        emit_event("team_run_started", "Team run started")
    
    elif hasattr(TeamRunEvent, 'run_completed') and event.event == TeamRunEvent.run_completed:
        emit_event("team_run_completed", "Team run completed")
    
    elif hasattr(TeamRunEvent, 'tool_call_started') and event.event == TeamRunEvent.tool_call_started:
        emit_event("tool_call_started", "", {
            "tool_name": event.tool.tool_name,
            "tool_args": event.tool.tool_args,
        })
    
    elif hasattr(TeamRunEvent, 'tool_call_completed') and event.event == TeamRunEvent.tool_call_completed:
        result_preview = event.tool.result[:200] if event.tool.result else ""
        emit_event("tool_call_completed", "", {
            "tool_name": event.tool.tool_name,
            "result": result_preview,
        })
    
    elif hasattr(TeamRunEvent, 'run_content') and event.event == TeamRunEvent.run_content:
        if event.content:
            result_content += str(event.content)
            emit_event("content", str(event.content))
    
    # Member-level events (RunEvent)
    elif event.event == RunEvent.tool_call_started:
        emit_event("member_tool_started", "", {
            "agent_id": event.agent_id,
            "tool_name": event.tool.tool_name,
            "tool_args": event.tool.tool_args,
        })
    
    elif event.event == RunEvent.tool_call_completed:
        result_preview = event.tool.result[:200] if event.tool.result else ""
        emit_event("member_tool_completed", "", {
            "agent_id": event.agent_id,
            "tool_name": event.tool.tool_name,
            "result": result_preview,
        })
    
    # Reasoning step
    if hasattr(event, 'event') and str(event.event) == "TeamReasoningStep":
        if event.content:
            emit_event("reasoning", "", {
                "title": getattr(event.content, 'title', ''),
                "action": getattr(event.content, 'action', ''),
                "reasoning": getattr(event.content, 'reasoning', ''),
            })
    
    # Member response events
    elif hasattr(event, 'event') and 'response' in str(event.event).lower():
        emit_event("member_response", "", {
            "event_type": str(event.event) if hasattr(event, 'event') else 'unknown',
            "agent_id": getattr(event, 'agent_id', 'unknown'),
            "content": str(getattr(event, 'content', ''))[:200],
        })
    
    # Catch-all for debugging
    elif hasattr(event, 'agent_id') and event.agent_id:
        emit_event("member_activity", "", {
            "event_type": str(event.event) if hasattr(event, 'event') else 'unknown',
            "agent_id": event.agent_id,
            "details": str(event)[:200],
        })
    
    return result_content


async def run_bot(data: Dict[str, Any]):
    """Run a single bot/agent with session reuse, streaming/non-streaming, debug mode"""
    global cancelled, _clients
    
    prompt = data.get("prompt", "")
    model_config = data.get("model", {})
    ghost_config = data.get("ghost", {})
    context = data.get("context", {})
    session_id = data.get("session_id", "default")
    mcp_config = data.get("mcp_tools", [])
    
    # New options
    enable_streaming = data.get("stream", True)
    debug_mode = data.get("debug", False)
    debug_level = data.get("debug_level", 2)
    reuse_session = data.get("reuse_session", True)
    git_url = context.get("git_url")
    branch = context.get("branch")
    
    thinking_manager.clear()
    thinking_manager.update_progress(10)
    thinking_manager.add_step("Initializing agent")
    
    # Download code if git_url provided
    project_path = None
    if git_url:
        project_path = download_code(git_url, branch)
        if project_path:
            context["project_path"] = project_path
    
    # Setup MCP tools
    thinking_manager.update_progress(20)
    mcp_tools = await setup_mcp_tools(mcp_config)
    
    try:
        # Check for cached agent (session reuse)
        agent = None
        if reuse_session and session_id in _clients:
            cached = _clients[session_id]
            if isinstance(cached, Agent):
                agent = cached
                thinking_manager.add_step("Reusing existing agent session", {"session_id": session_id})
                emit_event("session_reused", f"Reusing agent session: {session_id}")
        
        # Create new agent if not cached
        if agent is None:
            thinking_manager.update_progress(30)
            thinking_manager.add_step("Creating new agent")
            
            model = create_model(model_config)
            agent = Agent(
                model=model,
                instructions=ghost_config.get("personality", ""),
                name=ghost_config.get("name", "Agent"),
                tools=mcp_tools,
                markdown=True,
            )
            
            # Cache for reuse
            if reuse_session:
                _clients[session_id] = agent
        
        # Build prompt with context
        full_prompt = build_prompt_with_context(prompt, context)
        
        thinking_manager.update_progress(50)
        thinking_manager.add_step("Executing agent", {"streaming": enable_streaming})
        
        if enable_streaming:
            # Streaming mode
            result_content = ""
            async for event in agent.arun(
                full_prompt,
                stream=True,
                stream_intermediate_steps=True,
                session_id=session_id,
                user_id=session_id,
                debug_mode=debug_mode,
            ):
                if cancelled:
                    emit_event("cancelled", "Task was cancelled")
                    break
                
                result_content = await handle_agent_event(event, result_content)
        else:
            # Non-streaming mode
            thinking_manager.add_step("Running in non-streaming mode")
            result = await agent.arun(
                full_prompt,
                stream=False,
                session_id=session_id,
                user_id=session_id,
                debug_mode=debug_mode,
            )
            
            # Extract content from result
            if hasattr(result, "content") and result.content:
                result_content = str(result.content)
            elif hasattr(result, "to_dict"):
                result_content = json.dumps(result.to_dict(), ensure_ascii=False)
            else:
                result_content = str(result)
        
        thinking_manager.update_progress(100)
        if not cancelled:
            thinking_manager.add_step("Execution completed")
            emit_event("completed", result_content, {"thinking_steps": thinking_manager.get_steps()})
            
    finally:
        await cleanup_mcp_tools(mcp_tools)


async def run_team(data: Dict[str, Any]):
    """Run a team of agents with session reuse, streaming/non-streaming, debug mode"""
    global cancelled, _clients
    
    prompt = data.get("prompt", "")
    team_config = data.get("team", {})
    model_config = data.get("model", {})
    context = data.get("context", {})
    session_id = data.get("session_id", "default")
    mcp_config = data.get("mcp_tools", [])
    
    # New options
    enable_streaming = data.get("stream", True)
    debug_mode = data.get("debug", False)
    reuse_session = data.get("reuse_session", True)
    git_url = context.get("git_url")
    branch = context.get("branch")
    
    if not team_config:
        emit_event("error", "No team config provided")
        return
    
    thinking_manager.clear()
    thinking_manager.update_progress(10)
    thinking_manager.add_step("Initializing team")
    
    # Download code if git_url provided
    project_path = None
    if git_url:
        project_path = download_code(git_url, branch)
        if project_path:
            context["project_path"] = project_path
    
    # Setup MCP tools
    thinking_manager.update_progress(20)
    mcp_tools = await setup_mcp_tools(mcp_config)
    
    try:
        # Check for cached team (session reuse)
        team = None
        if reuse_session and session_id in _clients:
            cached = _clients[session_id]
            if isinstance(cached, Team):
                team = cached
                thinking_manager.add_step("Reusing existing team session", {"session_id": session_id})
                emit_event("session_reused", f"Reusing team session: {session_id}")
        
        # Create new team if not cached
        if team is None:
            thinking_manager.update_progress(30)
            thinking_manager.add_step("Creating new team")
            
            team_name = team_config.get("name", "Team")
            mode = team_config.get("mode", "coordinate")
            leader_config = team_config.get("leader")
            members_config = team_config.get("members", [])
            description = team_config.get("description", "")
            
            # Create member agents
            members: List[Agent] = []
            for member_cfg in members_config:
                agent = create_agent_from_member(member_cfg, mcp_tools)
                members.append(agent)
                thinking_manager.add_step(f"Created member: {member_cfg.get('name', 'Agent')}", report=False)
            
            # Create leader agent if specified
            leader = None
            if leader_config:
                leader = create_agent_from_member(leader_config, mcp_tools)
                thinking_manager.add_step(f"Created leader: {leader_config.get('name', 'Leader')}", report=False)
            
            if not members and not leader:
                emit_event("error", "Team has no members or leader")
                return
            
            # Use leader's model or first member's model
            team_model = create_model(model_config)
            
            # Get mode-specific configuration
            mode_config = get_mode_config(mode)
            
            # Prepare all team members
            all_members = list(members)
            
            # Create team
            thinking_manager.update_progress(40)
            team = Team(
                name=team_name,
                members=all_members if all_members else [leader],
                model=team_model,
                description=description,
                instructions=[description] if description else [],
                markdown=True,
                **mode_config
            )
            
            # Cache for reuse
            if reuse_session:
                _clients[session_id] = team
        
        # Build prompt with context
        full_prompt = build_prompt_with_context(prompt, context)
        
        thinking_manager.update_progress(50)
        thinking_manager.add_step("Executing team", {"mode": team_config.get("mode", "coordinate"), "streaming": enable_streaming})
        
        if enable_streaming:
            # Streaming mode
            result_content = ""
            event_count = 0
            try:
                async for event in team.arun(
                    full_prompt,
                    stream=True,
                    stream_intermediate_steps=True,
                    session_id=session_id,
                    user_id=session_id,
                    show_members_responses=True,
                    markdown=True,
                    debug_mode=debug_mode,
                ):
                    if cancelled:
                        emit_event("cancelled", "Task was cancelled")
                        break
                    
                    event_count += 1
                    result_content = await handle_team_event(event, result_content)
                
                # Check if no events were received (possible API error)
                if event_count == 0:
                    emit_event("warning", "No events received from team execution. Possible API error.")
            except Exception as stream_error:
                emit_event("error", f"Team streaming error: {str(stream_error)}")
                import traceback
                emit_event("error", f"Traceback: {traceback.format_exc()}")
                raise
        else:
            # Non-streaming mode
            thinking_manager.add_step("Running in non-streaming mode")
            try:
                result = await team.arun(
                    full_prompt,
                    stream=False,
                    session_id=session_id,
                    user_id=session_id,
                    show_members_responses=True,
                    markdown=True,
                    debug_mode=debug_mode,
                )
                
                # Extract content from result
                if hasattr(result, "content") and result.content:
                    result_content = str(result.content)
                elif hasattr(result, "to_dict"):
                    result_content = json.dumps(result.to_dict(), ensure_ascii=False)
                else:
                    result_content = str(result)
                
                # Check if result is empty
                if not result_content or result_content.strip() == "":
                    emit_event("warning", "Team execution returned empty result. Possible API error.")
            except Exception as exec_error:
                emit_event("error", f"Team execution error: {str(exec_error)}")
                import traceback
                emit_event("error", f"Traceback: {traceback.format_exc()}")
                raise
        
        thinking_manager.update_progress(100)
        if not cancelled:
            thinking_manager.add_step("Team execution completed")
            emit_event("completed", result_content, {"thinking_steps": thinking_manager.get_steps()})
            
    finally:
        await cleanup_mcp_tools(mcp_tools)


async def main(data: Dict[str, Any]):
    """Main entry point - dispatch based on type"""
    try:
        exec_type = data.get("type", "bot")
        
        emit_event("started", f"Starting {exec_type} execution")
        
        if exec_type == "team":
            await run_team(data)
        else:
            await run_bot(data)
            
    except Exception as e:
        import traceback
        emit_event("error", str(e), {"traceback": traceback.format_exc()})


if __name__ == "__main__":
    try:
        input_data = sys.stdin.read()
        if not input_data:
            emit_event("error", "No input data received")
            sys.exit(1)
            
        data = json.loads(input_data)
        asyncio.run(main(data))
    except json.JSONDecodeError as e:
        emit_event("error", f"Invalid JSON input: {e}")
    except Exception as e:
        import traceback
        emit_event("error", f"Unexpected error: {str(e)}", {"traceback": traceback.format_exc()})
