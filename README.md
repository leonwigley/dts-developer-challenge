# ⚖️ HMCTS DTS Developer Challenge

## 🚀 Usage

> Before you start, make sure you have the following installed:

Go (version 1.20 or higher): https://go.dev/dl/

Air (live reloading tool): https://github.com/air-verse/air

Run the following commands to start this project:

```bash
$ git clone https://github.com/leonwigley/dts-developer-challenge.git
$ cd dts-developer-challenge/
$ go mod tidy
$ air
```

Go to ```http://localhost:3000/``` in your browser.

### 🛠️ Tech Stack 
- **Server**: Go
- **Database**: SQLite
- **Front-end**: HTMX

<img src="https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fjuststickers.in%2Fwp-content%2Fuploads%2F2016%2F07%2Fgo-programming-language.png&f=1&nofb=1&ipt=7ac7a84b65a03543419662e947e8f6fc575353367542fe982a2417cf48d4cdad" alt="Go gopher mascot" height="50px" width="auto"><img src="https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fassets.stickpng.com%2Fimages%2F62c46cd2a75b8945b1696713.png&f=1&nofb=1&ipt=469f09cd390a354179eaf8704603d9754186300a873f6ca655e8eccc3bc5ea35" alt="SQLite Logo" height="50" style="height: 50px; width: auto;"><img src="https://external-content.duckduckgo.com/iu/?u=https%3A%2F%2Fwww.saaspegasus.com%2Fstatic%2Fimages%2Fpegasus%2Fhtmx-icon.png&f=1&nofb=1&ipt=62a23fc13ab6a205f1077bf891c9fa166f40dbb32010cf3f2482ee0c4e44adca" alt="HTMX Logo" height="50" style="height: 50px; width: auto;">

## ✅ Challenge Criteria
- API endpoint (JSON): ```localhost:3000/api/tasks```
- Stores tasks in an SQLite database
- Create a task with the following properties:
  - Title
  - Description
  - Status
  - Due date/time (default format)
- Retrieve a task by ID: ```localhost:3000/api/tasks/:id```
- Update task status (e.g. mark as completed): **press Complete**, or use:  ```curl -X PUT http://localhost:3000/api/tasks/:id -d 'status=completed'```
- Delete a task by id: **press Delete**, or use: ```curl -X DELETE localhost:3000/api/tasks/:id```
- Includes validation and error handling
- Includes unit tests

---

__Thank you for considering my application 😊__