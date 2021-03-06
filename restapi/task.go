package restapi

import (
	"net/http"
	"time"

	"github.com/ant0ine/go-json-rest/rest"
	"github.com/sebest/hooky/models"
	"gopkg.in/mgo.v2/bson"
)

// Task is used for the Rest API.
type Task struct {
	// ID is the Task ID.
	ID string `json:"id"`

	// Created is the date when the Task was created.
	Created string `json:"created"`

	// Account is the ID of the Account owning the Task.
	Account string `json:"account"`

	// Application is the name of the parent Application.
	Application string `json:"application"`

	// Name is the task's name.
	Name string `json:"name"`

	// Queue is the name of the parent Queue.
	Queue string `json:"queue"`

	// URL is the URL that the worker with requests.
	URL string `json:"url"`

	// HTTPAuth is the HTTP authentication to use if any.
	HTTPAuth models.HTTPAuth `json:"auth"`

	// Method is the HTTP method that will be used to execute the request.
	Method string `json:"method"`

	// Headers are the HTTP headers that will be used schedule executing the request.
	Headers map[string]string `json:"headers,omitempty"`

	// Payload is arbitrary data that will be POSTed on the URL.
	Payload string `json:"payload,omitempty"`

	// Schedule is a cron specification describing the recurrency if any.
	Schedule string `json:"schedule,omitempty"`

	// At is a date representing the next time a attempt will be executed.
	At string `json:"at,omitempty"`

	// Status is either `pending`, `retrying`, `canceled`, `success` or `error`
	Status string `json:"status"`

	// Executed is the date of the last time a attempt was executed.
	Executed string `json:"executed,omitempty"`

	// Active is the task active.
	Active *bool `json:"active"`

	// Errors counts the number of attempts that failed.
	Errors int `json:"errors"`

	// LastError is the date of the last attempt in error status.
	LastError string `json:"lastError,omitempty"`

	// LastSuccess is the date of the last attempt in success status.
	LastSuccess string `json:"lastSuccess,omitempty"`

	// Executions counts the number of attempts that were executed.
	Executions int `json:"executions"`

	// ErrorRate is the rate of errors in percent.
	ErrorRate int `json:"errorRate"`

	// Retry is the retry strategy parameters in case of errors.
	Retry models.Retry `json:"retry"`
}

// NewTaskFromModel returns a Task object for use with the Rest API
// from a Task model.
func NewTaskFromModel(task *models.Task) *Task {
	return &Task{
		ID:          task.ID.Hex(),
		Created:     task.ID.Time().UTC().Format(time.RFC3339),
		Application: task.Application,
		Account:     task.Account.Hex(),
		Queue:       task.Queue,
		Name:        task.Name,
		URL:         task.URL,
		Method:      task.Method,
		HTTPAuth:    task.HTTPAuth,
		Headers:     task.Headers,
		Payload:     task.Payload,
		Schedule:    task.Schedule,
		At:          UnixToRFC3339(int64(task.At / 1000000000)),
		Status:      task.Status,
		Executed:    UnixToRFC3339(task.Executed),
		Active:      &task.Active,
		Executions:  task.Executions,
		Errors:      task.Errors,
		LastSuccess: UnixToRFC3339(task.LastSuccess),
		LastError:   UnixToRFC3339(task.LastError),
		ErrorRate:   task.ErrorRate(),
		Retry:       task.Retry,
	}
}

func taskParams(r *rest.Request) (bson.ObjectId, string, string, error) {
	accountID, err := PathAccountID(r)
	if err != nil {
		return accountID, "", "", err
	}
	// TODO handle errors
	applicationName := r.PathParam("application")
	if applicationName == "" {
		applicationName = "default"
	}
	taskName := r.PathParam("task")
	return accountID, applicationName, taskName, nil
}

// PutTask ...
func PutTask(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, taskName, err := taskParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rt := &Task{}
	if err := r.DecodeJsonPayload(rt); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var active bool
	if rt.Active == nil {
		active = true
	} else {
		active = *rt.Active
	}
	b := GetBase(r)
	task, err := b.NewTask(accountID, applicationName, taskName, rt.Queue, rt.URL, rt.HTTPAuth, rt.Method, rt.Headers, rt.Payload, rt.Schedule, rt.Retry, active)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteJson(NewTaskFromModel(task))
}

// GetTask ...
func GetTask(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, taskName, err := taskParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	task, err := b.GetTask(accountID, applicationName, taskName)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if task == nil {
		rest.NotFound(w, r)
		return
	}
	w.WriteJson(NewTaskFromModel(task))
}

// DeleteTask ...
func DeleteTask(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, taskName, err := taskParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	err = b.DeleteTask(accountID, applicationName, taskName)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// DeleteTasks ...
func DeleteTasks(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, _, err := taskParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	err = b.DeleteTasks(accountID, applicationName)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetTasks ...
func GetTasks(w rest.ResponseWriter, r *rest.Request) {
	accountID, applicationName, _, err := taskParams(r)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	b := GetBase(r)
	lp := parseListQuery(r)
	var tasks []*models.Task
	lr := &models.ListResult{
		List: &tasks,
	}

	if err := b.GetTasks(accountID, applicationName, lp, lr); err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if lr.Count == 0 {
		rest.NotFound(w, r)
		return
	}
	rt := make([]*Task, len(tasks))
	for idx, task := range tasks {
		rt[idx] = NewTaskFromModel(task)
	}
	w.WriteJson(models.ListResult{
		List:    rt,
		HasMore: lr.HasMore,
		Total:   lr.Total,
		Count:   lr.Count,
		Page:    lr.Page,
		Pages:   lr.Pages,
	})
}
