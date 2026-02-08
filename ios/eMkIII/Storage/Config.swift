import Foundation

/// Application configuration
struct Config: Codable {
    var version: Int = 1
    var video: VideoConfig = VideoConfig()
    var audio: AudioConfig = AudioConfig()
    var library: LibraryConfig = LibraryConfig()

    struct VideoConfig: Codable {
        var cropBorder: Bool = false
    }

    struct AudioConfig: Codable {
        var mute: Bool = false
    }

    struct LibraryConfig: Codable {
        var viewMode: ViewMode = .icon
        var sortBy: Library.SortMethod = .title

        enum ViewMode: String, Codable, CaseIterable {
            case icon
            case list

            var displayName: String {
                switch self {
                case .icon: return "Icons"
                case .list: return "List"
                }
            }
        }
    }

    // MARK: - Persistence

    static func load() -> Config? {
        let path = StoragePaths.configPath
        guard FileManager.default.fileExists(atPath: path),
              let data = FileManager.default.contents(atPath: path) else {
            return nil
        }

        do {
            let decoder = JSONDecoder()
            return try decoder.decode(Config.self, from: data)
        } catch {
            Log.storage.error("Failed to decode config: \(error.localizedDescription)")
            return nil
        }
    }

    func save() {
        do {
            // Ensure directory exists
            let dir = URL(fileURLWithPath: StoragePaths.configPath).deletingLastPathComponent()
            try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)

            let encoder = JSONEncoder()
            encoder.outputFormatting = [.prettyPrinted, .sortedKeys]
            let data = try encoder.encode(self)
            try data.write(to: URL(fileURLWithPath: StoragePaths.configPath))
        } catch {
            Log.storage.error("Failed to save config: \(error.localizedDescription)")
        }
    }
}
