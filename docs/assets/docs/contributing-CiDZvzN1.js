import{c as s,j as e,R as n}from"../styles-CUNhQwI0.js";import{D as r}from"../DocsLayout-GqKUjnoJ.js";function t(){return e.jsxs(r,{active:"contributing",children:[e.jsxs("header",{children:[e.jsx("h1",{children:"Contributing"}),e.jsx("p",{className:"mt-2",children:"Thanks for your interest in improving Scriberr! Here’s how to get set up and contribute."})]}),e.jsxs("section",{children:[e.jsx("h2",{children:"Guidelines"}),e.jsxs("ul",{className:"list-disc pl-5 mt-2 space-y-1",children:[e.jsx("li",{children:"Open an issue first for large changes to discuss scope and approach."}),e.jsx("li",{children:"Keep pull requests focused and small; write clear descriptions."}),e.jsxs("li",{children:["Use conventional, imperative commit messages (e.g., ",e.jsx("code",{children:"add docs sidebar link"}),", ",e.jsx("code",{children:"fix queue status endpoint"}),")."]}),e.jsxs("li",{children:["Follow coding styles: run ",e.jsx("code",{children:"go fmt ./..."}),", ",e.jsx("code",{children:"go vet ./..."})," and ",e.jsx("code",{children:"npm run lint"})," in the frontend."]}),e.jsxs("li",{children:["Add tests where appropriate (Go tests live under ",e.jsx("code",{children:"tests/"})," or next to packages)."]}),e.jsx("li",{children:"Update docs (README, swagger) when you change API shapes."})]})]}),e.jsxs("section",{children:[e.jsx("h2",{children:"Prerequisites"}),e.jsxs("ul",{className:"list-disc pl-5 mt-2 space-y-1",children:[e.jsx("li",{children:"Node.js 18+ and npm"}),e.jsx("li",{children:"Go 1.24+"}),e.jsxs("li",{children:["Python 3.11+ and ",e.jsx("a",{href:"https://docs.astral.sh/uv/",target:"_blank",rel:"noopener noreferrer",children:"uv"})," (for transcription features)"]})]}),e.jsx("div",{className:"bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2 overflow-x-auto",children:e.jsx("pre",{children:`# macOS (Homebrew)
brew install node go python
curl -LsSf https://astral.sh/uv/install.sh | sh

# Verify
node -v
npm -v
go version
python3 --version
uv --version`})})]}),e.jsxs("section",{children:[e.jsx("h2",{children:"Build and run locally"}),e.jsx("h3",{className:"mt-2",children:"Backend (dev)"}),e.jsx("div",{className:"bg-gray-50 rounded-lg p-4 font-mono text-sm mt-1 overflow-x-auto",children:e.jsx("pre",{children:`# Copy and edit environment
cp -n .env.example .env || true

# Run API server
go run cmd/server/main.go`})}),e.jsx("h3",{className:"mt-4",children:"Frontend (dev)"}),e.jsx("div",{className:"bg-gray-50 rounded-lg p-4 font-mono text-sm mt-1 overflow-x-auto",children:e.jsx("pre",{children:`cd web/frontend
npm ci
npm run dev`})}),e.jsx("h3",{className:"mt-4",children:"Full build (embed UI)"}),e.jsx("p",{className:"mt-1",children:"Use the build script to bundle the React app and compile the Go binary with embedded assets."}),e.jsx("div",{className:"bg-gray-50 rounded-lg p-4 font-mono text-sm mt-1 overflow-x-auto",children:e.jsx("pre",{children:`# From repo root
chmod +x ./build.sh
./build.sh

# Run the server
./scriberr`})})]}),e.jsxs("section",{children:[e.jsx("h2",{children:"Testing"}),e.jsx("div",{className:"bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2 overflow-x-auto",children:e.jsx("pre",{children:`# Run all Go tests with verbose output
go test ./... -v

# Or target test suites
go test ./tests -run TestAPITestSuite -v

# Lint frontend
cd web/frontend && npm run lint`})})]}),e.jsxs("section",{children:[e.jsx("h2",{children:"Submitting changes"}),e.jsxs("ul",{className:"list-disc pl-5 mt-2 space-y-1",children:[e.jsxs("li",{children:["Create a feature branch from ",e.jsx("code",{children:"main"}),"."]}),e.jsx("li",{children:"Ensure CI passes locally: build, test, lint."}),e.jsx("li",{children:"Open a pull request with a clear summary and screenshots/GIFs for UI changes."}),e.jsxs("li",{children:["Link issues (e.g., ",e.jsx("code",{children:"Closes #123"}),") when applicable."]})]})]})]})}const i=s(document.getElementById("root"));i.render(e.jsx(n.StrictMode,{children:e.jsx(t,{})}));
