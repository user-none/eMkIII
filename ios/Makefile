# Makefile for building the iOS Swift app

.PHONY: all framework build clean

# Default target
all: framework

# Generate Go framework using gomobile bind
framework:
	@echo "Generating Emulator.xcframework..."
	GOFLAGS="-mod=mod" gomobile bind -target ios -o Frameworks/Emulator.xcframework ../bridge/ios
	@echo "Framework generated at Frameworks/Emulator.xcframework"

# Build the iOS app (requires Xcode)
build: framework
	@echo "Building iOS app..."
	xcodebuild -project eMkIII.xcodeproj -scheme eMkIII -destination 'generic/platform=iOS' build

# Build for simulator
simulator: framework
	@echo "Building for iOS Simulator..."
	xcodebuild -project eMkIII.xcodeproj -scheme eMkIII -destination 'platform=iOS Simulator,name=iPhone 15' build

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf Frameworks/Emulator.xcframework
	xcodebuild -project eMkIII.xcodeproj -scheme eMkIII clean 2>/dev/null || true
	rm -rf ~/Library/Developer/Xcode/DerivedData/eMkIII-*

# Help
help:
	@echo "Available targets:"
	@echo "  framework  - Generate Emulator.xcframework from Go code"
	@echo "  build      - Build iOS app for device"
	@echo "  simulator  - Build iOS app for simulator"
	@echo "  clean      - Clean build artifacts"
	@echo ""
	@echo "Prerequisites:"
	@echo "  - Go 1.21+"
	@echo "  - gomobile (go install golang.org/x/mobile/cmd/gomobile@latest)"
	@echo "  - Xcode 15+"
