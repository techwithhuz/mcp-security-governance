# Build stage
FROM node:20-alpine AS builder
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci || npm install
COPY . .
RUN npm run build

# Ensure public directory exists for COPY
RUN mkdir -p /app/public

# Runtime stage
FROM node:20-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production

# Install git (needed for sparse-checkout repo scanning) and create non-root user
RUN apk add --no-cache git && \
    addgroup -g 1001 -S nodejs && \
    adduser -u 1001 -S nextjs -G nodejs && \
    git config --system safe.directory '*'

COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public

# Give ownership to the non-root user
RUN chown -R nextjs:nodejs /app

USER 1001:1001

EXPOSE 3000
ENV PORT=3000
ENV HOSTNAME="0.0.0.0"
CMD ["node", "server.js"]
