# ⏱️ ponto-mais-autocomplete

Automate time adjustments in the PontoMais system via command line!

## ✨ About the Project

This project was developed to make it easier to adjust time records in the PontoMais system, allowing you to fetch absences and automatically adjust work hours via API, in a practical and fast way.

## 🚀 Technologies Used

- [Go (Golang)](https://golang.org/)
- REST API consumption (HTTP/JSON)

## ⚙️ Features

- Automatic query of time records (absences)
- Automatic adjustment of work hours via API
- Smart filters to avoid duplicate requests
- Command line execution

## 📦 Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/your-user/ponto-mais-autocomplete.git
   cd ponto-mais-autocomplete
   ```
2. **Install dependencies:**
   ```bash
   go mod download
   ```
3. **Build (optional):**
   ```bash
   go build -o ponto-mais-autocomplete main.go
   ```

## 🕹️ How to Use

You can run the program in two ways:

### 1. With access tokens (recommended for automation)
```bash
go run main.go --access-token=YOUR_ACCESS_TOKEN --uid=YOUR_UID --client=YOUR_CLIENT
```

### 2. With email and password (if implemented in your project)
```bash
go run main.go --email=YOUR_EMAIL --password=YOUR_PASSWORD
```

> **Tip:** You can compile and run the binary directly:
> ```bash
> ./ponto-mais-autocomplete --access-token=YOUR_ACCESS_TOKEN --uid=YOUR_UID --client=YOUR_CLIENT
> ```

## 🔑 Required Parameters

- `--access-token`: PontoMais API access token
- `--uid`: Registered user email
- `--client`: Application Client ID

**Or:**
- `--email`: User email
- `--password`: User password

## 📋 Usage Example

```bash
go run main.go --access-token=abc123 --uid=user@company.com --client=xyz789
```

## 📝 License

This project is licensed under the GNU License. See the [LICENSE](LICENSE) file for more details.

## 🤝 Contribute!

Contributions are welcome! Feel free to open issues or submit pull requests.

---

Made with 💙 to make your time tracking easier!