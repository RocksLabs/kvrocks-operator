package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func Test_RunE2E(t *testing.T) {
	// defer GinkgoRecover()
	runE2E()
	RegisterFailHandler(Fail)

	// TODO use RunSpecs to substitute RunSpecsWithDefaultAndCustomReporters, will be solve in test pr (jiayouxujin)
	// RunSpecsWithDefaultAndCustomReporters(t,
	// 	"Controller Suite",
	// 	[]Reporter{printer.NewlineReporter{}})
}
