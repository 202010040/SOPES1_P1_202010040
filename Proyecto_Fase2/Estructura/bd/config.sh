docker run -d \
  --name db-monitoreo \
  --restart unless-stopped \
  -e MYSQL_ROOT_PASSWORD=rootpass2025 \
  -e MYSQL_DATABASE=sistema_monitoreo \
  -e MYSQL_USER=user_monitoreo \
  -e MYSQL_PASSWORD=Ingenieria2025. \
  -p 3306:3306 \
  -v db_monitoreo:/var/lib/mysql \
  -v $(pwd)/init-db.sql:/docker-entrypoint-initdb.d/init-db.sql \
  -v $(pwd)/db_config.cnf:/etc/mysql/conf.d/custom.cnf \
  mysql:8.0
