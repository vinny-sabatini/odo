//go:build linux || darwin || dragonfly || solaris || openbsd || netbsd || freebsd
// +build linux darwin dragonfly solaris openbsd netbsd freebsd

package interactive

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/redhat-developer/odo/tests/helper"
	"log"
	"path/filepath"
)

var _ = Describe("odo init interactive command tests", func() {

	var commonVar helper.CommonVar

	// This is run before every Spec (It)
	var _ = BeforeEach(func() {
		commonVar = helper.CommonBeforeEach()
		helper.Chdir(commonVar.Context)
	})

	// Clean up after the test
	// This is run after every Spec (It)
	var _ = AfterEach(func() {
		helper.CommonAfterEach(commonVar)
	})

	It("should download correct devfile", func() {

		command := []string{"odo", "init"}
		output, err := helper.RunInteractive(command, nil, func(ctx helper.InteractiveContext) {

			helper.ExpectString(ctx, "Select language")
			helper.SendLine(ctx, "go")

			helper.ExpectString(ctx, "Select project type")
			helper.SendLine(ctx, "\n")

			helper.ExpectString(ctx, "Which starter project do you want to use")
			helper.SendLine(ctx, "\n")

			helper.ExpectString(ctx, "Enter component name")
			helper.SendLine(ctx, "my-go-app")

			helper.ExpectString(ctx, "Your new component \"my-go-app\" is ready in the current directory.")

		})

		Expect(err).To(BeNil())
		Expect(output).To(ContainSubstring("Your new component \"my-go-app\" is ready in the current directory."))
		Expect(helper.ListFilesInDir(commonVar.Context)).To(ContainElements("devfile.yaml"))
	})

	Describe("displaying welcoming messages", func() {

		// testFunc is a function that returns a `Tester` function (intended to be used via `helper.RunInteractive`),
		// which first expects all messages in `welcomingMsgs` to be read from the console,
		// then runs an `additionalTester` and finally expects the asking of a component name
		// (based on the `language` specified)
		testFunc := func(language string, welcomingMsgs []string, additionalTester helper.Tester) helper.Tester {
			return func(ctx helper.InteractiveContext) {
				for _, msg := range welcomingMsgs {
					helper.ExpectString(ctx, msg)
				}

				if additionalTester != nil {
					additionalTester(ctx)
				}

				helper.ExpectString(ctx, "Enter component name")
				helper.SendLine(ctx, fmt.Sprintf("my-%s-app", language))

				helper.ExpectString(ctx,
					fmt.Sprintf("Your new component \"my-%s-app\" is ready in the current directory.", language))
			}
		}

		assertBehavior := func(language string, output string, err error, msgs []string, additionalAsserter func()) {
			Expect(err).To(BeNil())

			lines, err := helper.ExtractLines(output)
			if err != nil {
				log.Fatal(err)
			}
			Expect(len(lines)).To(BeNumerically(">", len(msgs)))
			Expect(lines[0:len(msgs)]).To(Equal(msgs))
			Expect(lines).To(
				ContainElement(fmt.Sprintf("Your new component \"my-%s-app\" is ready in the current directory.", language)))

			Expect(helper.ListFilesInDir(commonVar.Context)).To(ContainElements("devfile.yaml"))

			if additionalAsserter != nil {
				additionalAsserter()
			}
		}

		testRunner := func(language string, welcomingMsgs []string, tester helper.Tester) (string, error) {
			command := []string{"odo", "init"}
			return helper.RunInteractive(command,
				// Setting verbosity level to 0, because we would be asserting the welcoming message is the first
				// message displayed to the end user. So we do not want any potential debug lines to be printed first.
				// Using envvars here (and not via the -v flag), because of https://github.com/redhat-developer/odo/issues/5513
				[]string{"ODO_LOG_LEVEL=0"},
				testFunc(language, welcomingMsgs, tester))
		}

		When("directory is empty", func() {

			BeforeEach(func() {
				Expect(helper.ListFilesInDir(commonVar.Context)).To(HaveLen(0))
			})

			It("should display appropriate welcoming messages", func() {
				language := "java"
				welcomingMsgs := []string{
					"The current directory is empty. odo will help you start a new project.",
				}

				output, err := testRunner(language, welcomingMsgs, func(ctx helper.InteractiveContext) {
					helper.ExpectString(ctx, "Select language")
					helper.SendLine(ctx, language)

					helper.ExpectString(ctx, "Select project type")
					helper.SendLine(ctx, "\n")

					helper.ExpectString(ctx, "Which starter project do you want to use")
					helper.SendLine(ctx, "\n")
				})

				assertBehavior(language, output, err, welcomingMsgs, nil)
			})
		})

		When("directory is not empty", func() {

			BeforeEach(func() {
				helper.CopyExample(filepath.Join("source", "python"), commonVar.Context)
				Expect(helper.ListFilesInDir(commonVar.Context)).To(
					SatisfyAll(
						HaveLen(2),
						ContainElements("requirements.txt", "wsgi.py")))
			})

			It("should display appropriate welcoming messages", func() {
				language := "python"
				welcomingMsgs := []string{
					"The current directory already contains source code. " +
						"odo will try to autodetect the language and project type in order to select the best suited Devfile for your project.",
				}

				output, err := testRunner(language, welcomingMsgs, func(ctx helper.InteractiveContext) {
					helper.ExpectString(ctx, "Based on the files in the current directory odo detected")

					helper.ExpectString(ctx, fmt.Sprintf("Language: %s", language))

					helper.ExpectString(ctx, fmt.Sprintf("Project type: %s", language))

					helper.ExpectString(ctx,
						fmt.Sprintf("The devfile \"%s\" from the registry \"DefaultDevfileRegistry\" will be downloaded.", language))

					helper.ExpectString(ctx, "Is this correct")
					helper.SendLine(ctx, "\n")

					helper.ExpectString(ctx, "Select container for which you want to change configuration")
					helper.SendLine(ctx, "\n")
				})

				assertBehavior(language, output, err, welcomingMsgs, func() {
					// Make sure the original source code files are still present
					Expect(helper.ListFilesInDir(commonVar.Context)).To(
						SatisfyAll(
							HaveLen(3),
							ContainElements("devfile.yaml", "requirements.txt", "wsgi.py")))
				})
			})
		})
	})
})
