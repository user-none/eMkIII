import SwiftUI

@main
struct eMkIIIApp: App {
    @StateObject private var appState = AppState()

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(appState)
                .preferredColorScheme(.dark)
        }
    }
}

struct ContentView: View {
    @EnvironmentObject var appState: AppState

    var body: some View {
        switch appState.currentScreen {
        case .library:
            LibraryView()
        case .settings:
            SettingsView()
        case .detail(let gameCRC):
            GameDetailView(gameCRC: gameCRC)
        case .gameplay(let gameCRC, let resume):
            GameplayView(gameCRC: gameCRC, resume: resume)
        }
    }
}
