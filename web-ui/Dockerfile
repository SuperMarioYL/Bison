# Build stage
FROM --platform=$BUILDPLATFORM node:20-alpine AS builder

WORKDIR /app

# Copy package files
COPY package*.json ./

# Install dependencies
# Remove package-lock.json and use npm install to ensure optional deps are installed correctly
RUN rm -f package-lock.json && npm install

# Copy source code
COPY . .

# Build the app
RUN npm run build

# Runtime stage - nginx supports multi-arch natively
FROM --platform=$TARGETPLATFORM nginx:alpine

# Copy nginx config
COPY nginx.conf /etc/nginx/nginx.conf

# Copy built files
COPY --from=builder /app/dist /usr/share/nginx/html

# Create cache directories with correct permissions
RUN mkdir -p /var/cache/nginx /var/run && \
    chown -R nginx:nginx /var/cache/nginx /var/run /usr/share/nginx/html

# Expose port
EXPOSE 80

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
