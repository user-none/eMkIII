import SwiftUI
import UniformTypeIdentifiers

/// Helper for loading game artwork
enum ArtworkLoader {
    static func loadImage(for crc: String) -> UIImage? {
        let artPath = StoragePaths.artworkPath(for: crc)
        return UIImage(contentsOfFile: artPath)
    }
}

/// Game library view showing all games
struct LibraryView: View {
    @EnvironmentObject var appState: AppState
    @State private var showingFilePicker = false
    @State private var showingSettings = false

    /// Allowed file types for ROM import
    private static let romContentTypes: [UTType] = [
        UTType(filenameExtension: "sms") ?? .data,
        .zip,
        .gzip,
        UTType(filenameExtension: "7z") ?? .data,
        UTType(filenameExtension: "rar") ?? .data
    ]

    /// Calculate grid columns based on available width (2-4 columns)
    private func gridColumns(for width: CGFloat) -> [GridItem] {
        let spacing: CGFloat = 16
        // Target column count based on width
        let columnCount: Int
        if width < 450 {
            columnCount = 2
        } else if width < 1000 {
            columnCount = 3
        } else {
            columnCount = 4
        }
        return Array(repeating: GridItem(.flexible(), spacing: spacing), count: columnCount)
    }

    var body: some View {
        NavigationStack {
            ZStack {
                Color.black.ignoresSafeArea()

                if appState.library.games.isEmpty {
                    emptyLibraryView
                } else if appState.config.library.viewMode == .list {
                    gameListView
                } else {
                    gameGridView
                }
            }
            .navigationTitle("Library")
            .navigationBarTitleDisplayMode(.large)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button(action: { showingSettings = true }) {
                        Image(systemName: "gear")
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    Button(action: { showingFilePicker = true }) {
                        Image(systemName: "plus")
                    }
                }
            }
            .fileImporter(
                isPresented: $showingFilePicker,
                allowedContentTypes: LibraryView.romContentTypes,
                allowsMultipleSelection: true
            ) { result in
                switch result {
                case .success(let urls):
                    handleROMImport(urls: urls)
                case .failure(let error):
                    Log.romImport.error("File picker error: \(error.localizedDescription)")
                }
            }
            .sheet(isPresented: $showingSettings) {
                SettingsView()
            }
        }
        .preferredColorScheme(.dark)
    }

    // MARK: - Subviews

    private var emptyLibraryView: some View {
        VStack(spacing: 20) {
            Image(systemName: "gamecontroller")
                .font(.system(size: 60))
                .foregroundColor(.gray)

            Text("No Games")
                .font(.title2)
                .foregroundColor(.gray)

            Text("Tap + to import ROM files")
                .font(.subheadline)
                .foregroundColor(.gray.opacity(0.7))

            Button(action: { showingFilePicker = true }) {
                Label("Import ROMs", systemImage: "plus.circle.fill")
                    .padding()
                    .background(Color.blue)
                    .foregroundColor(.white)
                    .cornerRadius(10)
            }
            .padding(.top, 20)
        }
    }

    private var gameGridView: some View {
        GeometryReader { geometry in
            ScrollView {
                LazyVGrid(columns: gridColumns(for: geometry.size.width), spacing: 16) {
                    ForEach(sortedGames) { game in
                        GameGridItem(game: game)
                            .onTapGesture {
                                appState.navigateToDetail(gameCRC: game.crc32)
                            }
                    }
                }
                .padding()
            }
        }
    }

    private var gameListView: some View {
        List(sortedGames) { game in
            GameListItem(game: game)
                .contentShape(Rectangle())
                .onTapGesture {
                    appState.navigateToDetail(gameCRC: game.crc32)
                }
                .listRowBackground(Color.clear)
        }
        .listStyle(.plain)
    }

    private var sortedGames: [GameEntry] {
        appState.library.sortedGames(by: appState.config.library.sortBy)
    }

    // MARK: - ROM Import

    private func handleROMImport(urls: [URL]) {
        for url in urls {
            importROM(from: url)
        }
    }

    private func importROM(from url: URL) {
        // When using asCopy: true, the file is already copied to the app's Inbox
        // so we may not need security scoped access. Try it, but don't fail if it returns false.
        let hasAccess = url.startAccessingSecurityScopedResource()
        defer {
            if hasAccess {
                url.stopAccessingSecurityScopedResource()
            }
        }

        do {
            // Ensure directories exist
            try StoragePaths.ensureDirectoriesExist()

            // Extract ROM from archive and store as {CRC32}.sms
            // Go handles archive extraction (zip, 7z, gzip, rar) and skips write if file exists
            guard let result = EmulatorBridge.extractAndStoreROM(
                srcPath: url.path,
                destDir: StoragePaths.romsDirectory.path
            ) else {
                Log.romImport.error("Failed to extract ROM: \(url.lastPathComponent)")
                return
            }

            let crcString = result.crc32
            let originalFilename = result.filename

            // Check if already in library
            if appState.library.games[crcString] != nil {
                // Already in library, file already exists with same content
                return
            }

            // Look up metadata from RDB (if available)
            var name = originalFilename
            var displayName = cleanDisplayName(name)
            var region = extractRegion(from: name)

            if let crc32 = UInt32(crcString, radix: 16),
               let rdbGame = appState.lookupGame(crc32: crc32) {
                name = rdbGame.name
                displayName = rdbGame.displayName
                region = rdbGame.region
            }

            // Create library entry - file is stored as {CRC32}.sms
            let entry = GameEntry(
                crc32: crcString,
                file: crcString + ".sms",
                name: name,
                displayName: displayName,
                region: region
            )

            // Add to library
            appState.library.addGame(entry)

            // Download artwork in background
            Task {
                if await appState.artworkDownloader.download(for: crcString, gameName: name) {
                    appState.artworkVersion += 1
                }
            }

        } catch {
            Log.romImport.error("Failed to import ROM '\(url.lastPathComponent)': \(error.localizedDescription)")
        }
    }

    /// Remove parenthetical info from name (region, version, etc.)
    private func cleanDisplayName(_ name: String) -> String {
        if let range = name.range(of: " (") {
            return String(name[..<range.lowerBound]).trimmingCharacters(in: .whitespaces)
        }
        return name
    }

    /// Extract region from No-Intro style name
    private func extractRegion(from name: String) -> String {
        let lower = name.lowercased()
        if lower.contains("(usa") || lower.contains("(us)") || lower.contains(", usa)") {
            return "us"
        }
        if lower.contains("(europe") || lower.contains("(eu)") || lower.contains(", europe)") {
            return "eu"
        }
        if lower.contains("(japan") || lower.contains("(jp)") || lower.contains(", japan)") {
            return "jp"
        }
        return ""
    }
}

