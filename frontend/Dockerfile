FROM node:22-alpine AS dependencies
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS builder
WORKDIR /app
ARG NEXT_PUBLIC_API_BASE_URL
ENV NEXT_PUBLIC_API_BASE_URL=$NEXT_PUBLIC_API_BASE_URL
COPY --from=dependencies /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:22-alpine AS runner
WORKDIR /app
ENV NODE_ENV=production
RUN addgroup -S skillarena && adduser -S -G skillarena skillarena
COPY --from=builder --chown=skillarena:skillarena /app/.next/standalone ./
COPY --from=builder --chown=skillarena:skillarena /app/.next/static ./.next/static
COPY --from=builder --chown=skillarena:skillarena /app/public ./public
USER skillarena
EXPOSE 3000
CMD ["node", "server.js"]
