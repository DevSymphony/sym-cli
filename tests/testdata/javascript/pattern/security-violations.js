// File with security violations

// Bad: hardcoded API key
const API_KEY = 'sk-1234567890abcdef1234567890abcdef';

// Bad: hardcoded password
const password = 'mySecretPassword123';

// Bad: hardcoded secret
const client_secret = 'super-secret-value-12345';

// Bad: hardcoded token
const access_token = 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9';

// Good: using environment variables
const apiKey = process.env.API_KEY;
const userPassword = process.env.PASSWORD;
const clientSecret = process.env.CLIENT_SECRET;
