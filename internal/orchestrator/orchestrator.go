package orchestrator

import (
	"errors"
	"math"
	"strconv"
	"strings"
	"sync"
)

var (
	exprMutex   sync.Mutex
	expressions = make(map[int]*Expression)
	tasks       = make(map[int]*Task)
	readyTasks  = make([]*Task, 0)
	exprCounter = 0
	taskCounter = 0
)

type Expression struct {
	ID         int     `json:"id"`
	Raw        string  `json:"expression"`
	Status     string  `json:"status"`
	Result     float64 `json:"result"`
	TaskIDs    []int
	RootTaskID int
}

type Task struct {
	ID              int     `json:"id"`
	ExpressionID    int     `json:"expression_id"`
	Operator        string  `json:"operation"`
	Arg1            float64 `json:"arg1"`
	Arg2            float64 `json:"arg2"`
	Result          float64 `json:"result"`
	Status          string  `json:"status"`
	LeftDependency  int
	RightDependency int
	OperationTime   int `json:"operation_time"`
}

type Node struct {
	Op    string
	Value float64
	Left  *Node
	Right *Node
}

type parser struct {
	input string
	pos   int
}

func (p *parser) peek() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	return p.input[p.pos]
}

func (p *parser) next() byte {
	if p.pos >= len(p.input) {
		return 0
	}
	ch := p.input[p.pos]
	p.pos++
	return ch
}

func (p *parser) skipWhitespace() {
	for p.pos < len(p.input) && (p.input[p.pos] == ' ' || p.input[p.pos] == '\t') {
		p.pos++
	}
}

func parseExpression(input string) (*Node, error) {
	p := &parser{input: strings.TrimSpace(input), pos: 0}
	p.skipWhitespace()
	node, err := parseExpr(p)
	if err != nil {
		return nil, err
	}
	p.skipWhitespace()
	if p.pos != len(p.input) {
		return nil, errors.New("CharactersError")
	}
	return node, nil
}

func parseExpr(p *parser) (*Node, error) {
	node, err := parseTerm(p)
	if err != nil {
		return nil, err
	}
	for {
		p.skipWhitespace()
		ch := p.peek()
		if ch == '+' || ch == '-' {
			op := string(p.next())
			right, err := parseTerm(p)
			if err != nil {
				return nil, err
			}
			node = &Node{
				Op:    op,
				Left:  node,
				Right: right,
			}
		} else {
			break
		}
	}
	return node, nil
}

func parseTerm(p *parser) (*Node, error) {
	node, err := parseFactor(p)
	if err != nil {
		return nil, err
	}
	for {
		p.skipWhitespace()
		ch := p.peek()
		if ch == '*' || ch == '/' {
			op := string(p.next())
			right, err := parseFactor(p)
			if err != nil {
				return nil, err
			}
			node = &Node{
				Op:    op,
				Left:  node,
				Right: right,
			}
		} else {
			break
		}
	}
	return node, nil
}

func parseFactor(p *parser) (*Node, error) {
	p.skipWhitespace()
	ch := p.peek()
	if ch >= '0' && ch <= '9' {
		p.next()
		if p.pos < len(p.input) {
			nextCh := p.input[p.pos]
			if nextCh >= '0' && nextCh <= '9' {
				return nil, errors.New("LenNumError")
			}
		}
		value, _ := strconv.ParseFloat(string(ch), 64)
		return &Node{
			Op:    "",
			Value: value,
		}, nil
	} else if ch == '(' {
		p.next()
		node, err := parseExpr(p)
		if err != nil {
			return nil, err
		}
		p.skipWhitespace()
		if p.peek() != ')' {
			return nil, errors.New("Parenthesis error")
		}
		p.next()
		return node, nil
	}
	return nil, errors.New("Unexpected character: " + string(ch))
}

