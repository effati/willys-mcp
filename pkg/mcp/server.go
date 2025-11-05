package mcp

import (
	"fmt"
	"log"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/shayan/willys-mcp/internal/willys"
)

type Server struct {
	mcpServer   *server.MCPServer
	toolHandler *ToolHandler
	client      willys.WillysAPI
}

func NewServer(client willys.WillysAPI) *Server {
	toolHandler := NewToolHandler(client)

	s := &Server{
		toolHandler: toolHandler,
		client:      client,
	}

	mcpServer := server.NewMCPServer(
		"Willys Grocery Store",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s.registerTools(mcpServer)

	s.mcpServer = mcpServer

	return s
}

func (s *Server) registerTools(mcpServer *server.MCPServer) {
	searchGroceriesTool := mcp.NewTool("search_groceries",
		mcp.WithDescription("Search for products on Willys.se with optional filters and sorting"),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query for products (e.g., 'milk', 'bread', 'vegetables')"),
		),
		mcp.WithNumber("page",
			mcp.Description("Page number for pagination (default: 0)"),
		),
		mcp.WithNumber("size",
			mcp.Description("Number of results per page (default: 30)"),
		),
		mcp.WithObject("preferences",
			mcp.Description("Search preferences for filtering and sorting"),
			mcp.Properties(map[string]any{
				"price_sensitivity": map[string]any{
					"type":        "string",
					"description": "Price preference: 'cheapest', 'balanced', or 'quality'",
				},
				"max_price_per_unit": map[string]any{
					"type":        "number",
					"description": "Maximum price per unit (kr/kg or kr/l)",
				},
				"required_labels": map[string]any{
					"type":        "array",
					"description": "Required quality labels (e.g., ['KRAV', 'Ekologisk', 'Nyckelhål'])",
					"items": map[string]any{
						"type": "string",
					},
				},
				"preferred_labels": map[string]any{
					"type":        "array",
					"description": "Preferred quality labels for sorting",
					"items": map[string]any{
						"type": "string",
					},
				},
				"sort_by": map[string]any{
					"type":        "string",
					"description": "Sort method: 'cheapest', 'best_value', or 'highest_quality'",
				},
			}),
		),
	)
	mcpServer.AddTool(searchGroceriesTool, s.toolHandler.SearchGroceries)

	addToCartTool := mcp.NewTool("add_to_cart",
		mcp.WithDescription("Add items to cart"),
		mcp.WithString("product_code",
			mcp.Required(),
			mcp.Description("Product code in format {id}_{ST|KG} (e.g., '101233933_ST')"),
		),
		mcp.WithNumber("quantity",
			mcp.Required(),
			mcp.Description("Quantity to add"),
		),
	)
	mcpServer.AddTool(addToCartTool, s.toolHandler.AddToCart)

	viewCartTool := mcp.NewTool("view_cart",
		mcp.WithDescription("View current cart contents"),
	)
	mcpServer.AddTool(viewCartTool, s.toolHandler.ViewCart)

	removeFromCartTool := mcp.NewTool("remove_from_cart",
		mcp.WithDescription("Remove items from cart"),
		mcp.WithString("product_code",
			mcp.Required(),
			mcp.Description("Product code to remove"),
		),
		mcp.WithNumber("quantity",
			mcp.Description("Quantity to remove (default: removes all)"),
		),
	)
	mcpServer.AddTool(removeFromCartTool, s.toolHandler.RemoveFromCart)

	selectDeliveryTimeTool := mcp.NewTool("select_delivery_time",
		mcp.WithDescription("Select delivery address and time slot"),
		mcp.WithObject("address",
			mcp.Required(),
			mcp.Description("Delivery address information"),
			mcp.Properties(map[string]any{
				"first_name": map[string]any{
					"type":        "string",
					"description": "Recipient's first name",
					"required":    true,
				},
				"last_name": map[string]any{
					"type":        "string",
					"description": "Recipient's last name",
					"required":    true,
				},
				"address": map[string]any{
					"type":        "string",
					"description": "Street address (e.g., 'Drottninggatan 1')",
					"required":    true,
				},
				"postal_code": map[string]any{
					"type":        "string",
					"description": "Postal code (e.g., '11151')",
					"required":    true,
				},
				"city": map[string]any{
					"type":        "string",
					"description": "City name (e.g., 'Stockholm')",
					"required":    true,
				},
				"door_code": map[string]any{
					"type":        "string",
					"description": "Optional door code for building access",
				},
				"message_to_driver": map[string]any{
					"type":        "string",
					"description": "Optional message to delivery driver (e.g., instructions or directions)",
				},
			}),
		),
		mcp.WithString("delivery_date",
			mcp.Required(),
			mcp.Description("Delivery date in ISO 8601 format (YYYY-MM-DD)"),
		),
		mcp.WithString("time_slot",
			mcp.Required(),
			mcp.Description("Time slot in format 'HH:MM-HH:MM' (e.g., '15:00-17:00')"),
		),
	)
	mcpServer.AddTool(selectDeliveryTimeTool, s.toolHandler.SelectDeliveryTime)

	getAvailableTimeSlotsTool := mcp.NewTool("get_available_time_slots",
		mcp.WithDescription("Get available delivery time slots for a postal code"),
		mcp.WithString("postal_code",
			mcp.Required(),
			mcp.Description("Postal code to check availability for (e.g., '11151')"),
		),
	)
	mcpServer.AddTool(getAvailableTimeSlotsTool, s.toolHandler.GetAvailableTimeSlots)

	proceedToCheckoutTool := mcp.NewTool("proceed_to_checkout",
		mcp.WithDescription("Get checkout URL to complete payment"),
	)
	mcpServer.AddTool(proceedToCheckoutTool, s.toolHandler.ProceedToCheckout)
}

func (s *Server) Start() error {
	log.Println("Starting Willys MCP server...")

	if err := server.ServeStdio(s.mcpServer); err != nil {
		return fmt.Errorf("failed to start MCP server: %w", err)
	}

	return nil
}
