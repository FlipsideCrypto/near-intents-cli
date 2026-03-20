package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	DefaultFlipsideAPIEndpoint = "https://api.flipsidecrypto.xyz"
	DefaultFlipsideAgent       = "trading_agent"
)

var (
	flagFlipsideAPIKey string
	flagMessage        string
	flagAgent          string
)

type FlipsideAgentRequest struct {
	Message string `json:"message"`
}

type IntelResponse struct {
	Agent   string `json:"agent"`
	Message string `json:"message"`
}

var intelCmd = &cobra.Command{
	Use:   "intel",
	Short: "Get portfolio intelligence from Flipside",
	Long:  "Query a Flipside AI agent for portfolio analysis, rebalancing recommendations, and on-chain intelligence.",
	Run: func(cmd *cobra.Command, args []string) {
		cfg := loadConfig()
		apiKey := resolveFlipsideAPIKey(flagFlipsideAPIKey, cfg)
		if apiKey == "" {
			PrintErrorResponse("MISSING_API_KEY", "Flipside API key required. Set via --flipside-api-key, FLIPSIDE_API_KEY env var, or flipside_api_key in ~/.near-intents.json")
			return
		}
		if flagMessage == "" {
			PrintErrorResponse("MISSING_MESSAGE", "--message is required")
			return
		}

		agent := flagAgent
		endpoint := resolveFlipsideEndpoint(cfg)
		url := fmt.Sprintf("%s/public/v3/agents/%s/stream", endpoint, agent)

		verbose("Flipside agent: %s", agent)
		verbose("Flipside endpoint: %s", url)

		body := FlipsideAgentRequest{Message: flagMessage}
		data, err := json.Marshal(body)
		if err != nil {
			PrintErrorResponse("MARSHAL_ERROR", fmt.Sprintf("failed to marshal request: %v", err))
			return
		}

		if flagVerbose {
			fmt.Fprintf(os.Stderr, "[verbose] request body: %s\n", string(data))
		}

		req, err := http.NewRequest("POST", url, bytes.NewReader(data))
		if err != nil {
			PrintErrorResponse("REQUEST_ERROR", fmt.Sprintf("failed to create request: %v", err))
			return
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-key", apiKey)

		client := &http.Client{Timeout: 5 * time.Minute}
		resp, err := client.Do(req)
		if err != nil {
			PrintErrorResponse("HTTP_ERROR", fmt.Sprintf("request failed: %v", err))
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			respBody, _ := io.ReadAll(resp.Body)
			var apiErr struct {
				Error string `json:"error"`
			}
			if json.Unmarshal(respBody, &apiErr) == nil && apiErr.Error != "" {
				PrintErrorResponse("FLIPSIDE_API_ERROR", apiErr.Error)
			} else {
				PrintErrorResponse("FLIPSIDE_API_ERROR", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)))
			}
			return
		}

		// Stream the response, accumulating the full text
		var result strings.Builder
		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			result.WriteString(scanner.Text())
			result.WriteString("\n")
		}
		if err := scanner.Err(); err != nil {
			PrintErrorResponse("STREAM_ERROR", fmt.Sprintf("error reading stream: %v", err))
			return
		}

		PrintSuccess(IntelResponse{
			Agent:   agent,
			Message: strings.TrimSpace(result.String()),
		})
	},
}

func resolveFlipsideAPIKey(flagValue string, cfg *Config) string {
	if flagValue != "" {
		return flagValue
	}
	if env := os.Getenv("FLIPSIDE_API_KEY"); env != "" {
		return env
	}
	return cfg.FlipsideAPIKey
}

func resolveFlipsideEndpoint(cfg *Config) string {
	if cfg.FlipsideAPIEndpoint != "" {
		return cfg.FlipsideAPIEndpoint
	}
	return DefaultFlipsideAPIEndpoint
}

func init() {
	intelCmd.Flags().StringVar(&flagFlipsideAPIKey, "flipside-api-key", "", "Flipside API key (or set FLIPSIDE_API_KEY)")
	intelCmd.Flags().StringVar(&flagMessage, "message", "", "Message to send to the Flipside agent")
	intelCmd.Flags().StringVar(&flagAgent, "agent", DefaultFlipsideAgent, "Flipside agent to query")
	rootCmd.AddCommand(intelCmd)
}
