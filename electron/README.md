# New API Electron Desktop App

This directory contains the Electron wrapper for New API, providing a native desktop application with system tray support. **本项目只维护 Linux 产物**（AppImage + deb）；Windows/macOS 打包已移除。

## Prerequisites

### 1. Go Binary (Required)
The Electron app requires the compiled Go binary to function. You have two options:

**Option A: Use existing binary (without Go installed)**
```bash
# 用 Release 里的 Linux 静态二进制
cp ../new-api-<版本> ../new-api
```

**Option B: Build from source (requires Go)**
TODO

### 2. Electron Dependencies
```bash
cd electron
npm install
```

## Development

Start the backend, the frontend, and Electron in separate terminals:
```bash
# Repository root
go run main.go

# Repository root
make dev-web

# electron/
npm run dev-app
```

This will:
- Use the Go backend on port 3000
- Use the Rsbuild frontend development server on port 5173
- Open an Electron window with DevTools enabled
- Create a system tray icon
- Store database in `../data/new-api.db`

## Building for Production

### Quick Build
```bash
# From electron/, build the frontend, Go binary, and desktop package
./build.sh

# Or package an existing binary for the current platform
npm run build

# Platform-specific builds
npm run build:mac    # Creates .dmg and .zip
npm run build:win    # Creates .exe installer
npm run build:linux  # Creates .AppImage and .deb
```

### Build Output
- Built applications are in `electron/dist/`
- Linux: `.AppImage`（免安装）与 `.deb`（安装包）
- Linux: `.AppImage` and `.deb`

## Configuration

### Port
Default port is 3000. To change, edit `main.js`:
```javascript
const PORT = 3000; // Change to desired port
```

### Database Location
- **Development**: `../data/new-api.db` (project directory)
- **Production**:
  - Linux: 见 electron-builder 的 `userData` 目录（默认 `~/.config/New-API-App/`）
  - Linux: `~/.config/New API/data/`
