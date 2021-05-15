chat-server-make-env-file:
	sed -e 's/export //g' .envrc > env_file

build-chat-server: chat-server-make-env-file
	docker-compose build

stop-chat-server:
	docker-compose down

run-chat-server: stop-chat-server
	docker-compose up -d

rebuild-and-run-chat-server: stop-chat-server build-chat-server run-chat-server
