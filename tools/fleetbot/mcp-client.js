const { Client } = require("@modelcontextprotocol/sdk/client/index.js");
const { SSEClientTransport } = require("@modelcontextprotocol/sdk/client/sse.js");

class McpClient {
  constructor({ url, authToken }) {
    this.url = url;
    this.authToken = authToken;
    this.client = null;
    this.tools = [];
    this._localTools = new Map(); // name → { definition, handler }
    this._connected = false;
  }

  /**
   * Register a local tool handled by fleetbot (not forwarded to MCP server).
   * @param {object} definition - Anthropic-format tool definition { name, description, input_schema }
   * @param {function} handler  - async (args) => string
   */
  addLocalTool(definition, handler) {
    this._localTools.set(definition.name, { definition, handler });
  }

  /**
   * Connect to the MCP server and discover available tools.
   * Safe to call multiple times — reconnects if the connection dropped.
   */
  async connect() {
    if (this._connected) return;

    console.log(`[mcp] Connecting to ${this.url}...`);
    const transport = new SSEClientTransport(new URL(this.url), {
      requestInit: { headers: { Authorization: `Bearer ${this.authToken}` } },
    });

    this.client = new Client({ name: "fleetbot", version: "1.0.0" });
    await this.client.connect(transport);

    // Mark disconnected when the transport closes so we can auto-reconnect
    this.client.onclose = () => {
      console.log("[mcp] Connection closed, will reconnect on next tool call");
      this._connected = false;
    };

    const { tools } = await this.client.listTools();
    this.tools = tools;
    this._connected = true;
    console.log(`[mcp] Connected. ${tools.length} tools available: ${tools.map((t) => t.name).join(", ")}`);
  }

  /**
   * Return tool definitions formatted for the Anthropic API tools parameter.
   */
  getAnthropicTools() {
    const remoteTools = this.tools.map((tool) => ({
      name: tool.name,
      description: tool.description || "",
      input_schema: tool.inputSchema,
    }));
    const localTools = [...this._localTools.values()].map((t) => t.definition);
    return [...remoteTools, ...localTools];
  }

  /**
   * Call a tool on the MCP server and return the result as a string.
   */
  async callTool(name, args) {
    // Check local tools first
    const local = this._localTools.get(name);
    if (local) {
      console.log(`[mcp] Calling local tool: ${name}(${JSON.stringify(args).slice(0, 200)})`);
      const text = await local.handler(args);
      console.log(`[mcp] Local tool ${name} returned ${text.length} chars`);
      return text;
    }

    if (!this._connected) {
      console.log("[mcp] Not connected, attempting to reconnect...");
      try {
        await this.connect();
      } catch (err) {
        throw new Error(`[mcp] Not connected and reconnect failed: ${err.message}`);
      }
    }
    console.log(`[mcp] Calling tool: ${name}(${JSON.stringify(args).slice(0, 200)})`);
    let result;
    try {
      result = await this.client.callTool({ name, arguments: args });
    } catch (err) {
      // Connection may have dropped mid-call — try one reconnect
      console.warn(`[mcp] Tool call failed (${err.message}), reconnecting and retrying...`);
      this._connected = false;
      await this.connect();
      result = await this.client.callTool({ name, arguments: args });
    }

    // MCP returns content as an array of content blocks
    const textParts = (result.content || [])
      .filter((c) => c.type === "text")
      .map((c) => c.text);

    const text = textParts.join("\n");
    console.log(`[mcp] Tool ${name} returned ${text.length} chars`);
    return text;
  }

  /**
   * Disconnect from the MCP server.
   */
  async disconnect() {
    if (this.client) {
      try {
        await this.client.close();
      } catch {
        // ignore close errors
      }
      this._connected = false;
      console.log("[mcp] Disconnected");
    }
  }
}

module.exports = McpClient;
