import SwiftUI
import Combine

/// Represents the current screen in the app navigation
enum AppScreen: Equatable {
    case library
    case settings
    case detail(gameCRC: String)
    case gameplay(gameCRC: String, resume: Bool)
}

/// Observable app state shared across all views
@MainActor
class AppState: ObservableObject {
    // MARK: - Navigation

    @Published var currentScreen: AppScreen = .library

    // MARK: - Data

    @Published var library: Library
    @Published var config: Config

    // MARK: - Managers

    let rdbParser: RDBParser
    let artworkDownloader: ArtworkDownloader

    // MARK: - RDB State

    @Published var isRDBLoaded: Bool = false
    @Published var isRDBDownloading: Bool = false

    // MARK: - Artwork State

    /// Incremented when artwork is downloaded, triggers UI refresh
    @Published var artworkVersion: Int = 0

    // MARK: - Subscriptions

    private var cancellables = Set<AnyCancellable>()

    // MARK: - Initialization

    init() {
        // Load or create config
        if let loadedConfig = Config.load() {
            self.config = loadedConfig
        } else {
            self.config = Config()
        }

        // Load or create library
        if let loadedLibrary = Library.load() {
            self.library = loadedLibrary
        } else {
            self.library = Library()
        }

        // Initialize managers
        self.rdbParser = RDBParser()
        self.artworkDownloader = ArtworkDownloader()

        // Forward library changes to trigger view updates
        library.objectWillChange
            .receive(on: RunLoop.main)
            .sink { [weak self] _ in
                self?.objectWillChange.send()
            }
            .store(in: &cancellables)

        // Load RDB if available
        Task {
            await loadRDBIfAvailable()
        }
    }

    // MARK: - Navigation

    func navigateToLibrary() {
        currentScreen = .library
    }

    func navigateToSettings() {
        currentScreen = .settings
    }

    func navigateToDetail(gameCRC: String) {
        currentScreen = .detail(gameCRC: gameCRC)
    }

    func launchGame(gameCRC: String, resume: Bool) {
        currentScreen = .gameplay(gameCRC: gameCRC, resume: resume)
    }

    // MARK: - Library Management

    func saveLibrary() {
        library.save()
    }

    func saveConfig() {
        config.save()
    }

    func getGame(crc: String) -> GameEntry? {
        return library.games[crc]
    }

    func updateGameLastPlayed(crc: String) {
        if var game = library.games[crc] {
            game.lastPlayed = Date().timeIntervalSince1970
            library.games[crc] = game
            saveLibrary()
        }
    }

    // MARK: - RDB Management

    func loadRDBIfAvailable() async {
        let rdbPath = StoragePaths.rdbPath
        if FileManager.default.fileExists(atPath: rdbPath) {
            do {
                try rdbParser.load(from: rdbPath)
                isRDBLoaded = true
            } catch {
                Log.storage.error("Failed to load RDB: \(error.localizedDescription)")
            }
        } else {
            // Auto-download if not present
            await downloadRDB()
        }
    }

    func downloadRDB() async {
        guard !isRDBDownloading else { return }

        isRDBDownloading = true
        defer { isRDBDownloading = false }

        do {
            try await rdbParser.downloadAndLoad()
            isRDBLoaded = true
            // Update existing library entries with RDB metadata
            refreshLibraryMetadata()
        } catch {
            Log.network.error("Failed to download RDB: \(error.localizedDescription)")
        }
    }

    /// Update library entries with metadata from RDB
    private func refreshLibraryMetadata() {
        var updated = false
        for (crc, var game) in library.games {
            if let crc32 = UInt32(crc, radix: 16),
               let rdbGame = rdbParser.lookup(crc32: crc32) {
                game.name = rdbGame.name
                game.displayName = rdbGame.displayName
                game.region = rdbGame.region
                library.games[crc] = game
                updated = true
            }
        }
        if updated {
            saveLibrary()
        }
    }

    func lookupGame(crc32: UInt32) -> RDBGameInfo? {
        return rdbParser.lookup(crc32: crc32)
    }

    // MARK: - Artwork

    func downloadMissingArtwork() async {
        for (crc, game) in library.games {
            let artPath = StoragePaths.artworkPath(for: crc)
            if !FileManager.default.fileExists(atPath: artPath) {
                if await artworkDownloader.download(for: crc, gameName: game.name) {
                    artworkVersion += 1
                }
            }
        }
    }
}
