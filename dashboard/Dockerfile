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

# Create a non-root user/group (UID/GID 1001) to run the Next.js server
RUN addgroup -g 1001 -S nodejs && \
    adduser -u 1001 -S nextjs -G nodejs

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
