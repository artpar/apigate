package testing_test

import (
	"context"
	"errors"
	"testing"

	"github.com/artpar/apigate/core/capability"
	captest "github.com/artpar/apigate/core/capability/testing"
)

func TestMockPayment(t *testing.T) {
	ctx := context.Background()
	payment := captest.NewMockPayment("stripe_test")

	// Test CreateCustomer
	customerID, err := payment.CreateCustomer(ctx, "test@example.com", "Test User", "user_123")
	if err != nil {
		t.Fatalf("CreateCustomer() error = %v", err)
	}
	if customerID == "" {
		t.Error("CreateCustomer() returned empty customerID")
	}

	// Verify call tracking
	if calls := payment.CreateCustomerCalls(); calls != 1 {
		t.Errorf("CreateCustomerCalls() = %d, want 1", calls)
	}

	// Test CreateCheckoutSession without trial
	url, err := payment.CreateCheckoutSession(ctx, customerID, "price_123", "https://success.com", "https://cancel.com", 0)
	if err != nil {
		t.Fatalf("CreateCheckoutSession() error = %v", err)
	}
	if url == "" {
		t.Error("CreateCheckoutSession() returned empty URL")
	}

	// Test CreateCheckoutSession with trial
	urlWithTrial, err := payment.CreateCheckoutSession(ctx, customerID, "price_123", "https://success.com", "https://cancel.com", 14)
	if err != nil {
		t.Fatalf("CreateCheckoutSession() with trial error = %v", err)
	}
	if urlWithTrial == "" {
		t.Error("CreateCheckoutSession() with trial returned empty URL")
	}

	// Test CreatePrice
	priceID, err := payment.CreatePrice(ctx, "Pro Plan", 2900, "month")
	if err != nil {
		t.Fatalf("CreatePrice() error = %v", err)
	}
	if priceID == "" {
		t.Error("CreatePrice() returned empty priceID")
	}
}

