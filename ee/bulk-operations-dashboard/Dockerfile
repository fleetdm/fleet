# Use the official Node.js 14 image as a base
FROM node:20@sha256:e06aae17c40c7a6b5296ca6f942a02e6737ae61bbbf3e2158624bb0f887991b5

# Set the working directory in the container
WORKDIR /usr/src/app

# Copy the package.json
COPY package.json ./

# Install vulnerability dashboard dependencies
RUN npm install

# Copy the vulnerability dashboard into the container
COPY . .

# Copy the entrypoint script into the container
COPY entrypoint.sh /usr/src/app/entrypoint.sh

# Make sure the entrypoint script is executable
RUN chmod +x /usr/src/app/entrypoint.sh

# Expose the port the vulnerability dashboard runs on
EXPOSE 1337

# Set the entrypoint script as the entry point
ENTRYPOINT ["/usr/src/app/entrypoint.sh"]
