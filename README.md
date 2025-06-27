# HoosierEats

## To run this project
---

### Prerequisites:
- must have Node v22.16.0 or greater installed on your system
  - https://nodejs.org/en/download/
- must have Go v1.24.4 or greater installed on your system (probably could be lower since it's pretty backwards compatible)
  - https://go.dev/doc/install

NOTE: if you are running on Windows we recommend you run this in WSL

### To run:
1) Copy the `.example-env` file in `backend/` and rename it to `.env`
2) In the `.env` file, set the `OPENAI_API_KEY` to your api key
3) run `make start` in your terminal

The server will now be running locally @ `http://localhost:8080`
