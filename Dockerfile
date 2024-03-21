FROM golang:1.22-alpine
LABEL authors="mavifindsbugs"

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./src/*.go ./

RUN go build -o /dockrinfo

CMD [ "/dockrinfo" ]