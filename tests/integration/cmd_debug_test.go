package integration

import (
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/openshift/odo/tests/helper"
)

var _ = FDescribe("odo debug command tests", func() {
	var project string
	var context string

	// Setup up state for each test spec
	// create new project (not set as active) and new context directory for each test spec
	// This is before every spec (It)
	BeforeEach(func() {
		SetDefaultEventuallyTimeout(10 * time.Minute)
		SetDefaultConsistentlyDuration(30 * time.Second)
		context = helper.CreateNewContext()
		project = helper.CreateRandProject()
	})

	// Clean up after the test
	// This is run after every Spec (It)
	AfterEach(func() {
		helper.DeleteProject(project)
		helper.DeleteDir(context)
	})

	Context("odo debug on a nodejs component", func() {
		It("should expect a ws connection when tried to connect on debug port locally", func() {
			helper.CopyExample(filepath.Join("source", "nodejs"), context)
			helper.CmdShouldPass("odo", "component", "create", "nodejs", "--project", project, "--context", context)
			helper.CmdShouldFail("odo", "push", "--context", context)

			go func() {
				helper.CmdShouldRunWithTimeout(20*time.Second, "odo", "experimental", "debug", "port-forward", "--context", context)
			}()

			// debug port
			helper.HttpWaitForWithStatus("http://localhost:5858", "WebSockets request was expected", 10, 2, 400)
		})
	})
})