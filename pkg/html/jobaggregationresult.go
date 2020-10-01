package html

import (
	"fmt"

	sippyprocessingv1 "github.com/openshift/sippy/pkg/apis/sippyprocessing/v1"
)

// PlatformResults
type jobAggregationDisplay struct {
	displayName             string
	totalJobRuns            int
	displayPercentage       float64
	parentDisplayPercentage float64

	// jobResults for all jobs that match this platform, ordered by lowest JobRunPassPercentage to highest
	jobResults []jobResultDisplay

	// TestResults holds entries for each test that is a part of this aggregation.  Each entry aggregates the results of all runs of a single test.  The array is sorted from lowest JobRunPassPercentage to highest JobRunPassPercentage
	testResults []testResultDisplay
}

func platformResultToDisplay(in sippyprocessingv1.PlatformResults) jobAggregationDisplay {
	ret := jobAggregationDisplay{
		displayName:             in.PlatformName,
		totalJobRuns:            in.JobRunSuccesses + in.JobRunFailures,
		displayPercentage:       in.JobRunPassPercentage,
		parentDisplayPercentage: in.JobRunPassPercentageWithoutInfrastructureFailures,
	}

	for _, jobResult := range in.JobResults {
		ret.jobResults = append(ret.jobResults, jobResultToDisplay(jobResult))
	}
	for _, testResult := range in.AllTestResults {
		ret.testResults = append(ret.testResults, testResultToDisplay(testResult))
	}

	return ret
}

type jobAggregationResultRenderBuilder struct {
	// sectionBlock needs to be unique for each part of the report.  It is used to uniquely name the collapse/expand
	// sections so they open properly
	sectionBlock string

	currAggregationResult jobAggregationDisplay
	prevAggregationResult *jobAggregationDisplay

	release              string
	maxTestResultsToShow int
	maxJobResultsToShow  int
	colors               colorizationCriteria
	collapsedAs          string
}

func newJobAggregationResultRenderer(sectionBlock string, currJobResult jobAggregationDisplay, release string) *jobAggregationResultRenderBuilder {
	return &jobAggregationResultRenderBuilder{
		sectionBlock:          sectionBlock,
		currAggregationResult: currJobResult,
		release:               release,
		maxTestResultsToShow:  10, // just a default, can be overridden
		maxJobResultsToShow:   10, // just a default, can be overridden
		colors: colorizationCriteria{
			minRedPercent:    0,  // failure.  In this range, there is a systemic failure so severe that a reliable signal isn't available.
			minYellowPercent: 60, // at risk.  In this range, there is a systemic problem that needs to be addressed.
			minGreenPercent:  80, // no action required. This *should* be closer to 85%
		},
	}
}
func newJobAggregationResultRendererFromPlatformResults(sectionBlock string, curr sippyprocessingv1.PlatformResults, release string) *jobAggregationResultRenderBuilder {
	return newJobAggregationResultRenderer(sectionBlock, platformResultToDisplay(curr), release)
}

func (b *jobAggregationResultRenderBuilder) withPrevious(prevJobResult *jobAggregationDisplay) *jobAggregationResultRenderBuilder {
	b.prevAggregationResult = prevJobResult
	return b
}

func (b *jobAggregationResultRenderBuilder) withPreviousPlatformResults(prev *sippyprocessingv1.PlatformResults) *jobAggregationResultRenderBuilder {
	if prev == nil {
		b.prevAggregationResult = nil
		return b
	}
	t := platformResultToDisplay(*prev)
	b.prevAggregationResult = &t
	return b
}

func (b *jobAggregationResultRenderBuilder) withMaxTestResultsToShow(maxTestResultsToShow int) *jobAggregationResultRenderBuilder {
	b.maxTestResultsToShow = maxTestResultsToShow
	return b
}

func (b *jobAggregationResultRenderBuilder) withMaxJobResultsToShow(maxJobResultsToShow int) *jobAggregationResultRenderBuilder {
	b.maxJobResultsToShow = maxJobResultsToShow
	return b
}

func (b *jobAggregationResultRenderBuilder) withColors(colors colorizationCriteria) *jobAggregationResultRenderBuilder {
	b.colors = colors
	return b
}

func (b *jobAggregationResultRenderBuilder) startCollapsedAs(collapsedAs string) *jobAggregationResultRenderBuilder {
	b.collapsedAs = collapsedAs
	return b
}

