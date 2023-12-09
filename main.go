package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"sync/atomic"

	"cloud.google.com/go/pubsub"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

// Define a struct to represent a topic and its subscribers
type TopicSubscriber struct {
	Topic        string
	Subscription string
}

type Event struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	Msg   string `json:"msg"`
}

func sendMail(event, recevier, msg string) error {
	from := mail.NewEmail("Vault Support", "rammankar003@gmail.com")
	subject := fmt.Sprintf("%s: %s", "Vault Customer", event)
	to := mail.NewEmail("Vault User", recevier)
	plainTextContent := msg
	htmlContent := fmt.Sprintf("<strong>%s</strong>", msg)
	message := mail.NewSingleEmail(from, subject, to, plainTextContent, htmlContent)
	client := sendgrid.NewSendClient(os.Getenv("SENDGRID_API_KEY"))
	response, err := client.Send(message)
	if err != nil {
		log.Println("failed to send the email", err)
		return err
	} else {
		fmt.Println("response")
		fmt.Println(response.StatusCode)
		fmt.Println(response.Body)
		fmt.Println(response.Headers)
	}

	log.Println("Email sent successfully!")
	return nil
}

func pullMsgs(w io.Writer, projectID, subID string) error {
	ctx := context.Background()
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		return fmt.Errorf("pubsub.NewClient: %w", err)
	}
	defer client.Close()

	sub := client.Subscription(subID)

	var received int32
	for {
		// Receive a single message at a time without a timeout
		err := sub.Receive(ctx, func(_ context.Context, msg *pubsub.Message) {
			fmt.Fprintf(w, "Got message from %s: %q\n", subID, string(msg.Data))

			e := Event{}
			err = json.Unmarshal(msg.Data, &e)
			if err != nil {
				fmt.Printf("filed to read the event details %v \n", err)
			} else {
				err = sendMail(e.Name, e.Email, e.Msg)
				if err != nil {
					fmt.Printf("failed to send the mail %v", err)
				}
			}

			atomic.AddInt32(&received, 1)
			msg.Ack()
		})
		if err != context.Canceled {
			// If the context was canceled (e.g., program exit), break the loop
			break
		} else if err != nil {
			return fmt.Errorf("sub.Receive: %w", err)
		}
	}
	fmt.Fprintf(w, "Received %d messages from %s\n", received, subID)

	return nil
}

func main() {
	projectID := "valut-svc" // Replace with your Google Cloud project ID

	currentDir, err := os.Getwd()
	if err != nil {
		fmt.Println("Error getting current working directory:", err)
		return
	}

	fileName := "valut-svc-firebase-adminsdk4.json"
	filePath := filepath.Join(currentDir, fileName)

	// Set the environment variable
	err = os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", filePath)
	if err != nil {
		fmt.Println("Error setting environment variable:", err)
		return
	}

	// Create a list of topics and their subscribers
	topicSubscribers := []TopicSubscriber{
		{Topic: "topic-otp", Subscription: "topic-otp-sub"},
		// {Topic: "topic2", Subscription: "topic2-sub"},
		// Add more topics and subscribers as needed
	}

	// Use a WaitGroup to wait for all subscribers to finish
	var wg sync.WaitGroup

	for _, ts := range topicSubscribers {
		wg.Add(1)

		go func(ts TopicSubscriber) {
			defer wg.Done()

			fmt.Printf("Starting subscriber for topic: %s, subscription: %s\n", ts.Topic, ts.Subscription)

			if err := pullMsgs(os.Stdout, projectID, ts.Subscription); err != nil {
				fmt.Printf("Error receiving messages for topic %s: %v\n", ts.Topic, err)
			}
		}(ts)
	}

	fmt.Println("Press Enter to stop receiving messages...")
	ready := make(chan struct{})

	go func() {
		// Wait for user input to stop the program
		bufio.NewReader(os.Stdin).ReadString('\n')
		close(ready)
	}()

	// Wait for all subscribers to finish
	wg.Wait()

	// Close the ready channel to signal that all subscribers have completed
	<-ready
}
