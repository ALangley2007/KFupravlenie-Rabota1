# vshell

Эмулятор командной оболочки UNIX-подобной ОС, написанный на Go.

Текущее состояние: **Этап 1 — REPL**. Эмулятор ведет интерактивный диалог,
разбирает командные строки с аргументами в кавычках и передает их командам.
`ls` и `cd` пока являются заглушками; `exit` полностью реализована.

## Требования

* Go 1.22 или новее

## Использование

```
./run.sh          # запуск из исходников
make build        # сборка bin/vshell
make test         # запуск тестов
make check        # gofmt + go vet

```

## Структура проекта

```
src/main.go            точка входа
src/shell/parser.go    парсер командной строки (кавычки, экранирование)
src/shell/prompt.go    определение данных хоста и отрисовка приглашения
src/shell/commands.go  реестр команд: ls, cd (заглушки), exit
src/shell/shell.go     REPL, диспетчеризация команд, статусы выхода
tests/                 black-box тесты пакета shell

```

## Возможности

### Prompt

Промпт формируется на основе реальных данных хостовой машины: имени пользователя, имени хоста
(без доменной части) и директории, из которой был запущен эмулятор.
Домашняя директория отображается как `~`; суперпользователь получает `#` вместо `$`.

```
username@hostname:~$

```

### Парсер

Правила экранирования и кавычек соответствуют стандарту POSIX shell.

| Ввод | Аргументы | Правило | 
| ----- | ----- | ----- | 
| `ls a   b` | `a`, `b` | пробелы разделяют аргументы | 
| `ls "my dir"` | `my dir` | двойные кавычки сохраняют пробелы | 
| `ls 'my dir'` | `my dir` | одинарные кавычки сохраняют всё буквально | 
| `ls my\ dir` | `my dir` | обратный слеш экранирует следующий символ | 
| `ls "a \"b\""` | `a "b"` | внутри двойных кавычек `\` экранирует `"`, `\`, `$`, `` ` `` | 
| `ls a"b c"d` | `ab cd` | соседние части объединяются в один аргумент | 
| `ls ""` | (пустая строка) | пустые кавычки дают пустой аргумент | 

### Команды

| Команда | Поведение | 
| ----- | ----- | 
| `ls [args]` | заглушка: выводит свое имя и аргументы | 
| `cd [args]` | заглушка: выводит свое имя и аргументы | 
| `exit [n]` | завершает сессию со статусом `n` (mod 256) или статусом последней команды | 

Конец ввода (Ctrl-D) также завершает сессию.

### Ошибки и статусы выхода

| Ситуация | Сообщение | Статус | 
| ----- | ----- | ----- | 
| неизвестная команда | `vshell: foo: command not found` | 127 | 
| незакрытая кавычка | `` vshell: syntax error: unexpected EOF while looking for matching quote `"' `` | 2 | 
| обратный слеш в конце строки | `vshell: syntax error: unexpected EOF after escape character` | 2 | 
| `exit 1 2` | `vshell: exit: too many arguments` (сессия продолжается) | 1 | 
| `exit abc` | `vshell: exit: abc: numeric argument required` (сессия завершается, как в bash) | 2 | 

## Демонстрационная сессия

```
user@host:~/kf$ ls
ls args: []
user@host:~/kf$ ls -l "my dir" 'single quoted' escaped\ space
ls args: ["-l" "my dir" "single quoted" "escaped space"]
user@host:~/kf$ cd "/usr/local/my folder"
cd args: ["/usr/local/my folder"]
user@host:~/kf$ cd a"b c"d "" 'it"s'
cd args: ["ab cd" "" "it\"s"]
user@host:~/kf$
user@host:~/kf$ foo --bar
vshell: foo: command not found
user@host:~/kf$ ls "unterminated
vshell: syntax error: unexpected EOF while looking for matching quote `"'
user@host:~/kf$ ls trailing\
vshell: syntax error: unexpected EOF after escape character
user@host:~/kf$ exit 1 2
vshell: exit: too many arguments
user@host:~/kf$ exit 3
$ echo $?
3

```