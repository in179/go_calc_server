document.getElementById("submitBtn").addEventListener("click", function() {
    const expr = document.getElementById("expression").value.trim();
    if (!expr) {
        alert("Получено пустое выражение");
        return;
    }
    fetch("/api/v1/calculate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ expression: expr })
    })
    .then(response => {
        if (!response.ok) {
            return response.text().then(text => { throw new Error(text) });
        }
        return response.json();
    })
    .then(data => {
        alert("Выражение принято. ID: " + data.id);
        document.getElementById("expression").value = "";
        loadExpressions();
    })
    .catch(err => {
        alert("Ошибка: " + err.message);
    });
});

function loadExpressions() {
    fetch("/api/v1/expressions")
    .then(response => response.json())
    .then(data => {
        const list = document.getElementById("expressionsList");
        list.innerHTML = "";
        data.expressions.forEach(expr => {
            const li = document.createElement("li");
            li.textContent = `ID: ${expr.id}, Статус: ${expr.status}, Результат: ${expr.result}`;
            list.appendChild(li);
        });
    })
    .catch(err => console.error(err));
}

setInterval(loadExpressions, 5000);
window.onload = loadExpressions;
