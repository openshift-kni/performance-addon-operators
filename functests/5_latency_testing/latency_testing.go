package __latency_testing

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

const (
	//tool to test
	oslat       = "oslat"
	cyclictest  = "cyclictest"
	hwlatdetect = "hwlatdetect"
	//Environment variables names
	latencyTestDelay     = "LATENCY_TEST_DELAY"
	latencyTestRun       = "LATENCY_TEST_RUN"
	latencyTestRuntime   = "LATENCY_TEST_RUNTIME"
	maximumLatency       = "MAXIMUM_LATENCY"
	oslatMaxLatency      = "OSLAT_MAXIMUM_LATENCY"
	hwlatdetecMaxLatency = "HWLATDETECT_MAXIMUM_LATENCY"
	cyclictestMaxLatency = "CYCLICTEST_MAXIMUM_LATENCY"
	latencyTestCpus      = "LATENCY_TEST_CPUS"
	//invalid values error messages
	unexpectedError = "Unexpected error"
	//incorrect values error messages
	incorrectMsgPart1              = "the environment variable "
	incorrectMsgPart2              = " has incorrect value"
	maxInt                         = "2147483647"
	mustBePositiveInt              = ".*it must be a positive integer with maximum value of " + maxInt
	mustBeNonNegativeInt           = ".*it must be a non-negative integer with maximum value of " + maxInt
	incorrectCpuNumber             = incorrectMsgPart1 + latencyTestCpus + incorrectMsgPart2 + mustBePositiveInt
	nonpositiveCpuNumber           = incorrectMsgPart1 + latencyTestCpus + " has a nonpositive value" + mustBePositiveInt
	incorrectDelay                 = incorrectMsgPart1 + latencyTestDelay + incorrectMsgPart2 + mustBeNonNegativeInt
	negativeDelay                  = incorrectMsgPart1 + latencyTestDelay + " has a negative value" + mustBeNonNegativeInt
	incorrectMaxLatency            = incorrectMsgPart1 + maximumLatency + incorrectMsgPart2 + mustBeNonNegativeInt
	negativeMaxLatency             = incorrectMsgPart1 + maximumLatency + " has a negative value" + mustBeNonNegativeInt
	incorrectOslatMaxLatency       = incorrectMsgPart1 + "\"" + oslatMaxLatency + "\"" + incorrectMsgPart2 + mustBeNonNegativeInt
	negativeOslatMaxLatency        = incorrectMsgPart1 + "\"" + oslatMaxLatency + "\" has a negative value" + mustBeNonNegativeInt
	incorrectCyclictestMaxLatency  = incorrectMsgPart1 + "\"" + cyclictestMaxLatency + "\"" + incorrectMsgPart2 + mustBeNonNegativeInt
	negativeCyclictestMaxLatency   = incorrectMsgPart1 + "\"" + cyclictestMaxLatency + "\" has a negative value" + mustBeNonNegativeInt
	incorrectHwlatdetectMaxLatency = incorrectMsgPart1 + "\"" + hwlatdetecMaxLatency + "\"" + incorrectMsgPart2 + mustBeNonNegativeInt
	negativeHwlatdetectMaxLatency  = incorrectMsgPart1 + "\"" + hwlatdetecMaxLatency + "\" has a negative value" + mustBeNonNegativeInt
	incorrectTestRun               = incorrectMsgPart1 + latencyTestRun + incorrectMsgPart2
	incorrectRuntime               = incorrectMsgPart1 + latencyTestRuntime + incorrectMsgPart2 + mustBePositiveInt
	nonpositiveRuntime             = incorrectMsgPart1 + latencyTestRuntime + " has a nonpositive value" + mustBePositiveInt
	//success messages regex
	success = `SUCCESS.*1 Passed.*0 Failed.*2 Skipped`
	//failure messages regex
	latencyFail = `The current latency .* is bigger than the expected one`
	fail        = `FAIL.*0 Passed.*1 Failed.*2 Skipped`
	//hwlatdetect fail message regex
	hwlatdetectFail = `Samples exceeding threshold: [^0]`
	//skip messages regex
	skipTestRun    = `Skip the latency test, the latencyTestRun set to false`
	skipMaxLatency = `no maximum latency value provided, skip buckets latency check`
	skip           = `SUCCESS.*0 Passed.*0 Failed.*3 Skipped`

	//used values parameters
	guaranteedLatency = "20000"
	negativeTesting   = false
	positiveTesting   = true
)

//Struct to hold each test parameters
type latencyTest struct {
	testDelay             string
	testRun               string
	testRuntime           string
	testMaxLatency        string
	oslatMaxLatency       string
	cyclictestMaxLatency  string
	hwlatdetectMaxLatency string
	testCpus              string
	outputMsgs            []string
	toolToTest            string
}

