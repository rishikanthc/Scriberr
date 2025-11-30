import{c as s,j as e,R as r}from"../styles-CUNhQwI0.js";import{D as a}from"../DocsLayout-GqKUjnoJ.js";function t(){return e.jsxs(a,{active:"installation",children:[e.jsxs("header",{children:[e.jsx("h1",{children:"Installation"}),e.jsx("p",{className:"mt-2",children:"Get Scriberr running on your system in a few minutes."})]}),e.jsxs("section",{children:[e.jsx("h2",{children:"Install with Homebrew (macOS & Linux)"}),e.jsxs("p",{className:"mt-2",children:["The easiest way to install Scriberr is using Homebrew. If you don’t have Homebrew installed,",e.jsx("a",{href:"https://brew.sh",target:"_blank",rel:"noopener noreferrer",className:"ml-1",children:"get it here first"}),"."]}),e.jsx("div",{className:"bg-gray-50 rounded-lg p-4 font-mono text-sm mt-3",children:e.jsxs("div",{className:"text-gray-800",children:[e.jsx("span",{className:"text-green-600",children:"# Add the Scriberr tap"}),e.jsx("br",{}),"brew tap rishikanthc/scriberr",e.jsx("br",{}),e.jsx("br",{}),e.jsx("span",{className:"text-green-600",children:"# Install Scriberr (automatically installs UV dependency)"}),e.jsx("br",{}),"brew install scriberr",e.jsx("br",{}),e.jsx("br",{}),e.jsx("span",{className:"text-green-600",children:"# Start the server"}),e.jsx("br",{}),"scriberr"]})}),e.jsxs("p",{className:"mt-3",children:["Open ",e.jsx("code",{className:"bg-gray-100 px-1 rounded",children:"http://localhost:8080"})," in your browser."]}),e.jsx("h3",{className:"mt-8",children:"Configuration"}),e.jsxs("p",{className:"mt-2",children:["Scriberr works out of the box. To customize settings, create a ",e.jsx("code",{className:"bg-gray-100 px-1 rounded",children:".env"})," file:"]}),e.jsx("div",{className:"bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2",children:e.jsxs("div",{className:"text-gray-800",children:[e.jsx("span",{className:"text-green-600",children:"# Server settings"}),e.jsx("br",{}),"HOST=localhost",e.jsx("br",{}),"PORT=8080",e.jsx("br",{}),e.jsx("br",{}),e.jsx("span",{className:"text-green-600",children:"# Data storage (optional)"}),e.jsx("br",{}),"DATABASE_PATH=./data/scriberr.db",e.jsx("br",{}),"UPLOAD_DIR=./data/uploads",e.jsx("br",{}),"WHISPERX_ENV=./data/whisperx-env",e.jsx("br",{}),e.jsx("br",{}),e.jsx("span",{className:"text-green-600",children:"# Custom paths (if needed)"}),e.jsx("br",{}),"UV_PATH=/custom/path/to/uv"]})}),e.jsx("h3",{className:"mt-8",children:"Troubleshooting"}),e.jsxs("div",{className:"space-y-3 mt-2",children:[e.jsxs("div",{children:[e.jsx("strong",{children:"Command not found"}),e.jsxs("p",{className:"mt-1",children:["Make sure the binary is in your PATH or run it with the full path: ",e.jsx("code",{className:"bg-gray-100 px-1 rounded",children:"./scriberr"})]})]}),e.jsxs("div",{children:[e.jsx("strong",{children:"Transcription not working"}),e.jsx("p",{className:"mt-1",children:"Ensure Python 3.11+ and UV are installed. Check logs on start for Python environment issues."})]}),e.jsxs("div",{children:[e.jsx("strong",{children:"Port already in use"}),e.jsxs("p",{className:"mt-1",children:["Set a different port with ",e.jsx("code",{className:"bg-gray-100 px-1 rounded",children:"PORT=8081 scriberr"})," or add it to your .env file."]})]})]})]}),e.jsxs("section",{children:[e.jsx("h2",{className:"mt-12",children:"Install with Docker"}),e.jsx("p",{className:"mt-2",children:"Run Scriberr in a container with all dependencies included. We provide images for both CPU and NVIDIA GPU (CUDA) environments."}),e.jsx("h3",{className:"mt-4",children:"CPU Version (Standard)"}),e.jsxs("div",{className:"bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2 overflow-x-auto",children:[e.jsx("span",{className:"text-green-600",children:"# Run with Docker (data persisted in volume)"}),e.jsx("pre",{className:"mt-2",children:`docker run -d \\
  --name scriberr \\
  -p 8080:8080 \\
  -v scriberr_data:/app/data \\
  --restart unless-stopped \\
  ghcr.io/rishikanthc/scriberr:latest`})]}),e.jsx("h3",{className:"mt-8",children:"NVIDIA GPU Version (CUDA)"}),e.jsxs("p",{className:"mt-2",children:["For hardware acceleration, use the CUDA image. Requires ",e.jsx("a",{href:"https://docs.nvidia.com/datacenter/cloud-native/container-toolkit/install-guide.html",target:"_blank",rel:"noopener noreferrer",children:"NVIDIA Container Toolkit"})," to be installed on your host."]}),e.jsx("div",{className:"bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2 overflow-x-auto",children:e.jsx("pre",{className:"mt-2",children:`docker run -d \\
  --name scriberr-cuda \\
  --gpus all \\
  -p 8080:8080 \\
  -v scriberr_data:/app/data \\
  --restart unless-stopped \\
  ghcr.io/rishikanthc/scriberr:cuda-latest`})}),e.jsx("h3",{className:"mt-8",children:"Docker Compose"}),e.jsxs("p",{className:"mt-2",children:["Create a ",e.jsx("code",{className:"bg-gray-100 px-1 rounded",children:"docker-compose.yml"})," with the following:"]}),e.jsx("div",{className:"bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2 overflow-x-auto",children:e.jsx("pre",{children:`version: '3.9'
services:
  scriberr:
    # Use ghcr.io/rishikanthc/scriberr:cuda-latest for GPU support
    image: ghcr.io/rishikanthc/scriberr:latest
    container_name: scriberr
    ports:
      - "8080:8080"
    volumes:
      - scriberr_data:/app/data
    # Uncomment for GPU support
    # deploy:
    #   resources:
    #     reservations:
    #       devices:
    #         - driver: nvidia
    #           count: 1
    #           capabilities: [gpu]
    restart: unless-stopped

volumes:
  scriberr_data:`})}),e.jsx("div",{className:"bg-gray-50 rounded-lg p-4 font-mono text-sm mt-2",children:e.jsxs("div",{className:"text-gray-800",children:[e.jsx("span",{className:"text-green-600",children:"# Start the service"}),e.jsx("br",{}),"docker compose up -d"]})}),e.jsxs("p",{className:"mt-3",children:["Access the web interface at ",e.jsx("code",{className:"bg-gray-100 px-1 rounded",children:"http://localhost:8080"}),"."]})]}),e.jsx("section",{children:e.jsxs("p",{className:"mt-10",children:["To configure speaker diarization, see the ",e.jsx("a",{href:"/docs/configuration.html",children:"Configuration guide"}),"."]})})]})}const n=s(document.getElementById("root"));n.render(e.jsx(r.StrictMode,{children:e.jsx(t,{})}));
