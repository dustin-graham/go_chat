# Chat Server
## Environment
- CHAT_SERVER_PORT - defaults to 8080
- CHAT_SERVER_IP - defaults to 127.0.0.1
- CHAT_SERVER_LOG_FILE_PATH - defaults to chat_log.txt

## Running this Sample
### GO Run
`go run .`

### Docker Compose
`docker-compose up --build`
Please note that the compose config depends on a file named `env_file` which contains the necessary environment variables for the program.

### Makefile
This just adds some convenience to docker-compose operations.

I won't enumerate the make file scripts available as those are easy enough to read directly. But if you are unfamiliar with Make you can try the following:

`make rebuild-and-run-chat-server`

This will stop the server if it is already running, rebuild the container, and bring it up