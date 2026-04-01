# Development Dockerfile for Vite React App
FROM node:20-alpine

WORKDIR /app

# Copy package.json and package-lock.json
COPY package*.json ./

# Install dependencies
RUN npm install

# Copy source code
COPY . .

# Expose Vite's default port
EXPOSE 5173

# Start Vite in development mode (bound to 0.0.0.0 so host can access it)
CMD ["npm", "run", "dev", "--", "--host"]