var _ = table.DescribeTable("Test latency measurement tools tests", func(testGroup []latencyTest, isPositiveTest bool) {
	for _, test := range testGroup {
		clearEnv()
		testDescription := setEnvAndGetDescription(test)
		By(testDescription)
		if _, err := os.Stat("../../build/_output/bin/latency-e2e.test"); os.IsNotExist(err) {
			Skip("The executable test file does not exist , skipping the test.")
		}
		output, err := exec.Command("../../build/_output/bin/latency-e2e.test", "-ginkgo.focus", test.toolToTest).Output()
		if err != nil {
			fmt.Println(err.Error())
		}
		//in any case we should not see "unexpected error" message in the output , if the test expected to fail it must do that gracefully
		if isPositiveTest {
			Expect(string(output)).NotTo(MatchRegexp(unexpectedError), "Unexpected error was detected in a positve test")
		}
		for _, msg := range test.outputMsgs {
			Expect(output).To(MatchRegexp(msg), "The output of the executed tool is not as expected")
		}
	}
},
	table.Entry("[test_id:42851] Latency tools shouldn't run with default environment variables values", []latencyTest{{"", "", "", "", "", "", "", "", []string{skip, skipTestRun}, ""}}, positiveTesting),
	table.Entry("[test_id:42850] Oslat - Verify that the tool is working properly with valid environment variables values", getValidValuesTests(oslat), positiveTesting),
	table.Entry("[test_id:42853] Oslat - Verify that the latency tool test should print an expected error message when passing invalid environment variables values", getNegativeTests(oslat), negativeTesting),
	table.Entry("[test_id:42115] Cyclictest - Verify that the tool is working properly with valid environment variables values", getValidValuesTests(cyclictest), positiveTesting),
	table.Entry("[test_id:42852] Cyclictest - Verify that the latency tool test should print an expected error message when passing invalid environment variables values", getNegativeTests(cyclictest), negativeTesting),
	table.Entry("[test_id:42849] Hwlatdetect - Verify that the tool is working properly with valid environment variables values", getValidValuesTests(hwlatdetect), positiveTesting),
	table.Entry("[test_id:42856] Hwlatdetect - Verify that the latency tool test should print an expected error message when passing invalid environment variables values", getNegativeTests(hwlatdetect), negativeTesting),
)

func setEnvAndGetDescription(tst latencyTest) string {
	sb := bytes.NewBufferString("")
	testName := tst.toolToTest
	if tst.toolToTest == "" {
		testName = "latency tools"
	}
	fmt.Fprintf(sb, "Run %s test : \n", testName)
	nonDefaultValues := false
	if tst.testDelay != "" {
		setEnvWriteDescription(latencyTestDelay, tst.testDelay, sb, &nonDefaultValues)
	}
	if tst.testRun != "" {
		setEnvWriteDescription(latencyTestRun, tst.testRun, sb, &nonDefaultValues)
	}
	if tst.testRuntime != "" {
		setEnvWriteDescription(latencyTestRuntime, tst.testRuntime, sb, &nonDefaultValues)
	}
	if tst.testMaxLatency != "" {
		setEnvWriteDescription(maximumLatency, tst.testMaxLatency, sb, &nonDefaultValues)
	}
	if tst.oslatMaxLatency != "" {
		setEnvWriteDescription(oslatMaxLatency, tst.oslatMaxLatency, sb, &nonDefaultValues)
	}
	if tst.cyclictestMaxLatency != "" {
		setEnvWriteDescription(cyclictestMaxLatency, tst.cyclictestMaxLatency, sb, &nonDefaultValues)
	}
	if tst.hwlatdetectMaxLatency != "" {
		setEnvWriteDescription(hwlatdetecMaxLatency, tst.hwlatdetectMaxLatency, sb, &nonDefaultValues)
	}
	if tst.testCpus != "" {
		setEnvWriteDescription(latencyTestCpus, tst.testCpus, sb, &nonDefaultValues)
	}
	if !nonDefaultValues {
		fmt.Fprint(sb, "With default values of the environment variables")
	}

	return sb.String()
}

func setEnvWriteDescription(envVar string, val string, sb *bytes.Buffer, flag *bool) {
	os.Setenv(envVar, val)
	fmt.Fprintf(sb, "%s = %s \n", envVar, val)
	*flag = true
}

func clearEnv() {
	os.Unsetenv(latencyTestDelay)
	os.Unsetenv(latencyTestRun)
	os.Unsetenv(latencyTestRuntime)
	os.Unsetenv(maximumLatency)
	os.Unsetenv(oslatMaxLatency)
	os.Unsetenv(cyclictestMaxLatency)
	os.Unsetenv(hwlatdetecMaxLatency)
	os.Unsetenv(latencyTestCpus)
}