/// Grid item for a single game
struct GameGridItem: View {
    @EnvironmentObject var appState: AppState
    let game: GameEntry
    @State private var artworkImage: UIImage?

    var body: some View {
        VStack(spacing: 8) {
            // Artwork
            ZStack {
                RoundedRectangle(cornerRadius: 8)
                    .fill(Color.gray.opacity(0.3))

                if let image = artworkImage {
                    Image(uiImage: image)
                        .resizable()
                        .aspectRatio(contentMode: .fit)
                        .cornerRadius(8)
                } else {
                    Image(systemName: "gamecontroller")
                        .font(.system(size: 40))
                        .foregroundColor(.gray)
                }

                if game.missing {
                    Color.red.opacity(0.5)
                        .cornerRadius(8)

                    Image(systemName: "exclamationmark.triangle")
                        .font(.system(size: 30))
                        .foregroundColor(.white)
                }
            }
            .aspectRatio(1, contentMode: .fit)

            // Title
            Text(game.displayName)
                .font(.caption)
                .foregroundColor(.white)
                .lineLimit(2)
                .multilineTextAlignment(.center)
        }
        .onAppear {
            artworkImage = ArtworkLoader.loadImage(for: game.crc32)
        }
        .onChange(of: appState.artworkVersion) { _, _ in
            artworkImage = ArtworkLoader.loadImage(for: game.crc32)
        }
    }
}

/// List item for a single game
struct GameListItem: View {
    @EnvironmentObject var appState: AppState
    let game: GameEntry
    @State private var artworkImage: UIImage?

    var body: some View {
        HStack(spacing: 12) {
            // Thumbnail
            ZStack {
                RoundedRectangle(cornerRadius: 6)
                    .fill(Color.gray.opacity(0.3))

                if let image = artworkImage {
                    Image(uiImage: image)
                        .resizable()
                        .aspectRatio(contentMode: .fit)
                        .cornerRadius(6)
                } else {
                    Image(systemName: "gamecontroller")
                        .font(.system(size: 20))
                        .foregroundColor(.gray)
                }

                if game.missing {
                    Color.red.opacity(0.5)
                        .cornerRadius(6)
                }
            }
            .frame(width: 50, height: 50)

            // Title and region
            VStack(alignment: .leading, spacing: 4) {
                Text(game.displayName)
                    .font(.body)
                    .foregroundColor(.white)
                    .lineLimit(1)

                Text(regionDisplayName(game.region))
                    .font(.caption)
                    .foregroundColor(.gray)
            }

            Spacer()

            // Chevron
            Image(systemName: "chevron.right")
                .foregroundColor(.gray)
        }
        .padding(.vertical, 4)
        .onAppear {
            artworkImage = ArtworkLoader.loadImage(for: game.crc32)
        }
        .onChange(of: appState.artworkVersion) { _, _ in
            artworkImage = ArtworkLoader.loadImage(for: game.crc32)
        }
    }

    private func regionDisplayName(_ region: String) -> String {
        switch region.lowercased() {
        case "us", "usa": return "USA"
        case "eu", "europe": return "Europe"
        case "jp", "japan": return "Japan"
        default: return region.isEmpty ? "Unknown" : region.uppercased()
        }
    }
}

#Preview {
    LibraryView()
        .environmentObject(AppState())
}
