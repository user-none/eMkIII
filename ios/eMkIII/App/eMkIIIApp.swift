import SwiftUI
import EblituiIOS

@main
struct eMkIIIApp: App {
    @StateObject private var appState: AppState

    init() {
        EmulatorBridge.register(EmkiiiBridgeProvider.self)
        _appState = StateObject(wrappedValue: AppState())
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
                .environmentObject(appState)
                .preferredColorScheme(.dark)
        }
    }
}
