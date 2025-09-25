FROM alpine:latest

RUN apk add --no-cache bash

COPY light_process.sh /app/light_process.sh
RUN chmod +x /app/light_process.sh

CMD ["/app/light_process.sh"]
