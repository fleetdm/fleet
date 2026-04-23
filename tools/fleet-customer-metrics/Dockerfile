FROM node:22-slim
WORKDIR /app
COPY package.json ./
RUN npm install
COPY import_data.ts ./
CMD ["npx", "tsx", "import_data.ts"]
