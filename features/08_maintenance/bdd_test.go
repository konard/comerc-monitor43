//go:build bdd

package maintenance_test

import (
	"testing"

	"github.com/raul/monitor/features/suite"
)

func Test(t *testing.T) { suite.RunEpic(t) }
