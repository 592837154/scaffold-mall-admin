FROM docker.m.daocloud.io/library/golang:1.25-alpine AS backend-builder

WORKDIR /src

COPY backend/go.mod backend/go.sum ./
RUN go env -w GOPROXY=https://goproxy.cn,direct && go mod download

COPY backend/main.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /server main.go

FROM docker.m.daocloud.io/library/nginx:1.27-alpine

WORKDIR /app

COPY frontend/dist /usr/share/nginx/html
COPY --from=backend-builder /server /app/server
COPY nginx.hf.conf /etc/nginx/conf.d/default.conf
COPY start-hf.sh /app/start-hf.sh

RUN chmod +x /app/server /app/start-hf.sh

EXPOSE 7860

CMD ["/app/start-hf.sh"]
