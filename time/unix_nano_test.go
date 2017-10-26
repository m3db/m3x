package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestUnixNano(t *testing.T) {
	time := time.Unix(0, 1000)
	unixNano := ToUnixNano(time)
	require.Equal(t, UnixNano(1000), unixNano)
	require.Equal(t, time, unixNano.ToTime())
}
