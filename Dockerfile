############################
# STEP 1 build executable binary
############################
FROM golang:1.27.1-alpine3.24 AS builder
# Install git + SSL ca certificates.
# Git is required for fetching the dependencies.
# Ca-certificates is required to call HTTPS endpoints.
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates
# Create appuser
ENV USER=appuser
ENV UID=10001
# See https://stackoverflow.com/a/55757473/12429735RUN 
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"

WORKDIR $GOPATH/src/mypackage/myapp/
COPY . .

RUN go mod tidy
RUN go mod verify
RUN GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /go/bin/app

############################
# STEP 2 build a small image
############################
FROM scratch

ENV DATABASE=/data/finance.db

# Import from builder.
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
# Copy our static executable
COPY --from=builder /go/bin/app /go/bin/app
COPY templates /templates/
# Use an unprivileged user.
USER appuser:appuser
ENV ROOT_PATH="/"

EXPOSE 9091

ENTRYPOINT ["/go/bin/app"]
