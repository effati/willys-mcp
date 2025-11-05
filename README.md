# Willys MCP

MCP server for Willys.se that lets you search products, manage your cart, and set up home delivery through an AI assistant.

Available tools: `search_groceries`, `add_to_cart`, `view_cart`, `remove_from_cart`, `get_available_time_slots`, `select_delivery_time`, and `proceed_to_checkout`. Pretty self-explanatory from the names.

## Setup

Build with `make build`.

The server needs your Willys credentials to authenticate. It uses headless browser automation to log in (takes about 10 seconds on startup) since Willys doesn't expose a straightforward API for this. Once authenticated, it extracts the session cookies and uses those for API requests.

Add it to your MCP client config (Claude Desktop, etc.):

```json
{
  "mcpServers": {
    "willys": {
      "command": "/path/to/willys-mcp",
      "env": {
        "WILLYS_USERNAME": "your-personnummer-or-willys-plus-number",
        "WILLYS_PASSWORD": "your-password"
      }
    }
  }
}
```

You'll need:
- `WILLYS_USERNAME`: Your Swedish personnummer (YYYYMMDDXXXX) or Willys Plus number
- `WILLYS_PASSWORD`: Your account password

On startup, it launches a headless browser, handles cookie consent, logs in, grabs the session cookies, and starts serving MCP requests.
