FROM golang:1.20
WORKDIR /app
COPY . .
RUN go build -o app main.go
CMD ["./app"]
EXPOSE 8080