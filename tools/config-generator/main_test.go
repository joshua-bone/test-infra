/*
Copyright 2020 The Knative Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gopkg.in/yaml.v2"
)

func TestNewOutputter(t *testing.T) {
	SetupForTesting()
	out := newOutputter(&bytes.Buffer{})
	if out.count != 0 {
		t.Fatalf("Count should be 0, was %v", out.count)
	}
}

func TestOutputConfig(t *testing.T) {
	SetupForTesting()
	output.outputConfig("")
	if diff := cmp.Diff(GetOutput(), ""); diff != "" {
		t.Errorf("Incorrect output for empty string: (-got +want)\n%s", diff)
	}

	output.outputConfig(" \t\n")
	if diff := cmp.Diff(GetOutput(), ""); diff != "" {
		t.Errorf("Incorrect output for whitespace string: (-got +want)\n%s", diff)
	}
	if output.count != 0 {
		t.Fatalf("Output count should have been 0, but was %d", output.count)
	}

	inputLine := "some-key: some-value"
	output.outputConfig(inputLine)
	if diff := cmp.Diff(GetOutput(), inputLine+"\n"); diff != "" {
		t.Errorf("Incorrect output for whitespace string: (-got +want)\n%s", diff)
	}
	if output.count != 1 {
		t.Fatalf("Output count should have been exactly 1, but was %d", output.count)
	}
}

func TestReadTemplate(t *testing.T) {
	SetupForTesting()
	templatesCache["foo"] = "bar"
	if diff := cmp.Diff(readTemplate("foo"), "bar"); diff != "" {
		t.Errorf("Cached template was not returned: (-got +want)\n%s", diff)
	}

	readTemplate("non/existent/file/path")
	if logFatalCalls != 1 {
		t.Errorf("Non existent file should have caused error")
	}

	delete(templatesCache, "foo")
}

func TestNewbaseProwJobTemplateData(t *testing.T) {
	SetupForTesting()
	out := newbaseProwJobTemplateData("foo/subrepo")
	if diff := cmp.Diff(out.PathAlias, ""); diff != "" {
		t.Errorf("Unexpected path alias: (-got +want)\n%s", diff)
	}

	pathAliasOrgs.Insert("foo")
	out = newbaseProwJobTemplateData("foo/subrepo")
	expected := "path_alias: knative.dev/subrepo"
	if diff := cmp.Diff(out.PathAlias, expected); diff != "" {
		t.Errorf("Unexpected path alias: (-got +want)\n%s", diff)
	}

	nonPathAliasRepos.Insert("foo/subrepo")
	out = newbaseProwJobTemplateData("foo/subrepo")
	if diff := cmp.Diff(out.PathAlias, ""); diff != "" {
		t.Errorf("Unexpected path alias: (-got +want)\n%s", diff)
	}

	// don't pollute the global setup
	pathAliasOrgs.Delete("foo")
	nonPathAliasRepos.Delete("foo/subrepo")
}

func TestCreateCommand(t *testing.T) {
	SetupForTesting()
	preCommand = "" // global
	in := baseProwJobTemplateData{Command: "foo", Args: []string{"bar", "baz"}}
	out := createCommand(in)
	expected := "[foo bar baz]"
	if diff := cmp.Diff(fmt.Sprintf("%v", out), expected); diff != "" {
		t.Errorf("Unexpected command & args list: (-got +want)\n%s", diff)
	}

	preCommand = "expelliarmus"
	out = createCommand(in)
	expected = "[expelliarmus foo bar baz]"
	if diff := cmp.Diff(fmt.Sprintf("%v", out), expected); diff != "" {
		t.Errorf("Unexpected command & args list: (-got +want)\n%s", diff)
	}

	preCommand = ""
}

func TestEnvNameToKey(t *testing.T) {
	SetupForTesting()
	if diff := cmp.Diff(envNameToKey("foo"), "- name: foo"); diff != "" {
		t.Errorf("Unexpected name to key conversion: (-got +want)\n%s", diff)
	}
}

func TestEnvValueToValue(t *testing.T) {
	SetupForTesting()
	if diff := cmp.Diff(envValueToValue("bar"), "  value: bar"); diff != "" {
		t.Errorf("Unexpected env value conversion: (-got +want)\n%s", diff)
	}
}

func TestAddEnvToJob(t *testing.T) {
	SetupForTesting()
	job := baseProwJobTemplateData{}
	job.addEnvToJob("foo", "bar")
	if diff := cmp.Diff(job.Env[0], "- name: foo"); diff != "" {
		t.Errorf("Unexpected env name: (-got +want)\n%s", diff)
	}
	if diff := cmp.Diff(job.Env[1], "  value: bar"); diff != "" {
		t.Errorf("Unexpected env value: (-got +want)\n%s", diff)
	}

	job = baseProwJobTemplateData{}
	job.addEnvToJob("num", "42")
	if diff := cmp.Diff(job.Env[0], "- name: num"); diff != "" {
		t.Errorf("Unexpected env name: (-got +want)\n%s", diff)
	}
	if diff := cmp.Diff(job.Env[1], "  value: \"42\""); diff != "" {
		t.Errorf("Unexpected env value: (-got +want)\n%s", diff)
	}
}

func TestAddLabelToJob(t *testing.T) {
	SetupForTesting()
	job := baseProwJobTemplateData{}
	addLabelToJob(&job, "foo", "bar")
	labelString := fmt.Sprintf("%v", job.Labels)
	expected := "[foo: bar]"
	if diff := cmp.Diff(labelString, expected); diff != "" {
		t.Errorf("Unexpected label string: (-got +want)\n%s", diff)
	}
}

func TestAddMonitoringPubsubLabelsToJob(t *testing.T) {
	SetupForTesting()
	job := baseProwJobTemplateData{}
	addMonitoringPubsubLabelsToJob(&job, "foobar")
	expected := []string{
		"prow.k8s.io/pubsub.project: knative-tests",
		"prow.k8s.io/pubsub.topic: knative-monitoring",
		"prow.k8s.io/pubsub.runID: foobar",
	}
	for i := range expected {
		if diff := cmp.Diff(job.Labels[i], expected[i]); diff != "" {
			t.Errorf("Unexpected pubsub label: (-got +want)\n%s", diff)
		}
	}
}

func TestAddVolumeToJob(t *testing.T) {
	SetupForTesting()
	mountPath := "somePath"
	name := "foo"
	content := []string{"bar", "baz"}

	job := baseProwJobTemplateData{}
	isSecret := false
	addVolumeToJob(&job, mountPath, name, isSecret, content)
	expectedVolumeMounts := []string{
		"- name: foo",
		"  mountPath: somePath",
	}
	for i := range expectedVolumeMounts {
		if diff := cmp.Diff(job.VolumeMounts[i], expectedVolumeMounts[i]); diff != "" {
			t.Errorf("Unexpected volume mount: (-got +want)\n%s", diff)
		}
	}
	expectedVolumes := []string{
		"- name: foo",
		"  bar",
		"  baz",
	}
	for i := range expectedVolumes {
		if diff := cmp.Diff(job.Volumes[i], expectedVolumes[i]); diff != "" {
			t.Errorf("Unexpected volume: (-got +want)\n%s", diff)
		}
	}

	job = baseProwJobTemplateData{}
	isSecret = true
	addVolumeToJob(&job, mountPath, name, isSecret, content)
	expectedVolumeMounts = []string{
		"- name: foo",
		"  mountPath: somePath",
		"  readOnly: true",
	}
	for i := range expectedVolumeMounts {
		if diff := cmp.Diff(job.VolumeMounts[i], expectedVolumeMounts[i]); diff != "" {
			t.Errorf("Unexpected volume mount: (-got +want)\n%s", diff)
		}
	}
	expectedVolumes = []string{
		"- name: foo",
		"  secret:",
		"    secretName: foo",
		"  bar",
		"  baz",
	}
	for i := range expectedVolumes {
		if diff := cmp.Diff(job.Volumes[i], expectedVolumes[i]); diff != "" {
			t.Errorf("Unexpected volume: (-got +want)\n%s", diff)
		}
	}
}

func TestConfigureServiceAccountForJob(t *testing.T) {
	SetupForTesting()
	job := baseProwJobTemplateData{ServiceAccount: ""}
	configureServiceAccountForJob(&job)
	if logFatalCalls != 0 || len(job.Volumes) != 0 {
		t.Errorf("Service Account was not specified, but action was performed")
	}

	badAccounts := []string{
		"/etc/foo/service-account.json/bar",
		"foo/etc/bar/service-account.json",
		"/foo/bar/service-account.json",
		"/etc/foo/some-other-account.json",
	}
	for _, acct := range badAccounts {
		job = baseProwJobTemplateData{ServiceAccount: acct}
		configureServiceAccountForJob(&job)
		if logFatalCalls != 1 {
			t.Errorf("Service account %v did not cause error", acct)
		}
		logFatalCalls = 0
	}

	job = baseProwJobTemplateData{ServiceAccount: "/etc/foo/service-account.json"}
	configureServiceAccountForJob(&job)
	expectedVolumeMounts := []string{
		"- name: foo",
		"  mountPath: /etc/foo",
		"  readOnly: true",
	}
	for i := range expectedVolumeMounts {
		if diff := cmp.Diff(job.VolumeMounts[i], expectedVolumeMounts[i]); diff != "" {
			t.Errorf("Unexpected volume mount: (-got +want)\n%s", diff)
		}
	}
	expectedVolumes := []string{
		"- name: foo",
		"  secret:",
		"    secretName: foo",
	}
	for i := range expectedVolumes {
		if diff := cmp.Diff(job.Volumes[i], expectedVolumes[i]); diff != "" {
			t.Errorf("Unexpected volume: (-got +want)\n%s", diff)
		}
	}
}

func TestAddExtraEnvVarsToJob(t *testing.T) {
	SetupForTesting()
	job := baseProwJobTemplateData{}

	in := []string{"foo=bar"}
	addExtraEnvVarsToJob(in, &job)
	if diff := cmp.Diff(job.Env[0], "- name: foo"); diff != "" {
		t.Errorf("Unexpected env name: (-got +want)\n%s", diff)
	}
	if diff := cmp.Diff(job.Env[1], "  value: bar"); diff != "" {
		t.Errorf("Unexpected env value: (-got +want)\n%s", diff)
	}

	in = []string{"foobar"}
	addExtraEnvVarsToJob(in, &job)
	if logFatalCalls != 1 {
		t.Errorf("Invalid string 'foobar' should have caused error")
	}
}

func TestSetupDockerInDockerForJob(t *testing.T) {
	SetupForTesting()
	job := baseProwJobTemplateData{}
	setupDockerInDockerForJob(&job)
	if len(job.Volumes) == 0 || len(job.VolumeMounts) == 0 {
		t.Errorf("Docker in Docker setup did not create volumes and/or mounts")
	}
	if len(job.Env) == 0 || len(job.SecurityContext) == 0 {
		t.Errorf("Docker in Docker setup did not add env and/or set security context")
	}
}

func TestSetResourcesReqForJob(t *testing.T) {
	SetupForTesting()
	job := baseProwJobTemplateData{}
	requests := yaml.MapSlice{
		yaml.MapItem{Key: "memory", Value: "12Gi"},
		yaml.MapItem{Key: "disk", Value: "12Ti"},
	}
	limits := yaml.MapSlice{
		yaml.MapItem{Key: "memory", Value: "16Gi"},
		yaml.MapItem{Key: "disk", Value: "16Ti"},
	}
	resources := yaml.MapSlice{
		yaml.MapItem{Key: "requests", Value: requests},
		yaml.MapItem{Key: "limits", Value: limits},
	}
	setResourcesReqForJob(resources, &job)
	expectedResources := []string{
		"  requests:",
		"    memory: 12Gi",
		"    disk: 12Ti",
		"  limits:",
		"    memory: 16Gi",
		"    disk: 16Ti",
	}
	for i := range expectedResources {
		if diff := cmp.Diff(job.Resources[i], expectedResources[i]); diff != "" {
			t.Errorf("Unexpected volume mount: (-got +want)\n%s", diff)
		}
	}
}

func TestSetReporterConfigReqForJob(t *testing.T) {
	SetupForTesting()
	job := baseProwJobTemplateData{}
	slack := yaml.MapSlice{
		yaml.MapItem{Key: "channel", Value: "serving-api"},
		yaml.MapItem{Key: "report_template", Value: "Report Template"},
		yaml.MapItem{Key: "foo", Value: []interface{}{"bar", "baz"}},
	}
	resources := yaml.MapSlice{
		yaml.MapItem{Key: "slack", Value: slack},
	}
	setReporterConfigReqForJob(resources, &job)

	expectedConfig := []string{
		"  slack:",
		"    channel: serving-api",
		"    report_template: Report Template",
	}
	for i := range expectedConfig {
		if diff := cmp.Diff(job.ReporterConfig[i], expectedConfig[i]); diff != "" {
			t.Errorf("Unexpected reporter config: (-got +want)\n%s", diff)
		}
	}
	expectedJobStates := "[bar baz]"
	if diff := cmp.Diff(fmt.Sprintf("%v", job.JobStatesToReport), expectedJobStates); diff != "" {
		t.Errorf("Unexpected job states: (-got +want)\n%s", diff)
	}
}

func TestParseBasicJobConfigOverrides(t *testing.T) {
	SetupForTesting()
	requests := yaml.MapSlice{
		yaml.MapItem{Key: "memory", Value: "12Gi"},
		yaml.MapItem{Key: "disk", Value: "12Ti"},
	}
	limits := yaml.MapSlice{
		yaml.MapItem{Key: "memory", Value: "16Gi"},
		yaml.MapItem{Key: "disk", Value: "16Ti"},
	}
	resources := yaml.MapSlice{
		yaml.MapItem{Key: "requests", Value: requests},
		yaml.MapItem{Key: "limits", Value: limits},
	}
	slack := yaml.MapSlice{
		yaml.MapItem{Key: "channel", Value: "serving-api"},
		yaml.MapItem{Key: "report_template", Value: "Report Template"},
		yaml.MapItem{Key: "foo", Value: []interface{}{"bar", "baz"}},
	}
	reporter_config := yaml.MapSlice{
		yaml.MapItem{Key: "slack", Value: slack},
	}

	repo_name := "foo_repo"
	repositories = []repositoryData{
		repositoryData{Name: repo_name, EnablePerformanceTests: false},
	}

	job := baseProwJobTemplateData{RepoBranch: "my_repo_branch", RepoName: repo_name}
	config := yaml.MapSlice{
		yaml.MapItem{Key: "skip_branches", Value: []interface{}{"skip", "branches"}},
		yaml.MapItem{Key: "branches", Value: []interface{}{"branch1", "branch2"}},
		yaml.MapItem{Key: "args", Value: []interface{}{"arg1", "arg2"}},
		yaml.MapItem{Key: "timeout", Value: 42},
		yaml.MapItem{Key: "command", Value: "foo_command"},
		yaml.MapItem{Key: "needs-monitor"},
		yaml.MapItem{Key: "needs-dind", Value: true},
		yaml.MapItem{Key: "always-run", Value: true},
		yaml.MapItem{Key: "performance"},
		yaml.MapItem{Key: "env-vars", Value: []interface{}{"foo=bar"}},
		yaml.MapItem{Key: "optional"},
		yaml.MapItem{Key: "resources", Value: resources},
		yaml.MapItem{Key: "reporter_config", Value: reporter_config},
	}

	parseBasicJobConfigOverrides(&job, config)

	expected := "[  base_ref: my_repo_branch]"
	actual := fmt.Sprintf("%v", job.ExtraRefs)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected base ref: (-got +want)\n%s", diff)
	}
	expected = "[skip branches]"
	actual = fmt.Sprintf("%v", job.SkipBranches)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected skip branches: (-got +want)\n%s", diff)
	}
	expected = "[branch1 branch2]"
	actual = fmt.Sprintf("%v", job.Branches)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected branches: (-got +want)\n%s", diff)
	}
	expected = "[arg1 arg2]"
	actual = fmt.Sprintf("%v", job.Args)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected args: (-got +want)\n%s", diff)
	}
	if job.Timeout != 42 {
		t.Errorf("Unexpected timeout: %v", job.Timeout)
	}
	if diff := cmp.Diff(job.Command, "foo_command"); diff != "" {
		t.Errorf("Unexpected command: (-got +want)\n%s", diff)
	}
	if !job.NeedsMonitor {
		t.Errorf("Expected job.NeedsMonitor to be true")
	}
	if len(job.Volumes) == 0 || len(job.VolumeMounts) == 0 || len(job.SecurityContext) == 0 {
		t.Errorf("Error in Docker in Docker setup")
	}
	if !job.AlwaysRun {
		t.Errorf("Expected job.AlwaysRun to be true")
	}
	if !repositories[0].EnablePerformanceTests {
		t.Errorf("Repository performance test should have been enabled")
	}
	// Note that the first 2 Env variables are from the Docker in Docker setup
	if diff := cmp.Diff(job.Env[2], "- name: foo"); diff != "" {
		t.Errorf("Unexpected env name: (-got +want)\n%s", diff)
	}
	if diff := cmp.Diff(job.Env[3], "  value: bar"); diff != "" {
		t.Errorf("Unexpected env value: (-got +want)\n%s", diff)
	}
	if diff := cmp.Diff(job.Optional, "optional: true"); diff != "" {
		t.Errorf("Unexpected job.Optional value: (-got +want)\n%s", diff)
	}
	expectedResources := []string{
		"  requests:",
		"    memory: 12Gi",
		"    disk: 12Ti",
		"  limits:",
		"    memory: 16Gi",
		"    disk: 16Ti",
	}
	for i := range expectedResources {
		if diff := cmp.Diff(job.Resources[i], expectedResources[i]); diff != "" {
			t.Errorf("Unexpected volume mount: (-got +want)\n%s", diff)
		}
	}
	expectedReporterConfig := []string{
		"  slack:",
		"    channel: serving-api",
		"    report_template: Report Template",
	}
	for i := range expectedReporterConfig {
		if diff := cmp.Diff(job.ReporterConfig[i], expectedReporterConfig[i]); diff != "" {
			t.Errorf("Unexpected reporter config: (-got +want)\n%s", diff)
		}
	}
	expectedJobStates := "[bar baz]"
	if diff := cmp.Diff(fmt.Sprintf("%v", job.JobStatesToReport), expectedJobStates); diff != "" {
		t.Errorf("Unexpected job states: (-got +want)\n%s", diff)
	}

	timeoutOverride = 999
	parseBasicJobConfigOverrides(&job, config)
	if job.Timeout != 999 {
		t.Errorf("Timeout override did not work")
	}
}

func TestGetProwConfigData(t *testing.T) {
	SetupForTesting()
	presubmits := yaml.MapSlice{
		yaml.MapItem{Key: "foo-repo"},
		yaml.MapItem{Key: "bar-repo"},
		yaml.MapItem{Key: "bar-repo-test-infra"},
		yaml.MapItem{Key: "dup-repo"},
		yaml.MapItem{Key: "dup-repo"},
	}
	config := yaml.MapSlice{
		yaml.MapItem{Key: "presubmits", Value: presubmits},
		yaml.MapItem{Key: "ignored-section"},
	}

	out := getProwConfigData(config)

	expectedRepos := "[bar-repo bar-repo-test-infra dup-repo foo-repo]"
	if diff := cmp.Diff(fmt.Sprintf("%v", out.TideRepos), expectedRepos); diff != "" {
		t.Errorf("Unexpected TideRepos: (-got +want)\n%s", diff)
	}
	if diff := cmp.Diff(out.TestInfraRepo, "bar-repo-test-infra"); diff != "" {
		t.Errorf("Unexpected test-infra repo: (-got +want)\n%s", diff)
	}
}
func TestParseSection(t *testing.T) {
	SetupForTesting()
	generated := []string{}
	generate := func(a, b string, s yaml.MapSlice) {
		for _, v := range s {
			generated = append(generated, fmt.Sprintf("%v, %v, %v, %v", a, b, v.Key, v.Value))
		}
	}
	finalized := []string{}
	finalize := func(a, b string, s yaml.MapSlice) {
		finalized = append(finalized, fmt.Sprintf("%v, %v", a, b))
	}
	title := "pet-store"
	dogs := []interface{}{
		yaml.MapSlice{
			yaml.MapItem{Key: "Spot", Value: "Dalmation"},
			yaml.MapItem{Key: "Fido", Value: "Terrier"},
		},
		yaml.MapSlice{
			yaml.MapItem{Key: "Remy", Value: "Retriever"},
		},
	}
	cats := []interface{}{
		yaml.MapSlice{
			yaml.MapItem{Key: "Whiskers", Value: "Calico"},
			yaml.MapItem{Key: "Twitch", Value: "Siamese"},
		},
	}
	config := yaml.MapSlice{
		yaml.MapItem{Key: "pet-store", Value: yaml.MapSlice{
			yaml.MapItem{Key: "dogs", Value: dogs},
			yaml.MapItem{Key: "cats", Value: cats},
		}},
		yaml.MapItem{Key: "toy-store"},
	}
	parseSection(config, title, generate, finalize)

	expected := []string{
		"pet-store, dogs, Spot, Dalmation",
		"pet-store, dogs, Fido, Terrier",
		"pet-store, dogs, Remy, Retriever",
		"pet-store, cats, Whiskers, Calico",
		"pet-store, cats, Twitch, Siamese",
	}
	for i := range expected {
		if diff := cmp.Diff(generated[i], expected[i]); diff != "" {
			t.Errorf("Unexpected generated output: (-got +want)\n%s", diff)
		}
	}
	expected = []string{
		"pet-store, dogs",
		"pet-store, cats",
	}
	for i := range expected {
		if diff := cmp.Diff(finalized[i], expected[i]); diff != "" {
			t.Errorf("Unexpected finalized output: (-got +want)\n%s", diff)
		}
	}
}

func TestGitHubRepo(t *testing.T) {
	SetupForTesting()
	repositoryOverride = ""
	in := baseProwJobTemplateData{RepoURI: "repoURI"}

	if diff := cmp.Diff(gitHubRepo(in), "repoURI"); diff != "" {
		t.Errorf("Bad output when RepoBranch unset and no override: (-got +want)\n%s", diff)
	}

	in = baseProwJobTemplateData{RepoURI: "repoURI", RepoBranch: "repoBranch"}
	if diff := cmp.Diff(gitHubRepo(in), "repoURI=repoBranch"); diff != "" {
		t.Errorf("Bad output when RepoBranch set and no override: (-got +want)\n%s", diff)
	}

	repositoryOverride = "repoOverride"
	if diff := cmp.Diff(gitHubRepo(in), "repoOverride"); diff != "" {
		t.Errorf("Bad output when override set: (-got +want)\n%s", diff)
	}
}

func TestExecuteJobTemplate(t *testing.T) {
	SetupForTesting()
	name := "foo"
	templ := `
- foo: [[.Foo]]
[[indent_section 2 "bar" .Bar]]
`
	title := "my-title"
	repoName := "my-repo-name"
	jobName := "my-job-name"
	groupByRepo := false
	data := struct {
		Foo string
		Bar []string
	}{
		Foo: "Foo",
		Bar: []string{"Bar", "Baz"},
	}

	jobNameFilter = "xyz"
	executeJobTemplate(name, templ, title, repoName, jobName, groupByRepo, data)
	if logFatalCalls != 0 {
		t.Errorf("Fatal log call recorded")
	}
	expected := ""
	if diff := cmp.Diff(GetOutput(), expected); diff != "" {
		t.Errorf("Expected job to be filtered: (-got +want)\n%s", diff)
	}

	ResetOutput()
	jobNameFilter = "my-job-name"
	executeJobTemplate(name, templ, title, repoName, jobName, groupByRepo, data)
	if logFatalCalls != 0 {
		t.Errorf("Fatal log call recorded")
	}
	if GetOutput() == "" {
		t.Errorf("Job should not have been filtered")
	}

	ResetOutput()
	jobNameFilter = ""
	sectionMap[title] = false
	executeJobTemplate(name, templ, title, repoName, jobName, groupByRepo, data)
	if logFatalCalls != 0 {
		t.Errorf("Fatal log call recorded")
	}
	expected = "my-title:\n- foo: Foo\nbar:\n  \"Bar\"\n  \"Baz\"\n"
	if diff := cmp.Diff(GetOutput(), expected); diff != "" {
		t.Errorf("Bad execute job template output: (-got +want)\n%s", diff)
	}

	ResetOutput()
	sectionMap[title] = true
	executeJobTemplate(name, templ, title, repoName, jobName, groupByRepo, data)
	if logFatalCalls != 0 {
		t.Errorf("Fatal log call recorded")
	}
	expected = "- foo: Foo\nbar:\n  \"Bar\"\n  \"Baz\"\n"
	if diff := cmp.Diff(GetOutput(), expected); diff != "" {
		t.Errorf("Bad execute job template output: (-got +want)\n%s", diff)
	}

	ResetOutput()
	groupByRepo = true
	sectionMap[title+repoName] = false
	executeJobTemplate(name, templ, title, repoName, jobName, groupByRepo, data)
	if logFatalCalls != 0 {
		t.Errorf("Fatal log call recorded")
	}
	expected = "  my-repo-name:\n- foo: Foo\nbar:\n  \"Bar\"\n  \"Baz\"\n"
	if diff := cmp.Diff(GetOutput(), expected); diff != "" {
		t.Errorf("Bad execute job template output: (-got +want)\n%s", diff)
	}
}

func TestExecuteTemplate(t *testing.T) {
	SetupForTesting()
	name := "foo"
	templ := `
- foo: [[.Foo]]
[[indent_section 2 "bar" .Bar]]
`
	data := struct {
		Foo string
		Bar []string
	}{
		Foo: "Foo",
		Bar: []string{"Bar", "Baz"},
	}
	executeTemplate(name, templ, data)

	if logFatalCalls != 0 {
		t.Errorf("Fatal log call recorded")
	}
	expected :=
		"- foo: Foo\nbar:\n  \"Bar\"\n  \"Baz\"\n"

	if diff := cmp.Diff(GetOutput(), expected); diff != "" {
		t.Errorf("Bad execute template output: (-got +want)\n%s", diff)
	}
}
func TestStringArrayFlagString(t *testing.T) {
	SetupForTesting()
	arr := stringArrayFlag{"a", "b", "c"}
	if diff := cmp.Diff(arr.String(), "a, b, c"); diff != "" {
		t.Errorf("(-got +want)\n%s", diff)
	}
}
func TestStringArrayFlagSet(t *testing.T) {
	SetupForTesting()
	arr := stringArrayFlag{"a", "b", "c"}
	arr.Set("d")
	if diff := cmp.Diff(arr.String(), "a, b, c, d"); diff != "" {
		t.Errorf("(-got +want)\n%s", diff)
	}
}

func TestParseJob(t *testing.T) {
	SetupForTesting()
	dogs := yaml.MapSlice{
		yaml.MapItem{Key: "Spot", Value: "Dalmation"},
		yaml.MapItem{Key: "Fido", Value: "Terrier"},
	}
	cats := yaml.MapSlice{
		yaml.MapItem{Key: "Fluffy", Value: "Calico"},
		yaml.MapItem{Key: "Maxine", Value: "Siamese"},
	}
	pets := yaml.MapSlice{
		yaml.MapItem{Key: "dogs", Value: dogs},
		yaml.MapItem{Key: "cats", Value: cats},
	}

	out := parseJob(pets, "dogs")
	expected := "[{Spot Dalmation} {Fido Terrier}]"
	if diff := cmp.Diff(fmt.Sprintf("%v", out), expected); diff != "" {
		t.Errorf("ParseJob did not return expected slice. (-got +want)\n%s", diff)
	}

	out = parseJob(pets, "hamsters")
	if logFatalCalls != 1 {
		t.Errorf("ParseJob did not return error as expected.")
	}
}

func TestParseGoCoverageMap(t *testing.T) {
	SetupForTesting()
	dogs := []interface{}{
		yaml.MapSlice{
			yaml.MapItem{Key: "Spot", Value: "Dalmation"},
			yaml.MapItem{Key: "Fido", Value: "Terrier"},
		},
		yaml.MapSlice{
			yaml.MapItem{Key: "go-coverage", Value: true},
		},
	}
	cats := []interface{}{
		yaml.MapSlice{
			yaml.MapItem{Key: "Whiskers", Value: "Calico"},
			yaml.MapItem{Key: "Twitch", Value: "Siamese"},
		},
	}
	config := yaml.MapSlice{
		yaml.MapItem{Key: "pets/dog-repo", Value: dogs},
		yaml.MapItem{Key: "pets/cat-repo", Value: cats},
	}

	out := parseGoCoverageMap(config)
	if out["cat-repo"] {
		t.Errorf("Go coverage should not have been enabled for cat-repo")
	}
	if !out["dog-repo"] {
		t.Errorf("Go coverage should have been enabled for dog-repo")
	}
}

func TestCollectMetaData(t *testing.T) {
	redDetailMap := JobDetailMap{
		"red-repo": []string{"red-a", "red-b"},
	}

	metaData = TestGridMetaData{
		md: map[string]JobDetailMap{
			"red-proj": redDetailMap,
		},
		projNames: []string{"red-proj"},
	}
	redRepo := []interface{}{
		yaml.MapSlice{
			yaml.MapItem{Key: "continuous", Value: true},
			yaml.MapItem{Key: "dot-release", Value: true},
			yaml.MapItem{Key: "auto-release", Value: false},
			yaml.MapItem{Key: "nightly", Value: false},
			yaml.MapItem{Key: "webhook-apicoverage", Value: false},
		},
		yaml.MapSlice{
			yaml.MapItem{Key: "branch-ci", Value: true},
		},
	}
	bluRepo := []interface{}{
		yaml.MapSlice{
			yaml.MapItem{Key: "release", Value: "0.1.2"},
			yaml.MapItem{Key: "custom-job", Value: "custom-job-name"},
			yaml.MapItem{Key: "ignore-me", Value: "ignore-me-too"},
		},
	}
	config := yaml.MapSlice{
		yaml.MapItem{Key: "red-proj/red-repo", Value: redRepo},
		yaml.MapItem{Key: "blu-proj/blu-repo", Value: bluRepo},
	}

	collectMetaData(config)

	expected := "[red-a red-b dot-release continuous]"
	actual := fmt.Sprintf("%v", metaData.md["red-proj"]["red-repo"])
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected metadata for red proj/repo. (-got +want)\n%s", diff)
	}

	expected = "[custom-job-name]"
	actual = fmt.Sprintf("%v", metaData.md["blu-proj-0.1.2"]["blu-repo"])
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected metadata for blu proj/repo. (-got +want)\n%s", diff)
	}

	expected = "[red-proj blu-proj blu-proj-0.1.2]"
	actual = fmt.Sprintf("%v", metaData.projNames)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected list of project names. (-got +want)\n%s", diff)
	}
}

func TestUpdateTestCoverageJobDataIfNeeded(t *testing.T) {
	SetupForTesting()
	repoName := "foo-repo"
	goCoverageMap = map[string]bool{repoName: true}
	jobDetailMap := JobDetailMap{
		"bar-repo": []string{"bar-a", "bar-b"},
	}
	updateTestCoverageJobDataIfNeeded(jobDetailMap, repoName)
	if len(goCoverageMap) != 0 {
		t.Errorf("foo-repo was not deleted from goCoverageMap")
	}
	expected := "[test-coverage]"
	actual := fmt.Sprintf("%v", jobDetailMap[repoName])
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected entry for repoName in job detail map (-got +want)\n%s", diff)
	}
}

func TestAddRemainingTestCoverageJobs(t *testing.T) {
	SetupForTesting()
	goCoverageMap = map[string]bool{
		"bar-repo": true,
		"baz-repo": false}
	jobDetailMap := JobDetailMap{
		"foo-repo": []string{"foo-a", "foo-b"},
	}
	metaData = TestGridMetaData{
		md:        map[string]JobDetailMap{"proj0": jobDetailMap},
		projNames: []string{"proj0"},
	}

	addRemainingTestCoverageJobs()

	expected := "[test-coverage]"
	actual := fmt.Sprintf("%v", jobDetailMap["bar-repo"])
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected entry for bar-repo in job detail map (-got +want)\n%s", diff)
	}
}
func TestBuildProjRepoStr(t *testing.T) {
	SetupForTesting()

	projName := "project-name"
	repoName := "repo-name"
	expected := "project-name-repo-name"
	actual := buildProjRepoStr(projName, repoName)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected project repo string: (-got +want)\n%s", diff)
	}

	projName = "knative-sandbox-0.15"
	repoName = "repo-name"
	expected = "knative-sandbox-repo-name-0.15"
	actual = buildProjRepoStr(projName, repoName)
	if diff := cmp.Diff(actual, expected); diff != "" {
		t.Errorf("Unexpected project repo string: (-got +want)\n%s", diff)
	}
}
func TestIsReleased(t *testing.T) {
	SetupForTesting()
	valid := []string{"abc-0", "def-1.2.3"}
	invalid := []string{"-4.5.6", "abc-1.2.3g"}
	for _, v := range valid {
		if !isReleased(v) {
			t.Errorf("Should be valid: %v", v)
		}
	}
	for _, v := range invalid {
		if isReleased(v) {
			t.Errorf("Should be invalid: %v", v)
		}
	}
}

func TestSetOutput(t *testing.T) {
	SetupForTesting()
	setOutput("")
	if logFatalCalls != 0 {
		t.Errorf("Fatal log call recorded")
	}
	// don't test setting an output file since this will create
	// a local file system change
}