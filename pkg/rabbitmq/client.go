package rabbitmq

import (
	"fmt"
	"sync"

	"github.com/streadway/amqp"
)

// Client is a RabbitMQ client
type Client struct {
	// connection is the active rabbitmq connection
	connection *amqp.Connection

	// number of consumer queues to listen on
	numConsumerQueues int

	// lastPublishRK contains the last routing index that was used
	// to publish to <string> queue
	lastPublishRk map[string]int

	// rk modification mutex
	rkmutex sync.Mutex
}

// NewClient returns a new rabbitmq client
func NewClient(endpoint string) (*Client, error) {
	conn, err := amqp.Dial(endpoint)
	if err != nil {
		return nil, fmt.Errorf("Failed to dial rabbitmq: %v", err)
	}

	return &Client{
		lastPublishRk: make(map[string]int),
		connection:    conn,
	}, nil
}

// ensureExchange ensures that your exchanges exists. Uses a seperate channel
// to prevent explosions
func (c *Client) ensureExchange(topic string) error {
	aChan, err := c.getChannel()
	if err != nil {
		return err
	}

	return aChan.ExchangeDeclare(topic, "direct", true, false, false, false, amqp.Table{})
}

// ensureConsumerQueues ensures that consumer queues we expect to exist do
func (c *Client) ensureConsumerQueues(topic string) error {
	aChan, err := c.getChannel()
	if err != nil {
		return err
	}

	for i := 0; i != c.numConsumerQueues; i++ {
		queue := c.getRk(topic, i)

		if _, err := aChan.QueueDeclare(queue, true, false, false, false, amqp.Table{}); err != nil {
			return err
		}

		if err := aChan.QueueBind(queue, queue, topic, false, amqp.Table{}); err != nil {
			return err
		}
	}

	return nil
}

// getChannel creates a new channel
func (c *Client) getChannel() (*amqp.Channel, error) {
	return c.connection.Channel()
}

// Channel returns a raw RabbitMQ channel
func (c *Client) Channel() (*amqp.Channel, error) {
	return c.connection.Channel()
}

// getRK gets the expected queue and rk name for a numberic consumer
func (c *Client) getRk(topic string, rkIndex int) string {
	return fmt.Sprintf("%s-%d", topic, rkIndex)
}

// Publish a message to an exchange, must be a serialized format
func (c *Client) Publish(topic string, body []byte) error {
	aChan, err := c.getChannel()
	if err != nil {
		return err
	}

	rkIndex := c.lastPublishRk[topic]
	rk := c.getRk(topic, rkIndex)

	c.rkmutex.Lock()
	c.lastPublishRk[topic]++
	if c.lastPublishRk[topic] == c.numConsumerQueues {
		c.lastPublishRk[topic] = 0
	}
	c.rkmutex.Unlock()

	return aChan.Publish(topic, rk, false, false, amqp.Publishing{
		DeliveryMode: amqp.Persistent,
		ContentType:  "application/octet-stream",
		Body:         body,
	})
}

// Consume from a RabbitMQ queue
func (c *Client) Consume(topic string) (<-chan amqp.Delivery, error) {
	aChan, err := c.getChannel()
	if err != nil {
		return nil, err
	}

	multiplexer := make(chan amqp.Delivery)
	for i := 0; i != c.numConsumerQueues; i++ {
		queue := c.getRk(topic, i)
		ch, err := aChan.Consume(queue, "", false, false, false, false, nil)
		if err != nil {
			return nil, err
		}

		// pipe from this consumer into multiplexed channel
		go func() {
			for {
				msg := <-ch
				multiplexer <- msg
			}
		}()
	}

	return (<-chan amqp.Delivery)(multiplexer), nil
}
