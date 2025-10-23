# Use an official Node.js runtime as a parent image
FROM node:20-alpine

# Set the working directory in the container
WORKDIR /app

# Copy package.json and install dependencies
# A wildcard is used to ensure both package.json and package-lock.json are copied
# if package-lock.json exists.
COPY package*.json ./

RUN npm install

# Copy the rest of the application code
COPY server.js ./

# Expose the port the app runs on
EXPOSE 3001

# Define the command to run the app
CMD ["node", "server.js"]