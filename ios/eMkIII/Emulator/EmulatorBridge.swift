import Foundation
import Emulator // The gomobile-generated framework

/// Swift wrapper for the Go emulator
class EmulatorBridge {
    /// Screen dimensions
    static let screenWidth = 256
    static let maxScreenHeight = 224

    /// Audio sample rate
    static let audioSampleRate = 48000

    /// Whether emulator is loaded
    private(set) var isLoaded = false

    /// Current active height (192 or 224)
    var activeHeight: Int {
        return EmuiosFrameHeight()
    }

    /// Current region (0=NTSC, 1=PAL)
    var region: Int {
        return EmuiosRegion()
    }

    /// FPS for current region
    var fps: Int {
        return EmuiosGetFPS(region)
    }

    /// Whether left column blanking is enabled (for border cropping)
    var leftColumnBlankEnabled: Bool {
        return EmuiosLeftBlank()
    }

    // MARK: - Initialization

    /// Load a ROM from file path with auto-detected region
    func loadROM(path: String) -> Bool {
        let regionCode = EmuiosDetectRegionFromPath(path)
        let success = EmuiosInitFromPath(path, regionCode)
        isLoaded = success
        return success
    }

    /// Load a ROM from file path with specified region
    func loadROM(path: String, region: Int) -> Bool {
        let success = EmuiosInitFromPath(path, region)
        isLoaded = success
        return success
    }

    /// Unload the current ROM
    func unload() {
        EmuiosClose()
        isLoaded = false
    }

    // MARK: - Frame Execution

    /// Run one frame of emulation
    func runFrame() {
        EmuiosRunFrame()
    }

    /// Get the framebuffer as RGBA data
    func getFrameBuffer() -> Data? {
        guard let data = EmuiosGetFrameData() else { return nil }
        return Data(data)
    }

    // MARK: - Audio

    /// Get audio samples as int16 stereo PCM data
    func getAudioSamples() -> Data? {
        guard let data = EmuiosGetAudioData() else { return nil }
        return Data(data)
    }

    // MARK: - Input

    /// Set player 1 controller state
    func setInput(up: Bool, down: Bool, left: Bool, right: Bool, button1: Bool, button2: Bool) {
        EmuiosSetInput(up, down, left, right, button1, button2)
    }

    /// Trigger SMS pause button (NMI)
    func triggerPause() {
        EmuiosSetPause()
    }

    // MARK: - Save States

    /// Create a save state
    func serialize() -> Data? {
        guard EmuiosSaveState() else { return nil }

        let len = EmuiosStateLen()
        guard len > 0 else { return nil }

        var bytes = [UInt8](repeating: 0, count: len)
        for i in 0..<len {
            bytes[i] = UInt8(EmuiosStateByte(i))
        }
        return Data(bytes)
    }

    /// Load a save state
    func deserialize(data: Data) -> Bool {
        return EmuiosLoadState(data)
    }

    // MARK: - SRAM (Battery Save)

    /// Get cartridge RAM (SRAM) data
    func getCartRAM() -> Data? {
        EmuiosPrepareSRAM()
        let len = EmuiosSRAMLen()
        guard len > 0 else { return nil }

        var bytes = [UInt8](repeating: 0, count: len)
        for i in 0..<len {
            bytes[i] = UInt8(EmuiosSRAMByte(i))
        }
        return Data(bytes)
    }

    /// Set cartridge RAM (SRAM) data
    func setCartRAM(data: Data) {
        EmuiosLoadSRAM(data)
    }

    // MARK: - Static Helpers

    /// Calculate CRC32 of ROM file
    /// Returns nil on error
    static func crc32(ofPath path: String) -> UInt32? {
        let result = EmuiosGetCRC32FromPath(path)
        if result < 0 {
            return nil
        }
        return UInt32(result)
    }

    /// Detect region for ROM file (0=NTSC, 1=PAL)
    static func detectRegion(path: String) -> Int {
        return EmuiosDetectRegionFromPath(path)
    }

    /// Get FPS for region code
    static func fps(for regionCode: Int) -> Int {
        return EmuiosGetFPS(regionCode)
    }

    /// Extract ROM from archive and store as {CRC32}.sms
    /// - Parameters:
    ///   - srcPath: Path to source file (archive or raw ROM)
    ///   - destDir: Directory to store extracted ROM
    /// - Returns: Tuple of (crc32 hex string, original filename) on success, nil on error
    static func extractAndStoreROM(srcPath: String, destDir: String) -> (crc32: String, filename: String)? {
        var error: NSError?
        guard let result = EmuiosExtractAndStoreROM(srcPath, destDir, &error) else {
            if let error = error {
                Log.romImport.error("Extract failed: \(error.localizedDescription)")
            }
            return nil
        }
        return (crc32: result.crc32, filename: result.filename)
    }
}
