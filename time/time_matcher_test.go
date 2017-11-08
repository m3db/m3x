package time

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMatcher(t *testing.T) {
	loc, _ := time.LoadLocation("Asia/Shanghai")
	t1 := time.Now().UTC()
	t2 := t1.In(loc)

	require.NotEqual(t, t1, t2)

	t1Matcher := NewMatcher(t1)
	require.True(t, t1Matcher.Matches(t2))

	require.NotEqual(t, t1Matcher, t2.Add(1*time.Hour))
}
