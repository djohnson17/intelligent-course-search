# Intelligent Course Search System

A full-stack university course discovery application that helps students find courses using natural-language queries.

Built with **Go, HTML, CSS, and JavaScript**, the application provides an interactive chat-style interface for searching course schedules and viewing relevant course information.

## Features

- **Natural-language search:** Search for courses using everyday questions rather than exact course codes or titles.
- **Relevance-ranked results:** Matches queries against course information and ranks results using a local search algorithm.
- **Course details:** View relevant course information, including course codes, instructors, schedules, and locations when available.
- **Interactive web interface:** Chat-style search experience with dynamically generated course results.

## Tech Stack

| Category | Technologies |
|---|---|
| Backend | Go |
| Frontend | HTML, CSS, JavaScript |
| Data | CSV |
| Search | Custom keyword matching and relevance scoring |
| Original AI architecture | OpenAI API, ChromaDB |

## How It Works

The current demo loads a university course schedule from a CSV file into memory. When a user submits a question, the Go backend processes the query, identifies relevant terms, scores matching courses, and returns ranked results through a JSON API. The frontend displays those results in an interactive interface.

The project originally explored an AI-powered retrieval architecture using OpenAI and ChromaDB for natural-language query interpretation and vector-based course retrieval. The current reconstruction uses local search so the application can run without external API credentials or database services.

## Getting Started

### Prerequisites

- Go installed on your machine
- Git

### Installation

Clone the repository:

```bash
git clone https://github.com/djohnson17/intelligent-course-search.git
cd intelligent-course-search
```

Run the application:

```bash
go run .
```

Open the application in your browser:

```text
http://localhost:8080
```

## Project Structure

```text
course-finder-reconstructed/
├── main.go
├── handlers.go
├── localSearch.go
├── database.go
├── go.mod
├── go.sum
├── Fall 2024 Class Schedule 08082024.csv
├── templates/
│   └── index.html
└── static/
    └── images/
```

## Project Background

This project began as a university software development assignment focused on building an intelligent course search system. The original implementation explored vector databases, embeddings, and large language models. This reconstructed version restores the interactive application with a lightweight local search backend.

## Screenshots

### Course Finder Interface
![Course Finder](screenshots/course_finder_home.png)

### Subject Search
![Search Results](screenshots/course_finder3.png)

### Instructor Search
![Instructor Search](screenshots/course_finder2.png)
