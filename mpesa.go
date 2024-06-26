package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/valyala/fasthttp"
)

func main() {
	// Define environment variables
	os.Setenv("SECRET_KEY", "lk2J0CJ8nz44VYUj")
	os.Setenv("CONSUMER_KEY", "GlczBB2hH6RPr3J0R5SuzatG76bz4ulC")

	// Initialize Fiber app
	app := fiber.New()

	// Enable CORS
	app.Use(cors.New())

	// Define routes
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Safaricom integration with Brian Itira")
	})

	app.Post("/token", createToken)
	app.Post("/stkpush", stkPush)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	log.Fatal(app.Listen(":" + port))
}

func createToken(c *fiber.Ctx) error {
	secretKey := os.Getenv("SECRET_KEY")
	consumerKey := os.Getenv("CONSUMER_KEY")
	auth := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + secretKey))

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI("https://sandbox.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials")
	req.Header.SetMethod("GET")
	req.Header.Set("Authorization", "Basic "+auth)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		return err
	}

	token := string(resp.Body())

	c.Set("Content-Type", "application/json")
	return c.SendString(token)
}

func stkPush(c *fiber.Ctx) error {
	// Retrieve and encode credentials
	secretKey := os.Getenv("SECRET_KEY")
	consumerKey := os.Getenv("CONSUMER_KEY")
	auth := base64.StdEncoding.EncodeToString([]byte(consumerKey + ":" + secretKey))

	// Acquire token
	token, err := getAccessToken(auth)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to get access token",
			"details": err.Error(),
		})
	}

	// Define request parameters
	businessShortCode := "174379"
	passKey := "bfb279f9aa9bdbcf158e97dd71a467cd2e0c893059b10f78e6b72ada1ed2c919"
	timestamp := time.Now().Format("20060102150405")
	password := base64.StdEncoding.EncodeToString([]byte(businessShortCode + passKey + timestamp))
	amount := "1"
	phoneNumber := "254794491054"
	callbackURL := "https://mydomain.com/path"
	requestPayload := map[string]interface{}{
		"BusinessShortCode": businessShortCode,
		"Password":          password,
		"Timestamp":         timestamp,
		"TransactionType":   "CustomerPayBillOnline",
		"Amount":            amount,
		"PartyA":            phoneNumber,
		"PartyB":            businessShortCode,
		"PhoneNumber":       phoneNumber,
		"CallBackURL":       callbackURL,
		"AccountReference":  "Mpesa Test",
		"TransactionDesc":   "Testing stk push",
	}

	requestBody, err := json.Marshal(requestPayload)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to marshal request payload",
			"details": err.Error(),
		})
	}

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI("https://sandbox.safaricom.co.ke/mpesa/stkpush/v1/processrequest")
	req.Header.SetMethod("POST")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.SetContentType("application/json")
	req.SetBody(requestBody)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to initiate STK push",
			"details": err.Error(),
		})
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return c.Status(resp.StatusCode()).JSON(fiber.Map{
			"error":   "Failed to initiate STK push",
			"details": string(resp.Body()),
		})
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error":   "Failed to parse response",
			"details": err.Error(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(result)
}

func getAccessToken(auth string) (string, error) {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)
	req.SetRequestURI("https://sandbox.safaricom.co.ke/oauth/v1/generate?grant_type=client_credentials")
	req.Header.SetMethod("GET")
	req.Header.Set("Authorization", "Basic "+auth)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	client := &fasthttp.Client{}
	if err := client.Do(req, resp); err != nil {
		return "", err
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return "", fmt.Errorf("failed to get access token: %s", resp.Body())
	}

	var result map[string]interface{}
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return "", err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("failed to parse access token")
	}

	return token, nil
}
