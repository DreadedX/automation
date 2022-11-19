FROM golang:alpine as build-automation

WORKDIR /src
COPY ./go.mod .
COPY ./go.sum .
RUN go mod download

COPY . .

RUN go build -o app


FROM golang:alpine

WORKDIR /app
COPY --from=build-automation /src/app /app/app
COPY --from=build-automation /src/config.yml /app/config.yml

CMD ["/app/app"]
