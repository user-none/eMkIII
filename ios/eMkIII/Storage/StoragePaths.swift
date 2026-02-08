import Foundation

/// Storage paths for app data
enum StoragePaths {
    /// Documents directory (visible in Files app)
    static var documentsDirectory: URL {
        FileManager.default.urls(for: .documentDirectory, in: .userDomainMask)[0]
    }

    /// Application Support directory (hidden from user)
    static var appSupportDirectory: URL {
        FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask)[0]
    }

    // MARK: - Documents paths

    /// ROMs directory in Documents
    static var romsDirectory: URL {
        documentsDirectory.appendingPathComponent("roms", isDirectory: true)
    }

    /// Library JSON file path
    static var libraryPath: String {
        documentsDirectory.appendingPathComponent("library.json").path
    }

    // MARK: - Application Support paths

    /// Config JSON file path
    static var configPath: String {
        appSupportDirectory.appendingPathComponent("config.json").path
    }

    /// Metadata directory
    static var metadataDirectory: URL {
        appSupportDirectory.appendingPathComponent("metadata", isDirectory: true)
    }

    /// RDB file path
    static var rdbPath: String {
        metadataDirectory.appendingPathComponent("sms.rdb").path
    }

    /// Saves directory
    static var savesDirectory: URL {
        appSupportDirectory.appendingPathComponent("saves", isDirectory: true)
    }

    /// Artwork directory
    static var artworkDirectory: URL {
        appSupportDirectory.appendingPathComponent("artwork", isDirectory: true)
    }

    // MARK: - Per-game paths

    /// Save directory for a specific game
    static func saveDirectory(for crc: String) -> URL {
        savesDirectory.appendingPathComponent(crc, isDirectory: true)
    }

    /// Resume state path for a game
    static func resumeStatePath(for crc: String) -> String {
        saveDirectory(for: crc).appendingPathComponent("resume.state").path
    }

    /// SRAM path for a game
    static func sramPath(for crc: String) -> String {
        saveDirectory(for: crc).appendingPathComponent("sram.bin").path
    }

    /// Artwork path for a game
    static func artworkPath(for crc: String) -> String {
        artworkDirectory.appendingPathComponent("\(crc).png").path
    }

    // MARK: - Directory creation

    /// Ensures all required directories exist
    static func ensureDirectoriesExist() throws {
        let fm = FileManager.default
        let directories = [
            romsDirectory,
            metadataDirectory,
            savesDirectory,
            artworkDirectory
        ]

        for dir in directories {
            if !fm.fileExists(atPath: dir.path) {
                try fm.createDirectory(at: dir, withIntermediateDirectories: true)
            }
        }
    }

    /// Ensures save directory exists for a game
    static func ensureSaveDirectoryExists(for crc: String) throws {
        let dir = saveDirectory(for: crc)
        if !FileManager.default.fileExists(atPath: dir.path) {
            try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        }
    }
}
