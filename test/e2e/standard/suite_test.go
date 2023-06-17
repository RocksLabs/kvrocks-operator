package standard_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStandard(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Standard Suite")
}
