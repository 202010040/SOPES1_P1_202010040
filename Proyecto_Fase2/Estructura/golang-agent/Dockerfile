# Multi-stage build para optimizar el tamaño de la imagen
FROM golang:1.22.2-alpine AS builder

# Instalar dependencias necesarias
RUN apk add --no-cache git ca-certificates

# Establecer directorio de trabajo
WORKDIR /app

# Copiar archivos de dependencias
COPY go.mod go.sum ./

# Descargar dependencias
RUN go mod download

# Copiar código fuente
COPY . .

# Compilar la aplicación con optimizaciones
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Imagen final minimalista
FROM alpine:latest

# Instalar certificados SSL y crear usuario no-root
RUN apk --no-cache add ca-certificates \
    && addgroup -g 1001 -S appgroup \
    && adduser -u 1001 -S appuser -G appgroup

# Crear directorio de trabajo
WORKDIR /root/

# Copiar el binario compilado desde el builder
COPY --from=builder /app/main .

# Crear directorio /proc para los archivos de monitoreo
RUN mkdir -p /proc

# Cambiar permisos y propietario
RUN chown -R appuser:appgroup /root
USER appuser

# Exponer puerto
EXPOSE 3001

# Variables de entorno por defecto
ENV API_PORT=3001

# Etiquetas de metadatos
LABEL maintainer="tu-email@ejemplo.com"
LABEL description="API de Monitoreo del Sistema"
LABEL version="1.0.0"

# Comando para ejecutar la aplicación
CMD ["./main"]