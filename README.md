# Audio Steganography Tool

## Program Description

Audio Steganography Tool is a web application that enables users to hide and extract secret messages within MP3 audio files using Multiple-LSB (Least Significant Bit) steganography method. The application provides a secure way to embed confidential data into audio files while maintaining audio quality and includes optional Vigenere cipher encryption for enhanced security.

Key features:
- Hide secret files within MP3 audio files using LSB steganography
- Extract hidden messages from steganographic audio files
- Vigenere cipher encryption for additional security
- PSNR (Peak Signal-to-Noise Ratio) quality assessment
- MP3 metadata preservation
- Configurable LSB bits (1-4) for capacity vs quality trade-off
- Random start position option for enhanced security

## Tech Stack

### Backend
- **Go 1.23.0** - Main programming language
- **Gin** - HTTP web framework
- **CORS** - Cross-Origin Resource Sharing middleware
- **LAME** - MP3 encoder (system dependency)
- **minimp3** - MP3 decoder library

### Frontend
- **React 19.1.1** - JavaScript library for user interfaces
- **TypeScript 4.9.5** - Type-safe JavaScript
- **Tailwind CSS 3.4.17** - Utility-first CSS framework
- **Axios 1.12.2** - HTTP client for API requests
- **Lucide React** - Icon library

### Infrastructure
- **Docker** - Containerization platform
- **Docker Compose** - Multi-container orchestration
- **Nginx** - Web server for frontend serving

## Dependencies

### Backend Dependencies
```
github.com/gin-contrib/cors v1.7.6
github.com/gin-gonic/gin v1.11.0
github.com/tosone/minimp3 v1.0.2
```

### Frontend Dependencies
```
react ^19.1.1
react-dom ^19.1.1
typescript ^4.9.5
axios ^1.12.2
tailwindcss ^3.4.17
lucide-react ^0.544.0
class-variance-authority ^0.7.1
clsx ^2.1.1
tailwind-merge ^3.3.1
```

### System Requirements
- **Docker** and **Docker Compose**
- **LAME encoder** (automatically installed in Docker containers)

## How to Run the Program

### Prerequisites
- Docker and Docker Compose installed on your system
- Ports 3000 and 8080 available on your machine

### Running with Docker Compose (Recommended)

1. Clone the repository and navigate to the project directory:
```bash
cd tugas-kecil-2
```

2. Start the application using Docker Compose:
```bash
docker-compose up --build
```

3. Access the application:
   - **Frontend**: http://localhost:3000
   - **Backend API**: http://localhost:8080

4. To stop the application:
```bash
docker-compose down
```

### API Endpoints

- `POST /api/v1/stego/insert` - Insert secret message into MP3 file
- `POST /api/v1/stego/extract` - Extract secret message from steganographic MP3 file
- `GET /api/v1/health` - Health check endpoint

### Usage Instructions

1. **Insert Mode**: 
   - Upload an MP3 audio file
   - Upload a secret file to hide
   - Enter a key for steganography
   - Configure optional settings (encryption, random start, LSB bits)
   - Download the generated steganographic MP3 file

2. **Extract Mode**:
   - Upload a steganographic MP3 file
   - Enter the same key used during insertion
   - Use the same configuration settings
   - Download the extracted secret file

### Configuration Options

- **Key**: Required string for steganography operations
- **Use Encryption**: Optional Vigenere cipher encryption
- **Use Random Start**: Random starting position for embedding
- **LSB Bits**: Number of LSB bits to use (1-4, affects capacity vs quality)
