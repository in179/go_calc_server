package v1

import (
	"calculator/internal/orchestrator"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

type ExpressionRequest struct {
	Expression string `json:"expression"`
}

type ExpressionResponse struct {
	ID int `json:"id"`
}

func SubmitExpression(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method Not Allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	var req ExpressionRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil || strings.TrimSpace(req.Expression) == "" {
		http.Error(w, `{"error": "Invalid expression"}`, http.StatusUnprocessableEntity)
		return
	}
	id, err := orchestrator.AddExpression(req.Expression)
	if err != nil {
		http.Error(w, `{"error": "Failed to add expression: `+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ExpressionResponse{ID: id})
}

type ExpressionInfo struct {
	ID     int     `json:"id"`
	Status string  `json:"status"`
	Result float64 `json:"result"`
}

func ListExpressions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method Not Allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	exprs := orchestrator.GetExpressions()
	infos := make([]ExpressionInfo, 0)
	for _, e := range exprs {
		infos = append(infos, ExpressionInfo{
			ID:     e.ID,
			Status: e.Status,
			Result: e.Result,
		})
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"expressions": infos})
}

func GetExpression(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method Not Allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		http.Error(w, `{"error": "Invalid URL"}`, http.StatusBadRequest)
		return
	}
	idStr := parts[len(parts)-1]
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, `{"error": "Invalid expression ID"}`, http.StatusBadRequest)
		return
	}
	expr, ok := orchestrator.GetExpression(id)
	if !ok {
		http.Error(w, `{"error": "Expression not found"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"expression": expr})
}

type TaskResponse struct {
	Task *orchestrator.Task `json:"task"`
}

func GetTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error": "Method Not Allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	task, err := orchestrator.GetReadyTask()
	if err != nil {
		http.Error(w, `{"error": "No task available"}`, http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(TaskResponse{Task: task})
}

type TaskResultRequest struct {
	ID     int     `json:"id"`
	Result float64 `json:"result"`
}

func PostTaskResult(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error": "Method Not Allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
		return
	}
	var req TaskResultRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusUnprocessableEntity)
		return
	}
	err = orchestrator.CompleteTask(req.ID, req.Result)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			http.Error(w, `{"error": "Task not found"}`, http.StatusNotFound)
		} else {
			http.Error(w, `{"error": "Failed to complete task"}`, http.StatusUnprocessableEntity)
		}
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
