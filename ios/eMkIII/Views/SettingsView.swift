import SwiftUI

/// Settings view for app configuration
struct SettingsView: View {
    @EnvironmentObject var appState: AppState
    @Environment(\.dismiss) var dismiss

    var body: some View {
        NavigationStack {
            Form {
                // Video settings
                Section("Video") {
                    Toggle("Crop Left Border", isOn: Binding(
                        get: { appState.config.video.cropBorder },
                        set: { newValue in
                            appState.config.video.cropBorder = newValue
                            appState.saveConfig()
                        }
                    ))
                }

                // Audio settings
                Section("Audio") {
                    Toggle("Mute", isOn: Binding(
                        get: { appState.config.audio.mute },
                        set: { newValue in
                            appState.config.audio.mute = newValue
                            appState.saveConfig()
                        }
                    ))
                }

                // Library settings
                Section("Library") {
                    Picker("View Mode", selection: Binding(
                        get: { appState.config.library.viewMode },
                        set: { newValue in
                            appState.config.library.viewMode = newValue
                            appState.saveConfig()
                        }
                    )) {
                        ForEach(Config.LibraryConfig.ViewMode.allCases, id: \.self) { mode in
                            Text(mode.displayName).tag(mode)
                        }
                    }

                    Picker("Sort By", selection: Binding(
                        get: { appState.config.library.sortBy },
                        set: { newValue in
                            appState.config.library.sortBy = newValue
                            appState.saveConfig()
                        }
                    )) {
                        ForEach(Library.SortMethod.allCases, id: \.self) { method in
                            Text(method.displayName).tag(method)
                        }
                    }
                }

                // Database section
                Section("Database") {
                    HStack {
                        Text("Game Database")
                        Spacer()
                        if appState.isRDBDownloading {
                            ProgressView()
                        } else if appState.isRDBLoaded {
                            Text("Loaded")
                                .foregroundColor(.green)
                        } else {
                            Text("Not Downloaded")
                                .foregroundColor(.gray)
                        }
                    }

                    Button(action: downloadRDB) {
                        HStack {
                            Text(appState.isRDBLoaded ? "Update Database" : "Download Database")
                            Spacer()
                            if appState.isRDBDownloading {
                                ProgressView()
                            }
                        }
                    }
                    .disabled(appState.isRDBDownloading)

                    Button(action: downloadArtwork) {
                        Text("Download Missing Artwork")
                    }
                }

                // About section
                Section("About") {
                    HStack {
                        Text("Version")
                        Spacer()
                        Text(Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0")
                            .foregroundColor(.gray)
                    }

                    HStack {
                        Text("Emulator")
                        Spacer()
                        Text("eMkIII")
                            .foregroundColor(.gray)
                    }
                }
            }
            .navigationTitle("Settings")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("Done") {
                        dismiss()
                    }
                }
            }
        }
        .preferredColorScheme(.dark)
    }

    private func downloadRDB() {
        Task {
            await appState.downloadRDB()
        }
    }

    private func downloadArtwork() {
        Task {
            await appState.downloadMissingArtwork()
        }
    }
}

#Preview {
    SettingsView()
        .environmentObject(AppState())
}
