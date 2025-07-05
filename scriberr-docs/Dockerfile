# ---- Build stage ----
FROM node:20-alpine AS builder

WORKDIR /app

# Install dependencies first
COPY package.json package-lock.json* ./
RUN npm ci --only=production=false

# Copy the rest of the project files
COPY . .

# Set environment to skip optional dependencies that cause issues
ENV ROLLUP_SKIP_NATIVE=true
ENV NODE_ENV=production

# Build SvelteKit app
RUN npm run build

# ---- Production stage ----
FROM node:20-alpine

WORKDIR /app

# Install only production dependencies
COPY package.json package-lock.json* ./
RUN npm ci --only=production

# Copy the built application
COPY --from=builder /app/build ./build
COPY --from=builder /app/static ./static

ENV NODE_ENV=production
ENV PORT=3000

# Expose SvelteKit default port
EXPOSE 3000

# Start the server
CMD ["node", "build"]


