SRC = "main.go"
BIN = "gh-vclone"

build:
	go build -o $(BIN) $(SRC)

install: build
	gh extension install .

uninstall:
	gh extension remove $(BIN)

reload: uninstall install