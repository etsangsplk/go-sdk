package event

import (
	"bytes"
	"encoding/gob"
	"github.com/nsqio/go-nsq"
	"github.com/nsqio/nsq/nsqd"
	snsq "github.com/segmentio/nsq-go"
	"time"
)

const NSQ_MAX_IN_FLIGHT int = 100 // Ideally match http.DefaultTransport.MaxIdleConns
const NSQ_CONSUMER_CHANNEL string = "optimizely"
const NSQ_LISTEN_SPEC string = "localhost:4150"
const NSQ_REQUEUE_WAIT time.Duration = time.Duration(30) * time.Second
const NSQ_TOPIC string = "user_event"

var embedded_nsqd *nsqd.NSQD = nil

var done = make(chan bool)

type NSQQueue struct {
	p *nsq.Producer
	c *snsq.Consumer
	messages Queue
}

// Get returns queue for given count size
func (i *NSQQueue) Get(count int) []interface{} {

	events := []interface{}{}

	messages := i.messages.Get(count)
	for _, message := range messages {
		mess, ok := message.(snsq.Message)
		if !ok {
			continue
		}
		events = append(events, i.decodeMessage(mess.Body))
	}

	return events
	//for {
	//	select {
	//	case res := <-i.c.Messages():
	//		i.messages = append(i.messages, res)
	//		event := i.decodeMessage(res.Body)
	//		events = append(events, event)
	//		if len(events) == count {
	//			return events
	//		}
	//	case <-time.After(30 * time.Second):
	//		fmt.Println("timeout 30 seconds")
	//		fmt.Println(len(events))
	//		return events
	//	}
	//}
	return events
}

// Add appends item to queue
func (i *NSQQueue) Add(item interface{}) {
	event, ok := item.(UserEvent)
	if !ok {
		// cannot add non-user events
		return
	}
//	go func() {
		buf := new(bytes.Buffer)
		enc := gob.NewEncoder(buf)
		enc.Encode(event)
		i.p.Publish(NSQ_TOPIC, buf.Bytes())
//	}()
}

func (i *NSQQueue) decodeMessage(body []byte) UserEvent {
	reader := bytes.NewReader(body)
	dec := gob.NewDecoder(reader)
	event := UserEvent{}
	dec.Decode(&event)

	return event
}

// Remove removes item from queue and returns elements slice
func (i *NSQQueue) Remove(count int) []interface{} {
	userEvents := make([]interface{},0, count)
	events := i.messages.Remove(count)
	for _,message := range events {
		mess, ok := message.(snsq.Message)
		if !ok {
			continue
		}
		userEvent := i.decodeMessage(mess.Body)
		mess.Finish()
		userEvents = append(userEvents, userEvent)
	}
	return userEvents
}

//func (h *NSQQueue) HandleMessage(m *nsq.Message) error {
//	if len(m.Body) == 0 {
//		// returning an error results in the message being re-enqueued
//		// a REQ is sent to nsqd
//		//return errors.New("body is blank re-enqueue message")
//	}
//
//	h.messages = append(h.messages, m)
//	// Let's log our message!
//	//log.Print(m.Body)
//
//	// Returning nil signals to the consumer that the message has
//	// been handled with success. A FIN is sent to nsqd
//	return nil
//}

// Size returns size of queue
func (i *NSQQueue) Size() int {
	return i.messages.Size()
}

// NewNSQueue returns new NSQ based queue with given queueSize
func NewNSQueue(queueSize int) Queue {

	// Run nsqd embedded
	if embedded_nsqd == nil {
		go func() {
			// running an nsqd with all of the default options
			// (as if you ran it from the command line with no flags)
			// is literally these three lines of code. the nsqd
			// binary mainly wraps up the handling of command
			// line args and does something similar
			if embedded_nsqd == nil {
				opts := nsqd.NewOptions()
				embedded_nsqd, _ := nsqd.New(opts)
				embedded_nsqd.Main()

				// wait until we are told to continue and exit
				<-done
				embedded_nsqd.Exit()
			}
		}()
	}

	nsqConfig := nsq.NewConfig()
	p, err := nsq.NewProducer(NSQ_LISTEN_SPEC, nsqConfig)
	if err != nil {
		//log.Fatal(err)
	}

	//c, err := nsq.NewConsumer(NSQ_TOPIC, "NSQQueue", nsqConfig)
	//if err != nil {
	//	//log.Panic("Could not create consumer")
	//}
	//
	//i := &NSQQueue{p:p,c:c, messages:[]*nsq.Message{}}
	//
	//c.ChangeMaxInFlight(1)
	//c.AddConcurrentHandlers(i, 1)
	//
	//err = c.ConnectToNSQD(NSQ_LISTEN_SPEC)
	//
	//if err != nil {
	//	// problem
	//}

	consumer, _ := snsq.StartConsumer(snsq.ConsumerConfig{
		Topic:       NSQ_TOPIC,
		Channel:     NSQ_CONSUMER_CHANNEL,
		Address:     NSQ_LISTEN_SPEC,
		MaxInFlight: queueSize,
	})

	i := &NSQQueue{p:p,c:consumer, messages:NewInMemoryQueue(queueSize)}

	go func() {
		for message := range i.c.Messages() {
			i.messages.Add(message)
		}
	}()

	return i
}
