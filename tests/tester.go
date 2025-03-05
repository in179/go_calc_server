package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const baseURL = "http://localhost:8080/api/v1"

type ExpressionRequest struct {
	Expression string `json:"expression"`
}

type ExpressionResponse struct {
	ID int `json:"id"`
}

type TaskResponse struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Result string `json:"result"`
}

type ResultSubmission struct {
	ID     int    `json:"id"`
	Result string `json:"result"`
}

type TestCase struct {
	Expression    string
	ExpectedCode  int
	ExpectedValue string
}

func sendRequest(method, url string, body any) (*http.Response, error) {
	var requestBody io.Reader
	if body != nil {
		jsonBody, _ := json.Marshal(body)
		requestBody = bytes.NewBuffer(jsonBody)
	}
	req, err := http.NewRequest(method, url, requestBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 5 * time.Second}
	return client.Do(req)
}

func testExpression(tc TestCase) {
	url := baseURL + "/calculate"
	resp, err := sendRequest("POST", url, ExpressionRequest{Expression: tc.Expression})
	if err != nil {
		fmt.Println("Ошибка запроса:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != tc.ExpectedCode {
		fmt.Printf("Ошибка: выражение \"%s\" ожидало код %d, получено %d\n", tc.Expression, tc.ExpectedCode, resp.StatusCode)
		return
	}

	if resp.StatusCode == 201 {
		var result ExpressionResponse
		json.NewDecoder(resp.Body).Decode(&result)
		fmt.Printf("Выражение \"%s\" принято, ID: %d\n", tc.Expression, result.ID)
		testGetExpressionByID(result.ID, tc.ExpectedValue)
		testSubmitResult(result.ID, tc.ExpectedValue)
	} else {
		fmt.Printf("Выражение \"%s\" вернуло код %d, тест успешен\n", tc.Expression, resp.StatusCode)
	}
}

func testGetExpressionByID(id int, expectedValue string) {
	url := fmt.Sprintf("%s/expressions/%d", baseURL, id)
	time.Sleep(3 * time.Second)

	resp, err := sendRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Ошибка при получении выражения:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Println("Ошибка при получении выражения, код:", resp.StatusCode)
		return
	}

	var result TaskResponse
	json.NewDecoder(resp.Body).Decode(&result)

	if result.Result == expectedValue {
		fmt.Printf("Выражение с ID %d вычислено корректно: %s\n", id, result.Result)
	} else {
		fmt.Printf("Ошибка: ожидалось %s, получено %s\n", expectedValue, result.Result)
	}
}

func testSubmitResult(id int, expectedValue string) {
	url := fmt.Sprintf("%s/internal/task", baseURL)

	resp, err := sendRequest("POST", url, ResultSubmission{ID: id, Result: expectedValue})
	if err != nil {
		fmt.Println("Ошибка при отправке результата:", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Printf("Ошибка при отправке результата, код: %d\n", resp.StatusCode)
	} else {
		fmt.Printf("Результат для ID %d успешно записан.\n", id)
	}
}

func main() {
	expressions := []TestCase{
		{"2+2", 201, "4"},
		{"10-5", 201, "5"},
		{"3*4", 201, "12"},
		{"8/2", 201, "4"},
		{"5+5*2", 201, "15"},
		{"(4+4)*2", 201, "16"},
		{"100-50+25", 201, "75"},
		{"(10+2)*(8/4)", 201, "24"},
		{"3+2*5-1", 201, "12"},
		{"(7-2)*(8/4)", 201, "10"},
		{"20/5+3*2", 201, "10"},
		{"0/1", 201, "0"},
		{"1-1", 201, "0"},
		{"7+3*6", 201, "25"},
		{"9-2/2", 201, "8"},
		{"4*5+3", 201, "23"},
		{"(6+4)/2", 201, "5"},
		{"3+3*3-3", 201, "9"},
		{"15/(3+2)", 201, "3"},
		{"4*3-2+8", 201, "18"},
	}

	invalidExpressions := []TestCase{
		{"10/0", 422, ""},
		{"invalid", 422, ""},
		{"", 422, ""},
		{"3**3", 422, ""},
		{"5//2", 422, ""},
	}

	for _, tc := range expressions {
		testExpression(tc)
	}

	for _, tc := range invalidExpressions {
		testExpression(tc)
	}

	fmt.Println("Тестирование завершено.")
}
