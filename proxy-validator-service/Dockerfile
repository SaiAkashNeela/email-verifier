# Use an official Node.js runtime as a parent image
FROM node:20-alpine

# Set the working directory in the container
WORKDIR /app

# Copy package.json and package-lock.json from the proxy-validator-service directory
# to the current working directory (/app) in the container.
COPY proxy-validator-service/package*.json ./

# Install app dependencies
RUN npm install

# Copy the rest of the application code from the proxy-validator-service directory
# to the current working directory (/app) in the container.
COPY proxy-validator-service/server.js ./

# Copy the static folder from the project root to the container root
COPY static /static

# Expose the port the app runs on
EXPOSE 3001

# Define environment variables
ENV PORT=3001
ENV GO_VALIDATOR_API_URL=http://email-validator:8080/api/validate
ENV DISPOSABLE_DOMAINS_URL=https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/refs/heads/main/disposable_email_blocklist.conf
ENV REFRESH_INTERVAL_HOURS=24
ENV API_KEYS="" # New: Default empty API_KEYS

# Run the application
CMD [ "node", "server.js" ]