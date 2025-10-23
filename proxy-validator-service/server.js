const express = require('express');
const axios = require('axios');
const dotenv = require('dotenv');
const path = require('path');

dotenv.config({ path: path.resolve(__dirname, '.env') });

const app = express();
const PORT = process.env.PORT || 3001;
const GO_VALIDATOR_API_URL = process.env.GO_VALIDATOR_API_URL || 'http://localhost:8080/api/validate';
const DISPOSABLE_DOMAINS_URL = process.env.DISPOSABLE_DOMAINS_URL || 'https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/refs/heads/main/disposable_email_blocklist.conf';

let disposableDomains = new Set();

// Function to load disposable domains from the external blocklist
async function loadDisposableDomains() {
    try {
        console.log(`Fetching disposable domains from: ${DISPOSABLE_DOMAINS_URL}`);
        const response = await axios.get(DISPOSABLE_DOMAINS_URL);
        // Split by newline, trim whitespace, filter out empty lines and comments
        const domains = response.data.split('\n').map(d => d.trim()).filter(d => d.length > 0 && !d.startsWith('#'));
        disposableDomains = new Set(domains);
        console.log(`Loaded ${disposableDomains.size} disposable domains.`);
    } catch (error) {
        console.error('Failed to load disposable domains:', error.message);
        // In a production environment, you might want to implement retry logic or
        // load from a local fallback file if the external fetch fails.
        // For this example, we'll proceed with an empty or previously loaded set.
    }
}

// Load domains on service startup
loadDisposableDomains();

// Middleware to parse JSON requests
app.use(express.json());

// Helper function to extract the domain from an email address
function getDomainFromEmail(email) {
    const parts = email.split('@');
    if (parts.length === 2) {
        return parts[1].toLowerCase();
    }
    return null;
}

// New API endpoint for proxy validation
app.get('/proxy-validate', async (req, res) => {
    const { email } = req.query;

    if (!email) {
        return res.status(400).json({ error: 'Email query parameter is required.' });
    }

    try {
        // 1. Call the existing Go email validator API
        const goApiResponse = await axios.get(`${GO_VALIDATOR_API_URL}?email=${encodeURIComponent(email)}`);
        const originalValidationResult = goApiResponse.data;

        let isDisposableByExternalBlocklist = false;
        let finalStatus = originalValidationResult.status;

        // 2. If the original API response is VALID, then perform an additional check
        //    against the external disposable email blocklist.
        if (originalValidationResult.status === 'VALID') {
            const domain = getDomainFromEmail(email);
            if (domain && disposableDomains.has(domain)) {
                isDisposableByExternalBlocklist = true;
                finalStatus = 'DISPOSABLE_BY_EXTERNAL_BLOCKLIST'; // Custom status for this specific check
            }
        }

        // 3. Return the combined result
        res.json({
            email: email,
            originalValidation: originalValidationResult,
            isDisposableByExternalBlocklist: isDisposableByExternalBlocklist,
            finalStatus: finalStatus
        });

    } catch (error) {
        console.error('Error during proxy validation:', error.message);
        if (error.response) {
            // The request was made and the server responded with a status code
            // that falls out of the range of 2xx (e.g., 400, 500 from Go API)
            res.status(error.response.status).json({
                error: `Error from upstream validator: ${error.response.data.error || error.response.statusText}`,
                details: error.response.data
            });
        } else if (error.request) {
            // The request was made but no response was received (e.g., Go API is down)
            res.status(500).json({ error: 'No response from upstream validator service. Is it running?' });
        } else {
            // Something happened in setting up the request that triggered an Error
            res.status(500).json({ error: `Internal server error in proxy service: ${error.message}` });
        }
    }
});

// Start the Node.js proxy service
app.listen(PORT, () => {
    console.log(`Proxy validator service listening on port ${PORT}`);
    console.log(`Configured Go Validator API URL: ${GO_VALIDATOR_API_URL}`);
    console.log(`Configured Disposable Domains Blocklist URL: ${DISPOSABLE_DOMAINS_URL}`);
});