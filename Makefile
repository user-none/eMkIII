.PHONY: all clean emkiii macos libretro icons iconset

# Output directories
BUILD_DIR := build
ICONSET_DIR := $(BUILD_DIR)/icon.iconset
APP_NAME := eMkIII
APP_BUNDLE := $(BUILD_DIR)/$(APP_NAME).app

# Source files
ICON_MASTER := assets/icon.png
ICON_ICNS := $(BUILD_DIR)/icon.icns
IOS_ICON := ios/eMkIII/Resources/Assets.xcassets/AppIcon.appiconset/icon.png

# SDL3 library location (Homebrew)
UNAME_M := $(shell uname -m)
ifeq ($(UNAME_M),arm64)
    SDL3_LIB := /opt/homebrew/lib/libSDL3.dylib
else
    SDL3_LIB := /usr/local/lib/libSDL3.dylib
endif

# Build all targets
all: emkiii

# Build the standalone binary
emkiii:
	go build -o $(BUILD_DIR)/emkiii .

# Build macOS .app bundle
macos: emkiii icons
	@echo "Creating $(APP_NAME).app bundle..."
	@mkdir -p "$(APP_BUNDLE)/Contents/MacOS"
	@mkdir -p "$(APP_BUNDLE)/Contents/Resources"
	@mkdir -p "$(APP_BUNDLE)/Contents/Frameworks"
	@cp $(BUILD_DIR)/emkiii "$(APP_BUNDLE)/Contents/MacOS/"
	@cp $(ICON_ICNS) "$(APP_BUNDLE)/Contents/Resources/icon.icns"
	@cp assets/macos_info.plist "$(APP_BUNDLE)/Contents/Info.plist"
	@echo "APPL????" > "$(APP_BUNDLE)/Contents/PkgInfo"
	@if [ -f "$(SDL3_LIB)" ]; then \
		cp "$(SDL3_LIB)" "$(APP_BUNDLE)/Contents/Frameworks/"; \
		install_name_tool -id "@executable_path/../Frameworks/libSDL3.dylib" \
			"$(APP_BUNDLE)/Contents/Frameworks/libSDL3.dylib"; \
		echo "Bundled SDL3 library"; \
	else \
		echo "Warning: SDL3 not found at $(SDL3_LIB), app may not be portable"; \
	fi
	@echo "Signing app bundle..."
	@codesign --force --sign - --deep "$(APP_BUNDLE)"
	@echo "Created $(APP_BUNDLE)"

# Build libretro core
libretro:
	go build -tags libretro -buildmode=c-shared -o $(BUILD_DIR)/emkiii_libretro.dylib ./bridge/libretro/

# Generate icons from master PNG
icons: $(ICON_ICNS) $(IOS_ICON)

# iOS icon (just copy the 1024x1024 master)
$(IOS_ICON): $(ICON_MASTER)
	@echo "Copying icon to iOS..."
	@cp $(ICON_MASTER) $(IOS_ICON)

$(ICON_ICNS): $(ICON_MASTER) | $(BUILD_DIR)
	@echo "Generating macOS icon..."
	@mkdir -p $(ICONSET_DIR)
	@sips -z 16 16 $(ICON_MASTER) --out $(ICONSET_DIR)/icon_16x16.png
	@sips -z 32 32 $(ICON_MASTER) --out $(ICONSET_DIR)/icon_16x16@2x.png
	@sips -z 32 32 $(ICON_MASTER) --out $(ICONSET_DIR)/icon_32x32.png
	@sips -z 64 64 $(ICON_MASTER) --out $(ICONSET_DIR)/icon_32x32@2x.png
	@sips -z 128 128 $(ICON_MASTER) --out $(ICONSET_DIR)/icon_128x128.png
	@sips -z 256 256 $(ICON_MASTER) --out $(ICONSET_DIR)/icon_128x128@2x.png
	@sips -z 256 256 $(ICON_MASTER) --out $(ICONSET_DIR)/icon_256x256.png
	@sips -z 512 512 $(ICON_MASTER) --out $(ICONSET_DIR)/icon_256x256@2x.png
	@sips -z 512 512 $(ICON_MASTER) --out $(ICONSET_DIR)/icon_512x512.png
	@sips -z 1024 1024 $(ICON_MASTER) --out $(ICONSET_DIR)/icon_512x512@2x.png
	@iconutil -c icns $(ICONSET_DIR) -o $(ICON_ICNS)
	@rm -rf $(ICONSET_DIR)
	@echo "Created $(ICON_ICNS)"

$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

clean:
	rm -rf $(BUILD_DIR)
