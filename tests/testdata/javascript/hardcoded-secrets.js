// Bad: Hardcoded secrets (for content pattern testing)

const API_KEY = "sk-1234567890abcdef";
const apiKey = "my-secret-key-12345";
const password = "admin123";

const config = {
  secret: "my-secret-token",
  apiKey: "hardcoded-api-key"
};

// This should also be caught
const SECRET_TOKEN = "secret_abc123";

module.exports = { API_KEY, config };
