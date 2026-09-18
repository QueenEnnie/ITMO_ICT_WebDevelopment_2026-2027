# Лабораторная работа 1

Номер в журнале: **26**, вариант задания 2: **2 — квадратное уравнение**.

## Запуск

Установите Go 1.22 или новее. Все команды выполняются из папки `laboratory_work_1`.
Сначала запустите сервер, затем клиент в другом терминале. Остановка сервера: Ctrl+C.
Все программы работают локально, на `127.0.0.1`.

| Задание | Сервер | Клиент |
| --- | --- | --- |
| 1: UDP | `go run ./task1/server/main.go` | `go run ./task1/client/main.go` |
| 2: TCP, уравнение | `go run ./task2/server/main.go` | `go run ./task2/client/main.go` |
| 3: HTML | `go run ./task3/main.go` | Открыть http://127.0.0.1:8003 |
| 4: TCP-чат | `go run ./task4/server/main.go` | `go run ./task4/client/main.go` в нескольких терминалах |
| 5: журнал | `go run ./task5/main.go` | Открыть http://127.0.0.1:8005 |


## Отчёт

Отчёт находится в `docs/index.md`, конфигурация — `mkdocs.yml`.
Для сборки документации нужен Python с MkDocs:

```text
python -m pip install mkdocs
python -m mkdocs serve
python -m mkdocs build --strict
```

