# URL Shortener & QR Generator

A simple yet powerful web application that allows users to shorten long URLs and generate QR codes for easy sharing and access.

## Features

- **URL Shortening**: Convert long URLs into shorter, more manageable links.
- **QR Code Generation**: Generate QR codes for shortened URLs for quick access.
- **Custom Aliases**: Allow users to define custom short link aliases.
- **API Support**: Provide a REST API for developers to integrate with their applications.

## Tech Stack

- **Backend**: Go (Gin Framework)
- **Frontend**: HTML, JavaScript, CSS (served by backend)
- **Database**: PostgreSQL

## Hosted Application

You can access the live application here:
[**Live Demo**](https://url-shortener-jl9o.onrender.com/)

## Screenshots

### Home Page
![Home Page 1](https://drive.google.com/file/d/1pxRyJn5Az4-IzQe2yG6R8n2w17gv38Gt/preview)![Alt text](https://drive.google.com/uc?export=view&id=1pxRyJn5Az4-IzQe2yG6R8n2w17gv38Gt)
![Home Page 2](https://drive.google.com/file/d/1WpW5Debq2AgU40uZ3GChZ7l-2kacK0nB/preview)

### URL Shortening And QR Code Generation
![Shortened URL](https://drive.google.com/file/d/1S8YLPImoXHVX7xMO_trWPXJ5yY6n6SXX/preview)

## Installation & Setup

### Prerequisites
Ensure you have the following installed:
- Go 1.18+
- PostgreSQL

### Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/Ajinkya2356/url_shortener.git
   cd url-shortener
   ```
2. Configure the PostgreSQL database in `.env`:
   ```env
   DB_HOST=localhost
   DB_PORT=5432
   DB_USER=your_user
   DB_PASSWORD=your_password
   DB_NAME=your_db
   ```
3. Install dependencies and run the application (backend serves the frontend as well):
   ```bash
   go mod tidy
   go run main.go
   ```

## API Endpoints

| Method | Endpoint | Description |
|--------|---------|-------------|
| POST | `/encode` | Shorten a URL and generate a QR code |

## Contributions
Contributions are welcome! Feel free to open an issue or submit a pull request.

## Contact
For any inquiries or suggestions, reach out via ajinkyajagtap2806@gmail.com.