func buildTasks(node *Node, exprID int, taskIDs *[]int) (float64, int, error) {
	if node == nil {
		return 0, 0, errors.New("nil node")
	}
	if node.Op == "" {
		return node.Value, 0, nil
	}
	leftVal, leftTaskID, err := buildTasks(node.Left, exprID, taskIDs)
	if err != nil {
		return 0, 0, err
	}
	rightVal, rightTaskID, err := buildTasks(node.Right, exprID, taskIDs)
	if err != nil {
		return 0, 0, err
	}
	task := &Task{
		ID:              nextTaskID(),
		ExpressionID:    exprID,
		Operator:        node.Op,
		Arg1:            leftVal,
		Arg2:            rightVal,
		Status:          "pending",
		LeftDependency:  leftTaskID,
		RightDependency: rightTaskID,
	}
	switch node.Op {
	case "+":
		task.OperationTime = getOperationTime("TIME_ADDITION_MS")
	case "-":
		task.OperationTime = getOperationTime("TIME_SUBTRACTION_MS")
	case "*":
		task.OperationTime = getOperationTime("TIME_MULTIPLICATIONS_MS")
	case "/":
		task.OperationTime = getOperationTime("TIME_DIVISIONS_MS")
	default:
		task.OperationTime = 1000
	}
	tasks[task.ID] = task
	*taskIDs = append(*taskIDs, task.ID)
	if task.LeftDependency != 0 {
		task.Arg1 = math.NaN()
	}
	if task.RightDependency != 0 {
		task.Arg2 = math.NaN()
	}
	return math.NaN(), task.ID, nil
}

func getOperationTime(varName string) int {
	switch varName {
	case "TIME_ADDITION_MS":
		return 500
	case "TIME_SUBTRACTION_MS":
		return 500
	case "TIME_MULTIPLICATIONS_MS":
		return 700
	case "TIME_DIVISIONS_MS":
		return 800
	default:
		return 1000
	}
}

func nextExprID() int {
	exprCounter++
	return exprCounter
}

func nextTaskID() int {
	taskCounter++
	return taskCounter
}

func AddExpression(raw string) (int, error) {
	exprMutex.Lock()
	defer exprMutex.Unlock()
	node, err := parseExpression(raw)
	if err != nil {
		return 0, err
	}
	exprID := nextExprID()
	var taskIDs []int
	_, rootTaskID, err := buildTasks(node, exprID, &taskIDs)
	if err != nil {
		return 0, err
	}
	exprStatus := "pending"
	if rootTaskID == 0 {
		exprStatus = "completed"
	}
	expression := &Expression{
		ID:         exprID,
		Raw:        raw,
		Status:     exprStatus,
		Result:     math.NaN(),
		TaskIDs:    taskIDs,
		RootTaskID: rootTaskID,
	}
	expressions[exprID] = expression
	for _, tid := range taskIDs {
		t := tasks[tid]
		if t.LeftDependency == 0 && t.RightDependency == 0 {
			readyTasks = append(readyTasks, t)
		}
	}
	return exprID, nil
}

func GetExpressions() []*Expression {
	exprMutex.Lock()
	defer exprMutex.Unlock()
	exprs := make([]*Expression, 0, len(expressions))
	for _, e := range expressions {
		exprs = append(exprs, e)
	}
	return exprs
}

func GetExpression(id int) (*Expression, bool) {
	exprMutex.Lock()
	defer exprMutex.Unlock()
	e, ok := expressions[id]
	return e, ok
}

func GetReadyTask() (*Task, error) {
	exprMutex.Lock()
	defer exprMutex.Unlock()
	if len(readyTasks) == 0 {
		return nil, errors.New("No task available")
	}
	t := readyTasks[0]
	readyTasks = readyTasks[1:]
	return t, nil
}

func CompleteTask(taskID int, result float64) error {
	exprMutex.Lock()
	defer exprMutex.Unlock()
	t, ok := tasks[taskID]
	if !ok {
		return errors.New("Task not found")
	}
	if t.Status != "pending" {
		return errors.New("Task already completed")
	}
	t.Result = result
	t.Status = "completed"
	for _, tid := range expressions[t.ExpressionID].TaskIDs {
		child := tasks[tid]
		updated := false
		if child.LeftDependency == t.ID {
			child.Arg1 = result
			child.LeftDependency = 0
			updated = true
		}
		if child.RightDependency == t.ID {
			child.Arg2 = result
			child.RightDependency = 0
			updated = true
		}
		if updated && child.LeftDependency == 0 && child.RightDependency == 0 && child.Status == "pending" {
			readyTasks = append(readyTasks, child)
		}
	}
	expr := expressions[t.ExpressionID]
	if t.ID == expr.RootTaskID {
		expr.Status = "completed"
		expr.Result = result
	}
	return nil
}
