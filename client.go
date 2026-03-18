package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
	cfg        *Config
}

func newClient(cfg *Config) *Client {
	token := resolveToken(flagToken, cfg)
	return &Client{
		baseURL:    cfg.APIEndpoint,
		token:      token,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		cfg:        cfg,
	}
}

func (c *Client) doRequest(method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(data)
		if flagVerbose {
			fmt.Fprintf(os.Stderr, "[verbose] request body: %s\n", string(data))
		}
	}

	req, err := http.NewRequest(method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}

	verbose("HTTP %s %s", method, c.baseURL+path)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "[verbose] response status: %d\n", resp.StatusCode)
		fmt.Fprintf(os.Stderr, "[verbose] response body: %s\n", string(respBody))
	}

	if resp.StatusCode >= 400 {
		var badReq struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(respBody, &badReq) == nil && badReq.Message != "" {
			return nil, fmt.Errorf("%s", badReq.Message)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// API types

type TokenResponse struct {
	AssetId         string  `json:"assetId"`
	Decimals        float32 `json:"decimals"`
	Blockchain      string  `json:"blockchain"`
	Symbol          string  `json:"symbol"`
	Price           float32 `json:"price"`
	PriceUpdatedAt  string  `json:"priceUpdatedAt"`
	ContractAddress *string `json:"contractAddress,omitempty"`
}

type QuoteRequest struct {
	Dry               bool      `json:"dry"`
	SwapType          string    `json:"swapType"`
	SlippageTolerance float32   `json:"slippageTolerance"`
	OriginAsset       string    `json:"originAsset"`
	DestinationAsset  string    `json:"destinationAsset"`
	Amount            string    `json:"amount"`
	Recipient         string    `json:"recipient"`
	RecipientType     string    `json:"recipientType"`
	RefundTo          string    `json:"refundTo"`
	RefundType        string    `json:"refundType"`
	DepositType       string    `json:"depositType"`
	Deadline          time.Time `json:"deadline"`
	AppFees           []AppFee  `json:"appFees,omitempty"`
	Referral          *string   `json:"referral,omitempty"`
}

type AppFee struct {
	Recipient string  `json:"recipient"`
	Fee       float32 `json:"fee"`
}

type QuoteResponse struct {
	CorrelationId string       `json:"correlationId"`
	Timestamp     string       `json:"timestamp"`
	Signature     string       `json:"signature"`
	QuoteRequest  QuoteRequest `json:"quoteRequest"`
	Quote         Quote        `json:"quote"`
}

type Quote struct {
	DepositAddress     *string `json:"depositAddress,omitempty"`
	DepositMemo        *string `json:"depositMemo,omitempty"`
	AmountIn           string  `json:"amountIn"`
	AmountInFormatted  string  `json:"amountInFormatted"`
	AmountInUsd        string  `json:"amountInUsd"`
	MinAmountIn        string  `json:"minAmountIn"`
	AmountOut          string  `json:"amountOut"`
	AmountOutFormatted string  `json:"amountOutFormatted"`
	AmountOutUsd       string  `json:"amountOutUsd"`
	MinAmountOut       string  `json:"minAmountOut"`
	Deadline           *string `json:"deadline,omitempty"`
	TimeWhenInactive   *string `json:"timeWhenInactive,omitempty"`
	TimeEstimate       float32 `json:"timeEstimate"`
}

type SubmitDepositTxRequest struct {
	TxHash            string  `json:"txHash"`
	DepositAddress    string  `json:"depositAddress"`
	NearSenderAccount *string `json:"nearSenderAccount,omitempty"`
	Memo              *string `json:"memo,omitempty"`
}

type SubmitDepositTxResponse struct {
	CorrelationId string      `json:"correlationId"`
	Status        string      `json:"status"`
	SwapDetails   SwapDetails `json:"swapDetails"`
}

type GetExecutionStatusResponse struct {
	CorrelationId string      `json:"correlationId"`
	Status        string      `json:"status"`
	UpdatedAt     string      `json:"updatedAt"`
	SwapDetails   SwapDetails `json:"swapDetails"`
}

type SwapDetails struct {
	AmountIn                 *string              `json:"amountIn,omitempty"`
	AmountInFormatted        *string              `json:"amountInFormatted,omitempty"`
	AmountInUsd              *string              `json:"amountInUsd,omitempty"`
	AmountOut                *string              `json:"amountOut,omitempty"`
	AmountOutFormatted       *string              `json:"amountOutFormatted,omitempty"`
	AmountOutUsd             *string              `json:"amountOutUsd,omitempty"`
	OriginChainTxHashes      []TransactionDetails `json:"originChainTxHashes"`
	DestinationChainTxHashes []TransactionDetails `json:"destinationChainTxHashes"`
	RefundedAmount           *string              `json:"refundedAmount,omitempty"`
	RefundReason             *string              `json:"refundReason,omitempty"`
}

type TransactionDetails struct {
	Hash        string `json:"hash"`
	ExplorerUrl string `json:"explorerUrl"`
}

// API methods

func (c *Client) GetTokens() ([]TokenResponse, error) {
	data, err := c.doRequest("GET", "/v0/tokens", nil)
	if err != nil {
		return nil, err
	}
	var tokens []TokenResponse
	if err := json.Unmarshal(data, &tokens); err != nil {
		return nil, fmt.Errorf("parse tokens: %w", err)
	}
	return tokens, nil
}

func (c *Client) PostQuote(req *QuoteRequest) (*QuoteResponse, error) {
	data, err := c.doRequest("POST", "/v0/quote", req)
	if err != nil {
		return nil, err
	}
	var resp QuoteResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse quote: %w", err)
	}
	return &resp, nil
}

func (c *Client) SubmitDepositTx(req *SubmitDepositTxRequest) (*SubmitDepositTxResponse, error) {
	data, err := c.doRequest("POST", "/v0/deposit/submit", req)
	if err != nil {
		return nil, err
	}
	var resp SubmitDepositTxResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse submit response: %w", err)
	}
	return &resp, nil
}

func (c *Client) GetExecutionStatus(depositAddress, depositMemo string) (*GetExecutionStatusResponse, error) {
	params := url.Values{}
	params.Set("depositAddress", depositAddress)
	if depositMemo != "" {
		params.Set("depositMemo", depositMemo)
	}
	data, err := c.doRequest("GET", "/v0/status?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var resp GetExecutionStatusResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("parse status: %w", err)
	}
	return &resp, nil
}

func verbose(format string, args ...any) {
	if flagVerbose {
		fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", args...)
	}
}
