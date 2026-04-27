//go:build bdd

package alerting_test

import (
	"testing"

	"github.com/raul/monitor/features/suite"
)

func Test(t *testing.T) { suite.RunEpic(t) }