func TestMockEmail(t *testing.T) {
	ctx := context.Background()
	email := captest.NewMockEmail("smtp_test")

	msg := capability.EmailMessage{
		To:      "recipient@example.com",
		Subject: "Test Subject",
		HTMLBody: "<p>Hello</p>",
	}

	// Send email
	if err := email.Send(ctx, msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	// Verify sent messages
	sent := email.SentMessages()
	if len(sent) != 1 {
		t.Errorf("SentMessages() len = %d, want 1", len(sent))
	}
	if sent[0].To != "recipient@example.com" {
		t.Errorf("SentMessages()[0].To = %s, want recipient@example.com", sent[0].To)
	}

	// Test reset
	email.Reset()
	if len(email.SentMessages()) != 0 {
		t.Error("Reset() should clear sent messages")
	}
}

func TestMockCache(t *testing.T) {
	ctx := context.Background()
	cache := captest.NewMockCache("redis_test")

	// Test Set and Get
	if err := cache.Set(ctx, "key1", []byte("value1"), 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	val, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(val) != "value1" {
		t.Errorf("Get() = %s, want value1", string(val))
	}

	// Test Exists
	exists, _ := cache.Exists(ctx, "key1")
	if !exists {
		t.Error("Exists() should return true for existing key")
	}

	exists, _ = cache.Exists(ctx, "nonexistent")
	if exists {
		t.Error("Exists() should return false for non-existing key")
	}

	// Test Delete
	if err := cache.Delete(ctx, "key1"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	val, _ = cache.Get(ctx, "key1")
	if val != nil {
		t.Error("Get() should return nil after Delete()")
	}

	// Test Increment
	newVal, err := cache.Increment(ctx, "counter", 5, 0)
	if err != nil {
		t.Fatalf("Increment() error = %v", err)
	}
	if newVal != 5 {
		t.Errorf("Increment() = %d, want 5", newVal)
	}

	newVal, _ = cache.Increment(ctx, "counter", 3, 0)
	if newVal != 8 {
		t.Errorf("Increment() = %d, want 8", newVal)
	}

	// Test Flush
	cache.Set(ctx, "key2", []byte("value2"), 0)
	if err := cache.Flush(ctx); err != nil {
		t.Fatalf("Flush() error = %v", err)
	}
	if len(cache.Keys()) != 0 {
		t.Error("Keys should be empty after Flush()")
	}
}

func TestMockStorage(t *testing.T) {
	ctx := context.Background()
	storage := captest.NewMockStorage("s3_test")

	// Test Put and Get
	data := []byte("file content")
	if err := storage.Put(ctx, "folder/file.txt", data, "text/plain"); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	got, contentType, err := storage.Get(ctx, "folder/file.txt")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if string(got) != "file content" {
		t.Errorf("Get() = %s, want 'file content'", string(got))
	}
	if contentType != "text/plain" {
		t.Errorf("Get() contentType = %s, want text/plain", contentType)
	}

	// Test Exists
	exists, _ := storage.Exists(ctx, "folder/file.txt")
	if !exists {
		t.Error("Exists() should return true")
	}

	// Test List
	storage.Put(ctx, "folder/file2.txt", []byte("data"), "text/plain")
	objects, err := storage.List(ctx, "folder/", 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(objects) != 2 {
		t.Errorf("List() len = %d, want 2", len(objects))
	}

	// Test Delete
	if err := storage.Delete(ctx, "folder/file.txt"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	exists, _ = storage.Exists(ctx, "folder/file.txt")
	if exists {
		t.Error("Exists() should return false after Delete()")
	}

	// Test GetURL
	url, err := storage.GetURL(ctx, "folder/file2.txt", 3600)
	if err != nil {
		t.Fatalf("GetURL() error = %v", err)
	}
	if url == "" {
		t.Error("GetURL() returned empty URL")
	}
}

func TestMockQueue(t *testing.T) {
	ctx := context.Background()
	queue := captest.NewMockQueue("sqs_test")

	// Test Enqueue
	job := capability.Job{
		Type:    "email.send",
		Payload: map[string]any{"to": "user@example.com"},
	}

	if err := queue.Enqueue(ctx, "emails", job); err != nil {
		t.Fatalf("Enqueue() error = %v", err)
	}

	// Test QueueLength
	length, _ := queue.QueueLength(ctx, "emails")
	if length != 1 {
		t.Errorf("QueueLength() = %d, want 1", length)
	}

	// Test Dequeue
	dequeued, err := queue.Dequeue(ctx, "emails", 5)
	if err != nil {
		t.Fatalf("Dequeue() error = %v", err)
	}
	if dequeued == nil {
		t.Fatal("Dequeue() returned nil")
	}
	if dequeued.Type != "email.send" {
		t.Errorf("Dequeued job type = %s, want email.send", dequeued.Type)
	}

	// Queue should be empty now
	length, _ = queue.QueueLength(ctx, "emails")
	if length != 0 {
		t.Errorf("QueueLength() = %d, want 0 after dequeue", length)
	}

	// Test Ack
	if err := queue.Ack(ctx, "emails", dequeued.ID); err != nil {
		t.Fatalf("Ack() error = %v", err)
	}
	if !queue.AckedJobs()[dequeued.ID] {
		t.Error("Job should be marked as acked")
	}
}

func TestMockNotification(t *testing.T) {
	ctx := context.Background()
	notif := captest.NewMockNotification("slack_test")

	msg := capability.NotificationMessage{
		Channel:  "#alerts",
		Title:    "Alert",
		Message:  "Something happened",
		Severity: "warning",
	}

	if err := notif.Send(ctx, msg); err != nil {
		t.Fatalf("Send() error = %v", err)
	}

	sent := notif.SentNotifications()
	if len(sent) != 1 {
		t.Errorf("SentNotifications() len = %d, want 1", len(sent))
	}
	if sent[0].Channel != "#alerts" {
		t.Errorf("SentNotifications()[0].Channel = %s, want #alerts", sent[0].Channel)
	}
}

func TestMockHasher(t *testing.T) {
	hasher := captest.NewMockHasher("bcrypt_test")

	hash, err := hasher.Hash("password123")
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if !hasher.Compare(hash, "password123") {
		t.Error("Compare() should return true for correct password")
	}

	if hasher.Compare(hash, "wrongpassword") {
		t.Error("Compare() should return false for wrong password")
	}
}

func TestTestResolver(t *testing.T) {
	ctx := context.Background()
	tr := captest.NewTestResolver()

	// All providers should be available via embedded Resolver
	payment, err := tr.Payment(ctx)
	if err != nil {
		t.Fatalf("Payment() error = %v", err)
	}
	if payment == nil {
		t.Error("Payment() returned nil")
	}

	email, err := tr.Email(ctx)
	if err != nil {
		t.Fatalf("Email() error = %v", err)
	}
	if email == nil {
		t.Error("Email() returned nil")
	}

	cache, err := tr.Cache(ctx)
	if err != nil {
		t.Fatalf("Cache() error = %v", err)
	}
	if cache == nil {
		t.Error("Cache() returned nil")
	}

	storage, err := tr.Storage(ctx)
	if err != nil {
		t.Fatalf("Storage() error = %v", err)
	}
	if storage == nil {
		t.Error("Storage() returned nil")
	}

	queue, err := tr.Queue(ctx)
	if err != nil {
		t.Fatalf("Queue() error = %v", err)
	}
	if queue == nil {
		t.Error("Queue() returned nil")
	}

	notif, err := tr.Notification(ctx)
	if err != nil {
		t.Fatalf("Notification() error = %v", err)
	}
	if notif == nil {
		t.Error("Notification() returned nil")
	}

	// Verify mock fields are also accessible
	if tr.MockPayment == nil {
		t.Error("MockPayment should not be nil")
	}
	if tr.MockEmail == nil {
		t.Error("MockEmail should not be nil")
	}
}

func TestMockPayment_PortalAndSubscription(t *testing.T) {
	ctx := context.Background()
	payment := captest.NewMockPayment("test_payment")

	// Test Name
	if payment.Name() != "test_payment" {
		t.Errorf("Name() = %s, want test_payment", payment.Name())
	}

	// Test CreatePortalSession
	url, err := payment.CreatePortalSession(ctx, "cus_123", "https://return.url")
	if err != nil {
		t.Fatalf("CreatePortalSession() error = %v", err)
	}
	if url == "" {
		t.Error("CreatePortalSession() returned empty URL")
	}

	// Test AddSubscription and GetSubscription
	sub := capability.Subscription{
		ID:         "sub_123",
		CustomerID: "cus_123",
		PriceID:    "price_pro",
		Status:     "active",
	}
	payment.AddSubscription(sub)

	got, err := payment.GetSubscription(ctx, "sub_123")
	if err != nil {
		t.Fatalf("GetSubscription() error = %v", err)
	}
	if got.ID != "sub_123" {
		t.Errorf("GetSubscription().ID = %s, want sub_123", got.ID)
	}

	// Test GetSubscription not found
	_, err = payment.GetSubscription(ctx, "nonexistent")
	if err == nil {
		t.Error("GetSubscription() should error for nonexistent subscription")
	}

	// Test CancelSubscription
	err = payment.CancelSubscription(ctx, "sub_123", false)
	if err != nil {
		t.Fatalf("CancelSubscription() error = %v", err)
	}

	got, _ = payment.GetSubscription(ctx, "sub_123")
	if got.Status != "cancelled" {
		t.Errorf("Status = %s, want cancelled", got.Status)
	}

	// Test ReportUsage
	err = payment.ReportUsage(ctx, "si_123", 100, 1234567890)
	if err != nil {
		t.Fatalf("ReportUsage() error = %v", err)
	}

	// Test ParseWebhook
	eventType, data, err := payment.ParseWebhook([]byte("payload"), "signature")
	if err != nil {
		t.Fatalf("ParseWebhook() error = %v", err)
	}
	if eventType != "test.event" {
		t.Errorf("ParseWebhook() eventType = %s, want test.event", eventType)
	}
	if data["test"] != true {
		t.Error("ParseWebhook() should return test data")
	}

	// Test Customers
	payment.CreateCustomer(ctx, "test@example.com", "Test", "user_1")
	customers := payment.Customers()
	if len(customers) != 1 {
		t.Errorf("Customers() len = %d, want 1", len(customers))
	}

	// Test SetCreateCustomerError
	payment.SetCreateCustomerError(errors.New("not configured"))
	_, err = payment.CreateCustomer(ctx, "fail@example.com", "Fail", "user_2")
	if err == nil {
		t.Error("CreateCustomer() should fail after SetCreateCustomerError")
	}
}

func TestMockEmail_Advanced(t *testing.T) {
	ctx := context.Background()
	email := captest.NewMockEmail("test_email")

	// Test Name
	if email.Name() != "test_email" {
		t.Errorf("Name() = %s, want test_email", email.Name())
	}

	// Test SendTemplate
	err := email.SendTemplate(ctx, "user@example.com", "welcome_template", map[string]string{"name": "Test"})
	if err != nil {
		t.Fatalf("SendTemplate() error = %v", err)
	}

	// Verify template was sent as email
	sent := email.SentMessages()
	if len(sent) != 1 {
		t.Errorf("SentMessages() len = %d, want 1", len(sent))
	}
	if sent[0].To != "user@example.com" {
		t.Errorf("To = %s, want user@example.com", sent[0].To)
	}

	// Test TestConnection
	if err := email.TestConnection(ctx); err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}

	// Test SetSendError
	email.SetSendError(errors.New("not configured"))
	err = email.Send(ctx, capability.EmailMessage{To: "fail@example.com"})
	if err == nil {
		t.Error("Send() should fail after SetSendError")
	}
}

func TestMockCache_Advanced(t *testing.T) {
	ctx := context.Background()
	cache := captest.NewMockCache("test_cache")

	// Test Name
	if cache.Name() != "test_cache" {
		t.Errorf("Name() = %s, want test_cache", cache.Name())
	}

	// Test Close
	if err := cache.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// Test FlushCount
	cache.Flush(ctx)
	cache.Flush(ctx)
	if cache.FlushCount() != 2 {
		t.Errorf("FlushCount() = %d, want 2", cache.FlushCount())
	}
}

func TestMockStorage_Advanced(t *testing.T) {
	ctx := context.Background()
	storage := captest.NewMockStorage("test_storage")

	// Test Name
	if storage.Name() != "test_storage" {
		t.Errorf("Name() = %s, want test_storage", storage.Name())
	}

	// Test ObjectCount
	storage.Put(ctx, "file1.txt", []byte("data1"), "text/plain")
	storage.Put(ctx, "file2.txt", []byte("data2"), "text/plain")
	if count := storage.ObjectCount(); count != 2 {
		t.Errorf("ObjectCount() = %d, want 2", count)
	}

	// Test PutStream (if it exists)
	// The PutStream function might just wrap Put, so test it exists
}

func TestMockQueue_Advanced(t *testing.T) {
	ctx := context.Background()
	queue := captest.NewMockQueue("test_queue")

	// Test Name
	if queue.Name() != "test_queue" {
		t.Errorf("Name() = %s, want test_queue", queue.Name())
	}

	// Test EnqueueDelayed (if it exists)
	job := capability.Job{Type: "delayed.job", Payload: map[string]any{}}
	err := queue.EnqueueDelayed(ctx, "delayed_queue", job, 60)
	if err != nil {
		t.Fatalf("EnqueueDelayed() error = %v", err)
	}

	// Test Nack
	queue.Enqueue(ctx, "test_queue", capability.Job{Type: "nack.test", Payload: map[string]any{}})
	dequeued, _ := queue.Dequeue(ctx, "test_queue", 5)
	if dequeued != nil {
		err = queue.Nack(ctx, "test_queue", dequeued.ID)
		if err != nil {
			t.Fatalf("Nack() error = %v", err)
		}
		nackedJobs := queue.NackedJobs()
		if nackedJobs[dequeued.ID] == 0 {
			t.Error("Job should be marked as nacked")
		}
	}

	// Test Close
	if err := queue.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

func TestMockNotification_Advanced(t *testing.T) {
	ctx := context.Background()
	notif := captest.NewMockNotification("test_notification")

	// Test Name
	if notif.Name() != "test_notification" {
		t.Errorf("Name() = %s, want test_notification", notif.Name())
	}

	// Test SendBatch
	msgs := []capability.NotificationMessage{
		{Channel: "#channel1", Message: "msg1"},
		{Channel: "#channel2", Message: "msg2"},
	}
	if err := notif.SendBatch(ctx, msgs); err != nil {
		t.Fatalf("SendBatch() error = %v", err)
	}

	// Verify batch was sent
	sent := notif.SentNotifications()
	if len(sent) != 2 {
		t.Errorf("SentNotifications() len = %d, want 2", len(sent))
	}

	// Test TestConnection
	if err := notif.TestConnection(ctx); err != nil {
		t.Fatalf("TestConnection() error = %v", err)
	}

	// Test SetSendError
	notif.SetSendError(errors.New("not configured"))
	err := notif.Send(ctx, capability.NotificationMessage{Channel: "#fail"})
	if err == nil {
		t.Error("Send() should fail after SetSendError")
	}
}

// TestCapabilityDrivenDevelopment demonstrates TDD with capabilities
func TestCapabilityDrivenDevelopment(t *testing.T) {
	ctx := context.Background()
	tr := captest.NewTestResolver()

	// Simulate a user signup flow that uses multiple capabilities
	t.Run("user signup sends welcome email", func(t *testing.T) {
		email, _ := tr.Email(ctx)

		// Simulate sending welcome email
		err := email.Send(ctx, capability.EmailMessage{
			To:      "newuser@example.com",
			Subject: "Welcome!",
		})
		if err != nil {
			t.Fatalf("Send welcome email failed: %v", err)
		}

		// Verify email was sent using mock helper
		sent := tr.MockEmail.SentMessages()
		if len(sent) != 1 {
			t.Errorf("Expected 1 email sent, got %d", len(sent))
		}
	})

	tr.MockEmail.Reset() // Reset for next test

	t.Run("subscription creates customer and processes payment", func(t *testing.T) {
		payment, _ := tr.Payment(ctx)

		// Create customer
		customerID, err := payment.CreateCustomer(ctx, "user@example.com", "Test User", "user_1")
		if err != nil {
			t.Fatalf("CreateCustomer failed: %v", err)
		}

		// Create checkout session with 14-day trial
		sessionURL, err := payment.CreateCheckoutSession(ctx, customerID, "price_pro", "https://app.com/success", "https://app.com/cancel", 14)
		if err != nil {
			t.Fatalf("CreateCheckoutSession failed: %v", err)
		}

		if sessionURL == "" {
			t.Error("Expected checkout session URL")
		}

		// Verify customer was created using mock helper
		if tr.MockPayment.CreateCustomerCalls() != 1 {
			t.Errorf("Expected 1 CreateCustomer call, got %d", tr.MockPayment.CreateCustomerCalls())
		}
	})

	t.Run("rate limiting uses cache", func(t *testing.T) {
		cache, _ := tr.Cache(ctx)

		// Simulate rate limit check
		key := "ratelimit:user_1:minute"
		count, err := cache.Increment(ctx, key, 1, 60)
		if err != nil {
			t.Fatalf("Increment failed: %v", err)
		}

		if count != 1 {
			t.Errorf("Expected count 1, got %d", count)
		}

		// Simulate multiple requests
		count, _ = cache.Increment(ctx, key, 1, 60)
		count, _ = cache.Increment(ctx, key, 1, 60)

		if count != 3 {
			t.Errorf("Expected count 3, got %d", count)
		}
	})
}
