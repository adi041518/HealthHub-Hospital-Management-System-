# 1️⃣ Base image (Go installed)
FROM golang:1.25

# 2️⃣ Set working directory inside container
WORKDIR /app

# 3️⃣ Copy go files
COPY go.mod go.sum ./

# 4️⃣ Download dependencies
RUN go mod download

# 5️⃣ Copy all source code
COPY . .

# 6️⃣ Build the Go app
RUN go build -o app

# 7️⃣ Expose port 
EXPOSE 8080

# 8️⃣ Command to run app
CMD ["./app"]
