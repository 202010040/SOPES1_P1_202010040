FROM node:22-alpine

# Crear directorio de trabajo
WORKDIR /app

# Copiar package.json y package-lock.json
COPY package*.json ./

# Instalar dependencias
RUN npm install

# Copiar el resto del código fuente
COPY . .

# Exponer el puerto 3000 (React Dev Server)
EXPOSE 3000

# Comando para iniciar el servidor de desarrollo
CMD ["npm", "start"]