func getValidValuesTests(toolToTest string) []latencyTest {
	var testSet []latencyTest
	testSet = append(testSet, latencyTest{testDelay: "0", testRun: "true", testRuntime: "5", testMaxLatency: guaranteedLatency, testCpus: "3", outputMsgs: []string{success}, toolToTest: toolToTest})
	testSet = append(testSet, latencyTest{testDelay: "0", testRun: "true", testRuntime: "1", testMaxLatency: guaranteedLatency, outputMsgs: []string{success}, toolToTest: toolToTest})
	testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "5", outputMsgs: []string{skip, skipMaxLatency}, toolToTest: toolToTest})
	if toolToTest == oslat {
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "5", testMaxLatency: "1", oslatMaxLatency: guaranteedLatency, outputMsgs: []string{success}, toolToTest: toolToTest})
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "5", oslatMaxLatency: guaranteedLatency, outputMsgs: []string{success}, toolToTest: toolToTest})
	}
	if toolToTest == cyclictest {
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "5", testMaxLatency: "1", cyclictestMaxLatency: guaranteedLatency, outputMsgs: []string{success}, toolToTest: toolToTest})
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "5", cyclictestMaxLatency: guaranteedLatency, outputMsgs: []string{success}, toolToTest: toolToTest})

	}
	if toolToTest == hwlatdetect {
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "5", testMaxLatency: "1", hwlatdetectMaxLatency: guaranteedLatency, outputMsgs: []string{success}, toolToTest: toolToTest})
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "5", hwlatdetectMaxLatency: guaranteedLatency, outputMsgs: []string{success}, toolToTest: toolToTest})

	}
	return testSet
}

func getNegativeTests(toolToTest string) []latencyTest {
	var testSet []latencyTest
	latencyFailureMsg := latencyFail
	if toolToTest == hwlatdetect {
		latencyFailureMsg = hwlatdetectFail
	}

	testSet = append(testSet, latencyTest{testDelay: "0", testRun: "true", testRuntime: "5", testMaxLatency: "1", outputMsgs: []string{latencyFailureMsg, fail}, toolToTest: toolToTest})
	testSet = append(testSet, latencyTest{testRun: "yes", testRuntime: "5", testMaxLatency: "1", outputMsgs: []string{incorrectTestRun, fail}, toolToTest: toolToTest})
	testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "-1", testMaxLatency: "1", outputMsgs: []string{nonpositiveRuntime, fail}, toolToTest: toolToTest})
	testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "5", testMaxLatency: "-2", outputMsgs: []string{negativeMaxLatency, fail}, toolToTest: toolToTest})
	testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "1H", outputMsgs: []string{incorrectRuntime, fail}, toolToTest: toolToTest})
	testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "2", testMaxLatency: "&", outputMsgs: []string{incorrectMaxLatency, fail}, toolToTest: toolToTest})
	testSet = append(testSet, latencyTest{testDelay: "J", testRun: "true", outputMsgs: []string{incorrectDelay, fail}, toolToTest: toolToTest})
	testSet = append(testSet, latencyTest{testDelay: "-5", testRun: "true", outputMsgs: []string{negativeDelay, fail}, toolToTest: toolToTest})
	testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "2", testMaxLatency: "1", testCpus: "p", outputMsgs: []string{incorrectCpuNumber, fail}, toolToTest: toolToTest})
	testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "2", testCpus: "-1", outputMsgs: []string{nonpositiveCpuNumber, fail}, toolToTest: toolToTest})
	if toolToTest == oslat {
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "2", oslatMaxLatency: "&", outputMsgs: []string{incorrectOslatMaxLatency, fail}, toolToTest: toolToTest})
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "2", oslatMaxLatency: "-3", outputMsgs: []string{negativeOslatMaxLatency, fail}, toolToTest: toolToTest})
	}
	if toolToTest == cyclictest {
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "2", cyclictestMaxLatency: "&", outputMsgs: []string{incorrectCyclictestMaxLatency, fail}, toolToTest: toolToTest})
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "2", cyclictestMaxLatency: "-3", outputMsgs: []string{negativeCyclictestMaxLatency, fail}, toolToTest: toolToTest})
	}
	if toolToTest == hwlatdetect {
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "2", hwlatdetectMaxLatency: "&", outputMsgs: []string{incorrectHwlatdetectMaxLatency, fail}, toolToTest: toolToTest})
		testSet = append(testSet, latencyTest{testRun: "true", testRuntime: "2", hwlatdetectMaxLatency: "-3", outputMsgs: []string{negativeHwlatdetectMaxLatency, fail}, toolToTest: toolToTest})
	}
	return testSet
}
