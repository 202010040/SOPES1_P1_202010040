# Build stage
FROM node:18-alpine as build
WORKDIR /app
COPY package*.json ./
RUN npm install
COPY . .
RUN npm run build

# Production stage
FROM nginx:alpine

# Copiar build
COPY --from=build /app/build /usr/share/nginx/html

# Crear script de inicio
RUN echo '#!/bin/sh' > /start.sh && \
    echo 'PORT=${PORT:-8080}' >> /start.sh && \
    echo 'sed -i "s/listen 80/listen $PORT/g" /etc/nginx/conf.d/default.conf' >> /start.sh && \
    echo 'nginx -g "daemon off;"' >> /start.sh && \
    chmod +x /start.sh

EXPOSE 8080
CMD ["/start.sh"]