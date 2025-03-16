# FakeNewsStream

## Overview

FakeNewsStream is a real-time streaming application designed to detect and display potentially fake news articles. It leverages Redpanda for message streaming, OpenAI's GPT-4 for fake news detection, and WebSockets for live updates to the frontend.

## Demo

Check out the [demo video](https://github.com/chandler767/FakeNewsStream/raw/refs/heads/master/Redpanda-FakeNewsDashboard-ChandlerMayo.mp4) for how to build this yourself using this code.

## Features

- **Real-time Data Streaming**: Fetches new posts from Reddit and processes them in real-time.
- **Fake News Detection**: Uses OpenAI's GPT-4 to evaluate the likelihood of news articles being fake.
- **WebSocket Integration**: Provides live updates to the frontend for the latest fake news.
- **Redpanda Integration**: Utilizes Redpanda for efficient message streaming and processing.

## Project Structure

- **go-proxy**: Contains the code for consuming messages from Redpanda and broadcasting them via WebSockets.
- **configs**: Configuration files for creating pipelines within Redpanda for both the Reddit data stream and for evaluating news.
- **frontend**: HTML, CSS, and JavaScript files for the frontend interface.

## License

This project is licensed under the Unlicense. See the [LICENSE](LICENSE) file for details.

## Acknowledgements

- [Redpanda](https://www.redpanda.com/)
- [OpenAI](https://www.openai.com/)
- [Gorilla WebSocket](https://github.com/gorilla/websocket)
- [Franz-go](https://github.com/twmb/franz-go)
- [Godotenv](https://github.com/joho/godotenv)
- [Chart.js](https://www.chartjs.org/)

## Contact

For more information, email chandler@chandlermayo.com
