##
## BUILD
## 
FROM golang:1.17-bullseye AS build

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download
COPY *.go ./

RUN go build -o /reciperandomizer

##
## Deploy
##
FROM gcr.io/distroless/base-debian10 AS deploy

WORKDIR /app

COPY --from=build /reciperandomizer /app/reciperandomizer
COPY templates/* /app/*

EXPOSE 32801

USER nonroot:nonroot

ENTRYPOINT [ "/app/reciperandomizer" ]
