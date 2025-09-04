FROM mysql:8.0

# Copiar archivos de configuración
COPY init-db.sql /docker-entrypoint-initdb.d/
COPY db_config.cnf /etc/mysql/conf.d/custom.cnf

# Variables de entorno por defecto
ENV MYSQL_ROOT_PASSWORD=rootpass2025
ENV MYSQL_DATABASE=sistema_monitoreo
ENV MYSQL_USER=user_monitoreo
ENV MYSQL_PASSWORD=Ingenieria2025.

# Exponer puerto
EXPOSE 3306

# Crear volumen para datos
VOLUME ["/var/lib/mysql"]
