package server

import (
	"testing"
	"time"
)

func TestBroadcasterPublishDelivers(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Subscribe()
	defer b.Unsubscribe(ch)

	b.Publish("hello")

	select {
	case got := <-ch:
		if got != "hello" {
			t.Fatalf("got %q want %q", got, "hello")
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for message")
	}
}

func TestBroadcasterUnsubscribeStopsDelivery(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Subscribe()
	b.Unsubscribe(ch)

	b.Publish("should not arrive")

	select {
	case msg := <-ch:
		t.Fatalf("unexpected message %q after unsubscribe", msg)
	case <-time.After(50 * time.Millisecond):
		// correct: nothing delivered
	}
}

func TestBroadcasterMultipleSubscribers(t *testing.T) {
	b := NewBroadcaster()
	ch1 := b.Subscribe()
	ch2 := b.Subscribe()
	defer b.Unsubscribe(ch1)
	defer b.Unsubscribe(ch2)

	b.Publish("ping")

	for _, ch := range []chan string{ch1, ch2} {
		select {
		case got := <-ch:
			if got != "ping" {
				t.Fatalf("got %q", got)
			}
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestBroadcasterSlowClientDropped(t *testing.T) {
	b := NewBroadcaster()
	ch := b.Subscribe() // buffer=4
	defer b.Unsubscribe(ch)

	// overflow the buffer — should not block
	for i := 0; i < 10; i++ {
		b.Publish("msg")
	}
}
