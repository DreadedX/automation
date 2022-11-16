FROM golang:alpine as build-automation

WORKDIR /src
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download

COPY . .

RUN go build


FROM golang:alpine

WORKDIR /app
COPY --from=build-automation /src/automation /app/automation
COPY --from=build-automation /src/config.yml /app/config.yml

CMD ["/app/automation"]
