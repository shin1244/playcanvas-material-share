# PlayCanvas Material Share

A automation tool using Go and Playwright to synchronize materials across multiple PlayCanvas projects, triggered by a Tampermonkey UserScript.


---

## 💡 The Problem (Why This Exists)

This tool was built to solve a simple, tedious problem:

When you use the same material (e.g., "Standard Glass") across multiple projects, what happens when you need to change a detailed property?

You have to manually copy and paste the new values (like `reflectivity` or `tiling`) to **every single project**. This is a hassle and highly prone to human error. This tool automates that entire synchronization process.

---

## 🔧 How It Works (Architecture)

This project uses a client/server architecture to bridge the gap between your browser and local automation.


1.  **Client (Tampermonkey UserScript):**
    * You install `PlayCanvas Sync Metarial.js` into your browser (via Tampermonkey).
    * This script adds a "Sync Metarial" button to your PlayCanvas editor UI.
    * When you select materials and click the button, the script scrapes their data.
    * It converts texture asset IDs into simple **texture names** (e.g., `PBR_Texture.PNG`).
    * It sends this data to your local Go server (`http://localhost:8080/sync-material`).

2.  **Backend (Go Server):**
    * The Go server (`main.go`) runs locally, listening for the request.
    * It reads `url.txt` to get a list of all **target scene URLs** you want to sync.
    * It launches a **Playwright** browser instance, logging into PlayCanvas (using a saved `auth.json` session for speed).

3.  **Automation (Playwright):**
    * For **each URL** in `url.txt`, the server:
        * Navigates to the target scene.
        * Waits for the editor and assets to be fully loaded.
        * Injects a script that finds the material by its **name**.
        * It then finds the required textures in that project by their **names** and gets their new asset IDs.
        * It updates (syncs) the material data.

---

## 🚀 How to Use (Usage)

### 1. Prerequisites
* [Go (Golang)](https://go.dev/doc/install) (1.18 or newer).
* A browser extension for UserScripts, such as [Tampermonkey](https://www.tampermonkey.net/).
* A PlayCanvas account.

### 2. Backend Setup (Go Server)
1.  Clone this repository:
    ```bash
    git clone https://github.com/shin1244/playcanvas-material-share.git
    cd playcanvas-material-share
    ```
2.  Install Go dependencies:
    ```bash
    go mod tidy
    ```
3.  Create a `url.txt` file in the same directory.
4.  Add all your **target scene URLs** to this file, one URL per line.
    ```
    https://playcanvas.com/editor/scene/111111
    https://playcanvas.com/editor/scene/222222
    https://playcanvas.com/editor/scene/333333
    ```

### 3. Client Setup (Tampermonkey)
1.  Open your Tampermonkey dashboard in your browser.
2.  Create a new script.
3.  Copy the entire contents of `PlayCanvas Sync Metarial.js` and paste it into the Tampermonkey editor.
4.  Save the script and ensure it is enabled.

### 4. Running the Sync
1.  **Start the backend server.** Run this command in your terminal and leave it running:
    ```bash
    go run main.go
    ```
2.  **First-time login:** The server will launch a (non-headless) browser. Manually log in to PlayCanvas. The server will save your session to `auth.json` and then continue.
3.  **Go to your "Source" project** in PlayCanvas (the one you want to copy *from*).
4.  Select one or more materials in the asset panel.
5.  Click the **"Sync Metarial"** button that now appears in the bottom-right corner.
6.  Look at your Go server's terminal. You will see it navigating to each URL in `url.txt` and applying the material changes.

---

## ⚠️ Important Notes & Limitations
* **Name-Based Matching:** This script syncs materials and textures by their **NAME**. The material `Metal_A` in the source project will only update `Metal_A` in the target projects. The same applies to texture names.
* **Security:** The Go server runs locally and only accepts requests from `playcanvas.com` via CORS.
* **Session File:** Do not share your `auth.json` file. It contains your login session.
* **No Concurrency:** While concurrent programming (e.g., Goroutines) was considered for faster synchronization, it was intentionally not used. The tool syncs projects sequentially to comply with PlayCanvas [API rate limits.](https://forum.playcanvas.com/t/important-introducing-rate-limits-to-playcanvas/33220)

## 🏁 Conclusion
This tool provides a powerful automated workflow for maintaining material consistency across many PlayCanvas projects. By bridging the editor UI (with a UserScript) to a powerful backend (Go + Playwright), it solves a common development bottleneck and saves significant time.
