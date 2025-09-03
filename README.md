# 1) Create a local .env from the template
cp dist.env .env

# 2) Fill in values in .env (JWT, Redis, SMTP, etc.)
# 3) Run the app
make run     # or: go run ./cmd/cargo-agent