func (b *jobAggregationResultRenderBuilder) toHTML() string {

	s := ""

	// TODO either make this a template or make this a builder that takes args and then has branches.
	//  that will fix the funny link that goes nowhere.
	template := `
			<tr class="%s">
				<td>
					%s
					<p>
					%s
				</td>
				<td>
					%0.2f%% (%0.2f%%)<span class="text-nowrap">(%d runs)</span>
				</td>
				<td>
					%s
				</td>
				<td>
					%0.2f%% (%0.2f%%)<span class="text-nowrap">(%d runs)</span>
				</td>
			</tr>
		`

	naTemplate := `
			<tr class="%s">
				<td>
					%s
					<p>
					%s
				</td>
				<td>
					%0.2f%% (%0.2f%%)<span class="text-nowrap">(%d runs)</span>
				</td>
				<td/>
				<td>
					NA
				</td>
			</tr>
		`

	class := b.colors.getColor(b.currAggregationResult.displayPercentage)
	if len(b.collapsedAs) > 0 {
		class += " collapse " + b.collapsedAs
	}

	testsCollapseName := makeSafeForCollapseName(b.sectionBlock + "---" + b.currAggregationResult.displayName + "---tests")
	jobsCollapseName := makeSafeForCollapseName(b.sectionBlock + "---" + b.currAggregationResult.displayName + "---jobs")
	button := ""
	button += "					" + getButtonHTML(testsCollapseName, "Expand Failing Tests")
	button += "					" + getButtonHTML(jobsCollapseName, "Expand Failing Jobs")

	if b.prevAggregationResult != nil {
		arrow := getArrow(b.currAggregationResult.totalJobRuns, b.currAggregationResult.displayPercentage, b.prevAggregationResult.displayPercentage)

		s = s + fmt.Sprintf(template,
			class,
			b.currAggregationResult.displayName,
			button,
			b.currAggregationResult.displayPercentage,
			b.currAggregationResult.parentDisplayPercentage,
			b.currAggregationResult.totalJobRuns,
			arrow,
			b.prevAggregationResult.displayPercentage,
			b.prevAggregationResult.parentDisplayPercentage,
			b.prevAggregationResult.totalJobRuns,
		)
	} else {
		s = s + fmt.Sprintf(naTemplate,
			class,
			b.currAggregationResult.displayName,
			button,
			b.currAggregationResult.displayPercentage,
			b.currAggregationResult.parentDisplayPercentage,
			b.currAggregationResult.totalJobRuns,
		)
	}

	// now render the individual jobs
	jobCount := b.maxJobResultsToShow
	jobRowCount := 0
	jobRows := ""
	jobAdditionalMatches := 0
	for _, job := range b.currAggregationResult.jobResults {
		if jobCount <= 0 {
			jobAdditionalMatches++
			continue
		}
		jobCount--

		var prev *jobResultDisplay
		if b.prevAggregationResult != nil {
			for _, prevJobInstance := range b.prevAggregationResult.jobResults {
				if prevJobInstance.displayName == job.displayName {
					prev = &prevJobInstance
					break
				}
			}
		}

		jobRows = jobRows + newJobResultRenderer(jobsCollapseName, job, b.release).
			withPrevious(prev).
			withMaxTestResultsToShow(b.maxTestResultsToShow).
			startCollapsed().
			withIndent(1).
			toHTML()

		jobRowCount++
	}
	if jobAdditionalMatches > 0 {
		jobRows += fmt.Sprintf(`<tr class="collapse %s"><td colspan=2 style="padding-left:60px">Plus %d more jobs</td></tr>`, jobsCollapseName, jobAdditionalMatches)
	}
	if jobRowCount > 0 {
		s = s + fmt.Sprintf(`<tr class="collapse %s"><td colspan=2 style="padding-left:60px" class="font-weight-bold">Job Name</td><td class="font-weight-bold">Job Pass Rate</td></tr>`, jobsCollapseName)
		s = s + jobRows
		s = s + fmt.Sprintf(`<tr class="collapse %s"><td colspan=3 style="padding-left:60px" class="font-weight-bold"></td><td class="font-weight-bold"></td></tr>`, jobsCollapseName)
	} else {
		s = s + fmt.Sprintf(`<tr class="collapse %s"><td colspan=3 style="padding-left:60px" class="font-weight-bold">No Jobs Matched Filters</td></tr>`, jobsCollapseName)
	}

	testCount := b.maxTestResultsToShow
	testRowCount := 0
	testRows := ""
	testAdditionalMatches := 0
	for _, test := range b.currAggregationResult.testResults {
		if testCount <= 0 {
			testAdditionalMatches++
			continue
		}
		testCount--

		var prev *testResultDisplay
		if b.prevAggregationResult != nil {
			for _, prevInstance := range b.prevAggregationResult.testResults {
				if prevInstance.displayName == test.displayName {
					prev = &prevInstance
					break
				}
			}
		}

		testRows = testRows +
			newTestResultRenderer(testsCollapseName, test, b.release).
				withIndent(1).
				withPrevious(prev).
				startCollapsed().
				toHTML()

		testRowCount++
	}
	if testAdditionalMatches > 0 {
		testRows += fmt.Sprintf(`<tr class="collapse %s"><td colspan=2 style="padding-left:60px">Plus %d more tests</td></tr>`, testsCollapseName, testAdditionalMatches)
	}
	if testRowCount > 0 {
		s = s + fmt.Sprintf(`<tr class="collapse %s"><td colspan=2 style="padding-left:60px" class="font-weight-bold">Test Name</td><td class="font-weight-bold">Test Pass Rate</td></tr>`, testsCollapseName)
		s = s + testRows
		s = s + fmt.Sprintf(`<tr class="collapse %s"><td colspan=2 style="padding-left:60px" class="font-weight-bold"></td><td class="font-weight-bold"></td></tr>`, testsCollapseName)
	} else {
		s = s + fmt.Sprintf(`<tr class="collapse %s"><td colspan=3 style="padding-left:60px" class="font-weight-bold">No Tests Matched Filters</td></tr>`, testsCollapseName)
	}

	return s
}

// aggregationToJobSubsetOverrides provides a mapping to
var aggregationToJobSubsetOverrides = map[string]string{
	"metal":       "metal-upi",
	"realtime":    "rt",
	"vsphere-ipi": "vsphere",
}

func getCIJobSubstring(aggregationName string) string {
	if ret, ok := aggregationToJobSubsetOverrides[aggregationName]; ok {
		return ret
	}
	return aggregationName
}
