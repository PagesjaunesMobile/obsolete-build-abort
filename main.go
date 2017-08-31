package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type AbortQuery struct {
	AbortReason string `json:"abort_reason"`
}

type Data struct {
	TriggeredAt                  time.Time `json:"triggered_at"`
	StartedOnWorkerAt            time.Time `json:"started_on_worker_at"`
	EnvironmentPrepareFinishedAt time.Time `json:"environment_prepare_finished_at"`
	FinishedAt                   string    `json:"finished_at"`
	Slug                         string    `json:"slug"`
	Status                       int       `json:"status"`
	StatusText                   string    `json:"status_text"`
	AbortReason                  string    `json:"abort_reason"`
	IsOnHold                     bool      `json:"is_on_hold"`
	Branch                       string    `json:"branch"`
	BuildNumber                  int       `json:"build_number"`
	CommitHash                   string    `json:"commit_hash"`
	CommitMessage                string    `json:"commit_message"`
	Tag                          string    `json:"tag"`
	TriggeredWorkflow            string    `json:"triggered_workflow"`
	TriggeredBy                  string    `json:"triggered_by"`
	StackConfigType              string    `json:"stack_config_type"`
	StackIdentifier              string    `json:"stack_identifier"`
	OriginalBuildParams          struct {
		Branch string `json:"branch"`
	} `json:"original_build_params"`
	PullRequestID           int    `json:"pull_request_id"`
	PullRequestTargetBranch string `json:"pull_request_target_branch"`
	PullRequestViewURL      string `json:"pull_request_view_url"`
	CommitViewURL           string `json:"commit_view_url"`
}

type Builds struct {
	Tasks []Data `json:"data"`
	// Paging struct {
	//   TotalItemCount int    `json:"total_item_count"`
	//   PageItemLimit  int    `json:"page_item_limit"`
	//   Next           string `json:"next"`
	// } `json:"paging"`
}

func Filter(vs []Data, f func(Data) bool) []Data {
	vsf := make([]Data, 0)

	for _, v := range vs {

		if f(v) {
			vsf = append(vsf, v)
		}
	}
	return vsf
}
func Fail(step string, err error) {
	if err != nil {
		fmt.Printf("%s: %s", step, err)
		os.Exit(1)
	}
}

func main() {
	token := os.Getenv("token")
	appSlug := os.Getenv("BITRISE_APP_SLUG")
	currentSlug := os.Getenv("BITRISE_BUILD_SLUG")
	currentTriggered := time.Unix(os.Getenv("BITRISE_BUILD_TRIGGER_TIMESTAMP"), 0)
	sourceBranch := os.Getenv("BITRISE_GIT_BRANCH")
	workflowID := os.Getenv("BITRISE_TRIGGERED_WORKFLOW_ID")

	url := fmt.Sprintf("https://api.bitrise.io/v0.1/apps/%s/builds?limit=10", appSlug)
	// Build the request
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", token)
	Fail("NewRequest: ", err)

	// For control over HTTP client headers,
	// redirect policy, and other settings,
	// create a Client
	// A Client is an HTTP client
	client := &http.Client{}

	// Send the request via a client
	// Do sends an HTTP request and
	// returns an HTTP response
	resp, err := client.Do(req)
	Fail("Do: ", err)

	// Callers should close resp.Body
	// when done reading from it
	// Defer the closing of the body
	defer resp.Body.Close()

	// Fill the record with the data from the JSON
	var record Builds
	// Use json.Decode for reading streams of JSON data
	errJson := json.NewDecoder(resp.Body).Decode(&record)
	Fail("Json: ", errJson)

	var otherTasks = Filter(record.Tasks, func(v Data) bool {
		// verif si job different de celui ci
		return v.Slug != currentSlug
	})
	var runningTasks = Filter(otherTasks, func(v Data) bool {
		// verif si job different de celui ci
		return v.Status == 0 && v.TriggeredAt.Before(currentTriggered)
	})

	json, _ := json.Marshal(AbortQuery{AbortReason: "obsolete"})

	for _, v := range runningTasks {
		if (v.Branch == sourceBranch) && (v.TriggeredWorkflow == workflowID) {
			fmt.Printf("Triggered at", v.Slug, currentTriggered)
			fmt.Println()
			fmt.Printf("Other running task  = %s, triggered at", v.Slug, v.TriggeredAt)
			fmt.Println()
			fmt.Printf("Status  = %s (%s)", v.Status, v.StatusText)
			fmt.Println()
			abort_url := fmt.Sprintf("https://api.bitrise.io/v0.1/apps/%s/builds/%s/abort", appSlug, v.Slug)
			req, err = http.NewRequest("POST", abort_url, bytes.NewBuffer(json))
			req.Header.Add("Authorization", token)
			req.Header.Set("Content-Type", "application/json")
			Fail("Request: ", err)

			resp, err := client.Do(req)
			Fail("Abort: ", err)

			defer resp.Body.Close()
		}
	}

	//
	// --- Exit codes:
	// The exit code of your Step is very important. If you return
	//  with a 0 exit code `bitrise` will register your Step as "successful".
	// Any non zero exit code will be registered as "failed" by `bitrise`.
	os.Exit(0)
}
