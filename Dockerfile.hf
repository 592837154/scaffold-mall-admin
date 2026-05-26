FROM docker.m.daocloud.io/library/nginx:1.27-alpine

WORKDIR /app

COPY frontend/dist /usr/share/nginx/html
COPY backend/server /app/server
COPY nginx.hf.conf /etc/nginx/conf.d/default.conf
COPY start-hf.sh /app/start-hf.sh

RUN chmod +x /app/server /app/start-hf.sh

EXPOSE 7860

CMD ["/app/start-hf.sh"]
