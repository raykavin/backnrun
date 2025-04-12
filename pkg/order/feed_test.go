package order

import (
	"testing"

	"github.com/raykavin/backnrun/internal/core"
	"github.com/stretchr/testify/require"
)

func TestFeed_NewOrderFeed(t *testing.T) {
	feed := NewOrderFeed()
	require.NotEmpty(t, feed)
}

func TestFeed_Subscribe(t *testing.T) {
	feed, pair := NewOrderFeed(), "blaus"
	called := make(chan bool, 1)

	feed.Subscribe(pair, func(_ core.Order) {
		called <- true
	}, false)

	feed.Start()
	feed.Publish(core.Order{Pair: pair}, false)
	require.True(t, <-called)
}
