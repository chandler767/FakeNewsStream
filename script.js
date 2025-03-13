/**
 * Extract and cleanup JSON from an input string.
 * @param {string} input - The raw input string that may contain extra characters or escaped sequences.
 * @returns {object|null} - The parsed JSON object if successful, otherwise null.
 */
function extractAndCleanJSON(input) {
  // Use a regular expression to capture the JSON object.
  // The "s" flag allows the dot (.) to match newline characters.
  const jsonMatch = input.match(/{.*}/s);
  if (!jsonMatch) {
    // No valid JSON object found in the input string.
    return null;
  }

  // Get the matched JSON string.
  let jsonString = jsonMatch[0];

  // Cleanup: Remove unnecessary newline escape sequences and unescape single quotes.
  jsonString = jsonString.replace(/\\n/g, '');
  jsonString = jsonString.replace(/\\'/g, "'");

  try {
    // Attempt to parse the cleaned JSON string.
    const jsonObj = JSON.parse(jsonString);
    return jsonObj;
  } catch (e) {
    // Parsing failed, return null.
    return null;
  }
}

// Create a WebSocket connection to the server endpoint.
const ws = new WebSocket("ws://localhost:8080/ws");
let last_post = ""; // Variable to keep track of the last news card element

// When the connection opens successfully.
ws.onopen = () => {
  console.log("Connected to the WebSocket server.");
};

// Handle incoming messages from the WebSocket.
ws.onmessage = (event) => {
  // Extract and cleanup JSON from the raw message data.
  const data = extractAndCleanJSON(event.data);
  if (!data || !data.result) {
    // If the data is invalid or missing the 'result' property, exit the function.
    return;
  }
  
  console.log("New news received:", data);

  // Get the container element for the stream of news cards.
  const superFakeStreamContainer = document.getElementById('superFakeStreamContainer');

  // Update the featured news item elements.
  document.getElementById('newsTitle').textContent = data.result.title;
  document.getElementById('newsFakeScore').textContent = data.result.score;
  const newsURL = document.getElementById('newsURL');
  newsURL.textContent = data.result.url;
  newsURL.href = data.result.url;
  document.getElementById('newsReason').textContent = data.result.reason;

  // Create a new card element for the incoming news.
  const newsEl = document.createElement('div');
  newsEl.className = 'card';
  // Set border style; color can be changed based on how fake the news is.
  newsEl.style.border = '5px solid';
  newsEl.style.borderColor = 'red';
  // Hide the new card initially until it's not featured.
  newsEl.style.display = 'none';
  
  // Set the inner HTML content for the news card.
  newsEl.innerHTML = `
    <h3>${data.result.title}</h3>
    <p>Fake Score: ${data.result.score}</p>
    <p>URL: <a href="${data.result.url}" target="_blank">${data.result.url}</a></p>
    <p>Reason: ${data.result.reason}</p>
  `;

  // If there is a previously featured card, unhide it.
  if (last_post) {
    last_post.style.display = 'block';
  }
  
  // Update the reference to the last post with the new card.
  last_post = newsEl;
  
  // Insert the new card at the top of the stream container.
  if (superFakeStreamContainer.firstChild) {
    superFakeStreamContainer.insertBefore(newsEl, superFakeStreamContainer.firstChild);
  } else {
    superFakeStreamContainer.appendChild(newsEl);
  }
  
  // Update visibility for various UI elements.
  document.getElementById('superFakeStream').style.display = 'block';
  document.getElementById('loadingData').style.display = 'none';
  document.getElementById('latestFakeNews').style.display = 'block';
};

// Handle any errors from the WebSocket.
ws.onerror = (error) => {
  console.error("WebSocket error:", error);
  displayError("WebSocket error.");
};

// Handle WebSocket connection closure.
ws.onclose = () => {
  console.log("WebSocket connection closed.");
  displayError("WebSocket connection closed.");
};

/**
 * Display an error message to the user.
 * @param {string} message - The error message to display.
 */
function displayError(message) {
  // Create an error message container element.
  const errorContainer = document.createElement('div');
  errorContainer.textContent = message;
  errorContainer.className = 'error-message';
  // Prepend the error container to the body.
  document.body.prepend(errorContainer);
  // Remove the error message after 5 seconds.
  setTimeout(() => errorContainer.remove(), 5000);
}
