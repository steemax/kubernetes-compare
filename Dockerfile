FROM golang:1.19 as builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o main main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates

# add non-root user
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# set workdir to created user
WORKDIR /home/appuser
COPY --from=builder /app/main .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/conf ./conf

RUN chown -R appuser:appgroup /home/appuser

# change user to non-root for run application
USER appuser

CMD ["./main"]
