const express = require('express');
const axios = require('axios');
const dotenv = require('dotenv');
const path = require('path');
const cors = require('cors'); // Import the cors package

dotenv.config({ path: path.resolve(__dirname, '.env') });

const app = express();
const PORT = process.env.PORT || 3001;
const GO_VALIDATOR_API_URL = process.env.GO_VALIDATOR_API_URL || 'http://localhost:8080/api/validate';
const DISPOSABLE_DOMAINS_URL = process.env.DISPOSABLE_DOMAINS_URL || 'https://raw.githubusercontent.com/disposable-email-domains/disposable-email-domains/refs/heads/main/disposable_email_blocklist.conf';
const REFRESH_INTERVAL_HOURS = parseInt(process.env.REFRESH_INTERVAL_HOURS || '24', 10);
const API_KEYS = process.env.API_KEYS ? process.env.API_KEYS.split(',').map(key => key.trim()) : []; // New: Read API keys

let disposableDomains = new Set();
let lastLoadedTimestamp = 0; // Timestamp of the last successful load

// Helper function to format date into a human-readable string
function formatTimestamp(timestamp) {
    if (timestamp === 0) {
        return 'Not yet refreshed';
    }
    const date = new Date(timestamp);
    const months = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"];
    const day = date.getDate();
    const month = months[date.getMonth()];
    const year = date.getFullYear();
    let hours = date.getHours();
    const minutes = date.getMinutes();
    const ampm = hours >= 12 ? 'PM' : 'AM';
    hours = hours % 12;
    hours = hours ? hours : 12; // the hour '0' should be '12'
    const minutesStr = minutes < 10 ? '0' + minutes : minutes;

    let daySuffix;
    if (day > 3 && day < 21) daySuffix = 'th';
    else {
        switch (day % 10) {
            case 1:  daySuffix = 'st'; break;
            case 2:  daySuffix = 'nd'; break;
            case 3:  daySuffix = 'rd'; break;
            default: daySuffix = 'th';
        }
    }

    return `${day}${daySuffix} ${month} ${year} at ${hours}:${minutesStr} ${ampm}`;
}

// Function to load disposable domains from the external blocklist
async function loadDisposableDomains() {
    try {
        console.log(`Fetching disposable domains from: ${DISPOSABLE_DOMAINS_URL}`);
        const response = await axios.get(DISPOSABLE_DOMAINS_URL);
        // Split by newline, trim whitespace, filter out empty lines and comments
        const domains = response.data.split('\n').map(d => d.trim()).filter(d => d.length > 0 && !d.startsWith('#'));
        disposableDomains = new Set(domains);
        lastLoadedTimestamp = Date.now(); // Update timestamp on successful load
        console.log(`Loaded ${disposableDomains.size} disposable domains. Next refresh in ${REFRESH_INTERVAL_HOURS} hours.`);
    } catch (error) {
        console.error('Failed to load disposable domains:', error.message);
        // In a production environment, you might want to implement retry logic or
        // load from a local fallback file if the external fetch fails.
        // For this example, we'll proceed with an empty or previously loaded set.
    }
}

// Load domains on service startup
loadDisposableDomains();

// Set up periodic refresh for disposable domains
setInterval(() => {
    const now = Date.now();
    const timeSinceLastLoad = now - lastLoadedTimestamp;
    const refreshThreshold = REFRESH_INTERVAL_HOURS * 60 * 60 * 1000; // Convert hours to milliseconds

    if (timeSinceLastLoad >= refreshThreshold) {
        console.log(`Refreshing disposable domains blocklist after ${REFRESH_INTERVAL_HOURS} hours.`);
        loadDisposableDomains();
    }
}, 60 * 60 * 1000); // Check every hour if a refresh is due

// Middleware to parse JSON requests
app.use(express.json());

// Enable CORS for all origins
app.use(cors());

// Serve static files from the 'static' directory at the project root
app.use(express.static(path.join(__dirname, '..', 'static')));

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
    // New: API Key authentication
    if (API_KEYS.length > 0) { // Only enforce if API_KEYS are configured
        const clientApiKey = req.headers['x-api-key'] || req.query.api_key;
        if (!clientApiKey || !API_KEYS.includes(clientApiKey)) {
            return res.status(403).json({ error: 'Forbidden: Invalid or missing API Key.' });
        }
    }

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

        // Determine the simple boolean result
        const result = (finalStatus === 'VALID' || finalStatus === 'PROBABLY_VALID');

        // 3. Return the combined result
        res.json({
            email: email,
            originalValidation: originalValidationResult,
            isDisposableByExternalBlocklist: isDisposableByExternalBlocklist,
            lastRefreshed: formatTimestamp(lastLoadedTimestamp), // Use the new formatting function
            finalStatus: finalStatus,
            RESULT: result // Added the new RESULT field
        });

    } catch (error) {
        console.error('Error during proxy validation:', error.message);
        if (error.response) {
            // The request was made and the server responded with a status code
            // that falls out of the range of 2xx (e.g., 400, 500 from Go API)
            res.status(error.response.status).json({
                error: `Error from upstream validator: ${error.response.data.error || error.response.statusText}`,
                details: error.response.data,
                RESULT: false // Set RESULT to false on error
            });
        } else if (error.request) {
            // The request was made but no response was received (e.g., Go API is down)
            res.status(500).json({ 
                error: 'No response from upstream validator service. Is it running?',
                RESULT: false // Set RESULT to false on error
            });
        } else {
            // Something happened in setting up the request that triggered an Error
            res.status(500).json({ 
                error: `Internal server error in proxy service: ${error.message}`,
                RESULT: false // Set RESULT to false on error
            });
        }
    }
});

// Start the Node.js proxy service
app.listen(PORT, () => {
    console.log(`Proxy validator service listening on port ${PORT}`);
    console.log(`Configured Go Validator API URL: ${GO_VALIDATOR_API_URL}`);
    console.log(`Configured Disposable Domains Blocklist URL: ${DISPOSABLE_DOMAINS_URL}`);
    if (API_KEYS.length > 0) {
        console.log(`API Key authentication is ENABLED for /proxy-validate.`);
    } else {
        console.log(`API Key authentication is DISABLED for /proxy-validate (no API_KEYS configured).`);
    }
});