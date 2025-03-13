/**
 * Extract and cleanup JSON from an input string.
 * @param {string} input - The raw input string that may contain extra characters or escaped sequences.
 * @returns {object|null} - The parsed JSON object if successful, otherwise null.
 */
function extractAndCleanJSON(input) {
    // Use a regular expression to capture the JSON object.
    // The "s" flag allows dot (.) to match newlines.
    const jsonMatch = input.match(/{.*}/s);
    if (!jsonMatch) {
      //console.error("No JSON object found in input.");
      return null;
    }
  
    let jsonString = jsonMatch[0];
  
    // Cleanup: Remove unnecessary newline characters or unwanted escapes.
    jsonString = jsonString.replace(/\\n/g, ''); 
    jsonString = jsonString.replace(/\\'/g, "'");
  
    try {
      // Parse the cleaned JSON string.
      const jsonObj = JSON.parse(jsonString);
      return jsonObj;
    } catch (e) {
      //console.error("Error parsing JSON:", e);
      return null;
    }
  }
  
  // Connect to the WebSocket endpoint on the Go server.
  const ws = new WebSocket("ws://localhost:8080/ws");
  let last_post = "";
  
  ws.onopen = () => {
    console.log("Connected to the WebSocket server.");
  };
  
  ws.onmessage = (event) => {
    // Extract and cleanup JSON from the raw message data.
    const data = extractAndCleanJSON(event.data);
    if (!data || !data.result) {
      //console.error("Failed to extract valid JSON news data.");
      return;
    }
    
    console.log("New news received:", data);
    
    const superFakeStreamContainer = document.getElementById('superFakeStreamContainer');
    //superFakeStreamContainer.innerHTML = '';
  
    // Update featured new item and add new card to the top of the list. Unhide the featured previous card.
    document.getElementById('newsTitle').textContent = data.result.title;
    document.getElementById('newsFakeScore').textContent = data.result.score;
    const newsURL = document.getElementById('newsURL');
    newsURL.textContent = data.result.url;
    newsURL.href = data.result.url;
    document.getElementById('newsReason').textContent = data.result.reason;

    const newsEl = document.createElement('div');
    newsEl.className = 'card';
    newsEl.style.border = '5px solid';
    newsEl.style.borderColor = 'red'; // Todo: change color based on how fake the news is.
    newsEl.style.display = 'none'; // Hide the new card by default until it's not featured
  
    newsEl.innerHTML = `
          <h3>${data.result.title}</h3>
          <p>Fake Score: ${data.result.score}</p>
          <p>URL: <a href="${data.result.url}" target="_blank">${data.result.url}</a></p>
          <p>Reason: ${data.result.reason}</p>
      `;
    if (last_post) { // Unhide the previous card - it's no longer featured
        last_post.style.display = 'block';
    }
    last_post = newsEl;
    if (superFakeStreamContainer.firstChild) {
        superFakeStreamContainer.insertBefore(newsEl, superFakeStreamContainer.firstChild);
    } else {
        superFakeStreamContainer.appendChild(newsEl);
    }
    document.getElementById('superFakeStream').style.display = 'block';
    document.getElementById('loadingData').style.display = 'none';
    document.getElementById('latestFakeNews').style.display = 'block';
  };
  
  ws.onerror = (error) => {
    console.error("WebSocket error:", error);
    displayError("WebSocket error.");
  };
  
  ws.onclose = () => {
    console.log("WebSocket connection closed.");
    displayError("WebSocket connection closed.");
  };
  
  function displayError(message) {
    const errorContainer = document.createElement('div');
    errorContainer.textContent = message;
    errorContainer.className = 'error-message';
    document.body.prepend(errorContainer);
    setTimeout(() => errorContainer.remove(), 5000);
  }
  