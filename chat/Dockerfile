FROM golang:alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

FROM builder AS chat_builder

COPY . .
RUN go build -o ./chat .

FROM alpine

COPY --from=chat_builder /src/chat /chat

CMD ["/chat"]

#FROM scratch as bin
#COPY --from=chat_builder /src/chat /