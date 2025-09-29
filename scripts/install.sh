#!/usr/bin/env bash
set -e

GITHUB_REPO="tkcrm/pgxgen"
TARGET_BINARY="pgxgen"
INSTALL_DIR="/usr/local/bin"

# Determine the operating system
OS=$(uname)
if [ "$OS" == "Darwin" ]; then
  PLATFORM="darwin"
elif [ "$OS" == "Linux" ]; then
  PLATFORM="linux"
else
  echo "Unsupported OS: $OS"
  exit 1
fi

# Determine the architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)
    ARCH="amd64"
    ;;
  arm64|aarch64)
    ARCH="arm64"
    ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "Detected platform: $PLATFORM, architecture: $ARCH"

# Fetch the latest release information from GitHub API
API_URL="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
RELEASE_INFO=$(curl --silent "$API_URL")

# Ensure jq is installed
if ! command -v jq &>/dev/null; then
  echo "Error: jq is not installed. Please install jq and try again."
  exit 1
fi

# Find the asset matching the platform and architecture
ASSET_NAME=$(echo "$RELEASE_INFO" | jq -r --arg platform "$PLATFORM" --arg arch "$ARCH" '
  .assets[] | select(.name | test($platform) and test($arch)) | .name
')

if [ -z "$ASSET_NAME" ]; then
  echo "No archive found for platform $PLATFORM and architecture $ARCH"
  exit 1
fi

echo "Found archive: $ASSET_NAME"

# Get the download URL for the asset
ASSET_URL=$(echo "$RELEASE_INFO" | jq -r --arg asset "$ASSET_NAME" '
  .assets[] | select(.name == $asset) | .browser_download_url
')

if [ -z "$ASSET_URL" ]; then
  echo "Failed to obtain download URL."
  exit 1
fi

echo "Downloading binary from $ASSET_URL"
curl -L --silent -o "$ASSET_NAME" "$ASSET_URL"

# Extract the archive
echo "Extracting archive..."
tar -xzf "$ASSET_NAME" || {
  echo "Failed to extract archive."
  exit 1
}

# Remove the downloaded archive after extraction
echo "Removing archive $ASSET_NAME..."
rm "$ASSET_NAME"

# The extracted directory name matches the archive name without .tar.gz
EXTRACTED_DIR="${ASSET_NAME%.tar.gz}"
BINARY_PATH="${EXTRACTED_DIR}/${TARGET_BINARY}"

if [ ! -f "$BINARY_PATH" ]; then
  echo "Error: Binary file $BINARY_PATH not found after extraction."
  exit 1
fi

echo "Found binary at: $BINARY_PATH"

# Move the binary from subdirectory to current directory
mv "$BINARY_PATH" "$TARGET_BINARY"

# Remove the extracted directory
rm -rf "$EXTRACTED_DIR"

# Ensure the binary is executable
if [ ! -x "$TARGET_BINARY" ]; then
  echo "Error: Binary file is not executable."
  exit 1
fi

# Check install directory
if [ ! -d "$INSTALL_DIR" ]; then
  echo "Directory $INSTALL_DIR does not exist."
  exit 1
fi

if [ -f "$INSTALL_DIR/$TARGET_BINARY" ]; then
  echo "Removing existing binary from $INSTALL_DIR..."
  sudo rm "$INSTALL_DIR/$TARGET_BINARY"
fi

echo "Moving binary to $INSTALL_DIR..."
sudo mv "$TARGET_BINARY" "$INSTALL_DIR/$TARGET_BINARY"

echo "Making the binary executable..."
sudo chmod +x "$INSTALL_DIR/$TARGET_BINARY"

echo "Installation complete. You can now run $TARGET_BINARY from the terminal."
