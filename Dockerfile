# JED Servidor — imagem de produção para Railway e outros hosts Docker.
FROM golang:1.23.12-alpine AS build
WORKDIR /src
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/jed-server .

FROM alpine:3.22
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=build /out/jed-server /app/jed-server
ENV JED_SERVER_DATA=/data
EXPOSE 8787
CMD ["/app/jed-server", "--server"]
