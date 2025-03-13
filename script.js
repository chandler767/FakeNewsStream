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

// ----- Chart.js Initialization Code Start -----
// This block initializes the chart inside the chartContainer element.
// Ensure that your index.html includes a canvas element with the id "fakeOMeterChart".
const chartCtx = document.getElementById('fakeOMeterChart').getContext('2d');

// Create the Chart.js chart with type "line" (configured as an area chart by enabling fill).
const fakeOMeterChart = new Chart(chartCtx, {
    type: 'line', // Using line chart, with fill enabled to simulate an area chart
    data: {
        labels: [], // Labels (time) will be added but hidden in the chart
        datasets: [{
            label: 'Fake Score',
            data: [], // Array to hold the score values received
            fill: true, // Fill the area under the line
            backgroundColor: 'rgba(139, 0, 0, 0.5)', // Dark red fill color with opacity
            borderColor: 'darkred', // Dark red border for the line
            tension: 0.4 // Smooths the line (curved line)
        }]
    },
    options: {
        responsive: true, // Chart adjusts to container size
        maintainAspectRatio: false, // Allows the chart to fill the container's width and height
        plugins: {
            title: {
                display: true,
                text: 'Fake-O-Meter' // Chart title
            }
        },
        scales: {
            x: {
                ticks: {
                    display: false // Hide x-axis tick labels (time)
                },
                grid: {
                    display: false // Optionally hide gridlines on x-axis
                }
            },
            y: {
                title: {
                    display: true,
                    text: 'Score' // Label for y-axis
                }
            }
        }
    }
});
// ----- Chart.js Initialization Code End -----

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

    // ----- Chart Update Code Start -----
    // If a valid score is received, update the chart with the new score.
    if (data.result.score !== undefined) {
        // Add a new label with the current time (even though it's hidden).
        fakeOMeterChart.data.labels.push(new Date().toLocaleTimeString());
        // Add the new score value to the dataset.
        fakeOMeterChart.data.datasets[0].data.push(data.result.score);

        // Limit the dataset to the last 50 values.
        if (fakeOMeterChart.data.labels.length > 50) {
            fakeOMeterChart.data.labels.shift(); // Remove oldest label
            fakeOMeterChart.data.datasets[0].data.shift(); // Remove oldest data point
        }
        // Update the chart to reflect the new data.
        fakeOMeterChart.update();
    }
    // ----- Chart Update Code End -----
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
