FROM node:20-alpine
WORKDIR /app

# Install basic build tools
RUN apk add --no-cache bash python3 build-base

# Copy package manifest and install dependencies
COPY package.json package-lock.json* ./
RUN npm install

# Copy source and build
COPY . .
RUN npm run build

ENV PORT=3000
EXPOSE 3000

# Serve the built app via Vite preview on port 3000
CMD ["npm", "run", "preview", "--", "--port", "3000"]
