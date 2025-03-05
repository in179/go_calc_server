package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Task struct {
	ID            int     `json:"id"`
	Arg1          float64 `json:"arg1"`
	Arg2          float64 `json:"arg2"`
	Operator      string  `json:"operation"`
	OperationTime int     `json:"operation_time"`
}

type TaskResponse struct {
	Task *Task `json:"task"`
}

type TaskResultRequest struct {
	ID     int     `json:"id"`
	Result float64 `json:"result"`
}

func computeOperation(op string, a, b float64) (float64, error) {
	switch op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		if b == 0 {
			return 0, fmt.Errorf("division by zero")
		}
		return a / b, nil
	default:
		return 0, fmt.Errorf("unknown operator")
	}
}

func worker(wg *sync.WaitGroup, client *http.Client, agentID int) {
	defer wg.Done()
	for {
		resp, err := client.Get("http://localhost:8080/internal/task")
		if err != nil {
			log.Printf("Agent %d: error getting task: %v", agentID, err)
			time.Sleep(2 * time.Second)
			continue
		}
		if resp.StatusCode == http.StatusNotFound {
			time.Sleep(2 * time.Second)
			continue
		}
		var taskResp TaskResponse
		err = json.NewDecoder(resp.Body).Decode(&taskResp)
		resp.Body.Close()
		if err != nil || taskResp.Task == nil {
			log.Printf("Agent %d: invalid task response", agentID)
			time.Sleep(2 * time.Second)
			continue
		}
		task := taskResp.Task
		log.Printf("Agent %d: received task %d: %f %s %f, delay %d ms", agentID, task.ID, task.Arg1, task.Operator, task.Arg2, task.OperationTime)
		time.Sleep(time.Duration(task.OperationTime) * time.Millisecond)
		result, err := computeOperation(task.Operator, task.Arg1, task.Arg2)
		if err != nil {
			log.Printf("Agent %d: error computing task %d: %v", agentID, task.ID, err)
			continue
		}
		reqBody, _ := json.Marshal(TaskResultRequest{
			ID:     task.ID,
			Result: result,
		})
		postResp, err := client.Post("http://localhost:8080/internal/task", "application/json", bytes.NewBuffer(reqBody))
		if err != nil {
			log.Printf("Agent %d: error posting result for task %d: %v", agentID, task.ID, err)
			continue
		}
		body, _ := ioutil.ReadAll(postResp.Body)
		postResp.Body.Close()
		if postResp.StatusCode != http.StatusOK {
			log.Printf("Agent %d: failed to post result for task %d: %s", agentID, task.ID, string(body))
		} else {
			log.Printf("Agent %d: successfully completed task %d with result %f", agentID, task.ID, result)
		}
	}
}

func main() {
	cpStr := os.Getenv("COMPUTING_POWER")
	cp, err := strconv.Atoi(cpStr)
	if err != nil || cp <= 0 {
		cp = 2
	}
	log.Printf("Starting agent with computing power: %d", cp)
	var wg sync.WaitGroup
	client := &http.Client{Timeout: 10 * time.Second}
	for i := 1; i <= cp; i++ {
		wg.Add(1)
		go worker(&wg, client, i)
	}
	wg.Wait()
}
