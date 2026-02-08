import SwiftUI
import MetalKit
import GameController

/// Main gameplay view with emulator and touch controls
struct GameplayView: View {
    @EnvironmentObject var appState: AppState
    let gameCRC: String
    let resume: Bool

    @State private var controlState = ControlState()
    @State private var showPauseMenu = false
    @State private var emulatorManager: EmulatorManager?
    @State private var gamepadObserver: NSObjectProtocol?

    var body: some View {
        GeometryReader { geometry in
            ZStack {
                Color.black.ignoresSafeArea()

                if let manager = emulatorManager {
                    // Metal view for emulator display - explicitly sized to fill
                    MetalEmulatorView(manager: manager, cropBorder: appState.config.video.cropBorder, size: geometry.size)
                        .frame(width: geometry.size.width, height: geometry.size.height)
                        .ignoresSafeArea()
                }

                // Touch controls overlay
                TouchControlsView(
                    controlState: $controlState,
                    onMenuTap: { showPauseMenu = true }
                )
                .ignoresSafeArea()

                // Pause menu overlay
                if showPauseMenu {
                    PauseMenuView(
                        onResume: { showPauseMenu = false },
                        onLibrary: { exitToLibrary() }
                    )
                }
            }
        }
        .ignoresSafeArea()
        .statusBar(hidden: true)
        .persistentSystemOverlays(.hidden)
        .onAppear {
            startEmulator()
            setupGamepadObserver()
        }
        .onDisappear {
            stopEmulator()
            removeGamepadObserver()
        }
        .onChange(of: controlState) { _, newState in
            updateInput(state: newState)
        }
        .onChange(of: showPauseMenu) { _, isPaused in
            if isPaused {
                emulatorManager?.pause()
            } else {
                emulatorManager?.resume()
            }
        }
    }

    // MARK: - Emulator Lifecycle

    private func startEmulator() {
        guard let game = appState.library.games[gameCRC] else {
            appState.navigateToLibrary()
            return
        }

        // Create emulator manager and load ROM from path
        let manager = EmulatorManager()
        guard manager.loadROM(path: game.filePath) else {
            appState.navigateToLibrary()
            return
        }

        // Load SRAM if available
        manager.loadSRAM(gameCRC: gameCRC)

        // Load resume state if requested
        if resume {
            manager.loadResumeState(gameCRC: gameCRC)
        }

        // Start emulation
        manager.start(muted: appState.config.audio.mute)

        self.emulatorManager = manager

        // Update last played
        appState.updateGameLastPlayed(crc: gameCRC)
    }

    private func stopEmulator() {
        guard let manager = emulatorManager else { return }

        // Save state
        manager.saveResumeState(gameCRC: gameCRC)
        manager.saveSRAM(gameCRC: gameCRC)

        // Stop emulation
        manager.stop()

        emulatorManager = nil
    }

    private func exitToLibrary() {
        showPauseMenu = false
        appState.navigateToLibrary()
    }

    private func updateInput(state: ControlState) {
        emulatorManager?.setInput(
            up: state.up,
            down: state.down,
            left: state.left,
            right: state.right,
            button1: state.buttonA,
            button2: state.buttonB
        )

        if state.pause {
            emulatorManager?.triggerPause()
        }
    }

    // MARK: - Gamepad Support

    private func setupGamepadObserver() {
        gamepadObserver = NotificationCenter.default.addObserver(
            forName: .GCControllerDidConnect,
            object: nil,
            queue: .main
        ) { [weak emulatorManager] _ in
            guard emulatorManager != nil else { return }
            setupGamepadInput()
        }

        // Setup any already connected controller
        setupGamepadInput()
    }

    private func removeGamepadObserver() {
        if let observer = gamepadObserver {
            NotificationCenter.default.removeObserver(observer)
            gamepadObserver = nil
        }
        // Clear gamepad handlers to prevent any lingering closure references
        clearGamepadHandlers()
    }

    private func clearGamepadHandlers() {
        guard let controller = GCController.controllers().first,
              let gamepad = controller.extendedGamepad else {
            return
        }
        gamepad.dpad.valueChangedHandler = nil
        gamepad.buttonA.valueChangedHandler = nil
        gamepad.buttonB.valueChangedHandler = nil
        gamepad.buttonMenu.valueChangedHandler = nil
        controller.extendedGamepad?.buttonOptions?.valueChangedHandler = nil
    }

    private func setupGamepadInput() {
        guard let controller = GCController.controllers().first,
              let gamepad = controller.extendedGamepad else {
            return
        }

        gamepad.dpad.valueChangedHandler = { [weak emulatorManager] dpad, _, _ in
            emulatorManager?.setInput(
                up: dpad.up.isPressed,
                down: dpad.down.isPressed,
                left: dpad.left.isPressed,
                right: dpad.right.isPressed,
                button1: gamepad.buttonA.isPressed,
                button2: gamepad.buttonB.isPressed
            )
        }

        gamepad.buttonA.valueChangedHandler = { [weak emulatorManager] _, _, pressed in
            if let manager = emulatorManager {
                manager.setInput(
                    up: gamepad.dpad.up.isPressed,
                    down: gamepad.dpad.down.isPressed,
                    left: gamepad.dpad.left.isPressed,
                    right: gamepad.dpad.right.isPressed,
                    button1: pressed,
                    button2: gamepad.buttonB.isPressed
                )
            }
        }

        gamepad.buttonB.valueChangedHandler = { [weak emulatorManager] _, _, pressed in
            if let manager = emulatorManager {
                manager.setInput(
                    up: gamepad.dpad.up.isPressed,
                    down: gamepad.dpad.down.isPressed,
                    left: gamepad.dpad.left.isPressed,
                    right: gamepad.dpad.right.isPressed,
                    button1: gamepad.buttonA.isPressed,
                    button2: pressed
                )
            }
        }

        // Menu button opens pause menu
        gamepad.buttonMenu.valueChangedHandler = { [self] _, _, pressed in
            if pressed {
                self.showPauseMenu = true
            }
        }

        // Start button triggers SMS pause
        controller.extendedGamepad?.buttonOptions?.valueChangedHandler = { [weak emulatorManager] _, _, pressed in
            if pressed {
                emulatorManager?.triggerPause()
            }
        }
    }
}

/// Metal view wrapper for SwiftUI
struct MetalEmulatorView: UIViewRepresentable {
    let manager: EmulatorManager
    let cropBorder: Bool
    let size: CGSize

    func makeUIView(context: Context) -> MTKView {
        let mtkView = MTKView()
        mtkView.preferredFramesPerSecond = manager.fps
        mtkView.enableSetNeedsDisplay = false
        mtkView.isPaused = false
        mtkView.autoResizeDrawable = true

        // Create renderer and attach to view
        manager.setupRenderer(for: mtkView)

        if let renderer = manager.renderer {
            renderer.cropLeftBorder = cropBorder
            renderer.viewSize = size
            renderer.onFrameRequest = { [weak manager] in
                manager?.getFrameBuffer()
            }
        }

        return mtkView
    }

    func updateUIView(_ uiView: MTKView, context: Context) {
        manager.renderer?.cropLeftBorder = cropBorder
        manager.renderer?.viewSize = size
        // Force drawable to resize to match the view's current size
        uiView.drawableSize = CGSize(width: size.width * UIScreen.main.scale,
                                      height: size.height * UIScreen.main.scale)
    }
}

/// Pause menu overlay
struct PauseMenuView: View {
    var onResume: () -> Void
    var onLibrary: () -> Void

    var body: some View {
        ZStack {
            // Dim background
            Color.black.opacity(0.7)
                .ignoresSafeArea()

            VStack(spacing: 16) {
                Text("Paused")
                    .font(.title)
                    .foregroundColor(.white)
                    .padding(.bottom, 20)

                Button(action: onResume) {
                    Text("Resume")
                        .frame(width: 200)
                        .padding()
                        .background(Color.blue)
                        .foregroundColor(.white)
                        .cornerRadius(10)
                }

                Button(action: onLibrary) {
                    Text("Exit to Library")
                        .frame(width: 200)
                        .padding()
                        .background(Color.gray.opacity(0.5))
                        .foregroundColor(.white)
                        .cornerRadius(10)
                }
            }
            .padding(40)
            .background(Color.gray.opacity(0.3))
            .cornerRadius(20)
        }
    }
}

/// Manages emulator state and frame loop
class EmulatorManager: ObservableObject {
    private let emulator = EmulatorBridge()
    private var audioEngine: AudioEngine?
    private(set) var renderer: MetalRenderer?

    // Emulation runs on dedicated high-priority thread
    private var emulationThread: Thread?
    private var isRunning = false
    private var isPaused = false
    private let emulationLock = NSLock()

    // Cached frame buffer for fast access from Metal renderer
    private var cachedFrameBuffer: Data?
    private let frameBufferLock = NSLock()

    private let saveStateManager = SaveStateManager()

    var fps: Int {
        emulator.fps
    }

    func loadROM(path: String) -> Bool {
        return emulator.loadROM(path: path)
    }

    func start(muted: Bool) {
        // Setup audio
        if !muted {
            audioEngine = AudioEngine()
            do {
                try audioEngine?.start()
            } catch {
                Log.emulator.error("Failed to start audio engine: \(error.localizedDescription)")
            }
        }

        // Start emulation on dedicated thread
        isRunning = true
        isPaused = false
        emulationThread = Thread { [weak self] in
            self?.emulationLoop()
        }
        emulationThread?.qualityOfService = .userInteractive
        emulationThread?.name = "EmulatorThread"
        emulationThread?.start()
    }

    func stop() {
        isRunning = false

        // Wait for emulation thread to finish before releasing
        while emulationThread?.isExecuting == true {
            Thread.sleep(forTimeInterval: 0.001)
        }
        emulationThread = nil

        audioEngine?.stop()
        audioEngine = nil
        emulator.unload()
    }

    func pause() {
        emulationLock.lock()
        defer { emulationLock.unlock() }
        isPaused = true
        audioEngine?.clearBuffer()
    }

    func resume() {
        emulationLock.lock()
        defer { emulationLock.unlock() }
        isPaused = false
    }

    private func emulationLoop() {
        let targetFPS = fps
        let frameTime = 1.0 / Double(targetFPS)
        var lastFrameTime = CACurrentMediaTime()

        while isRunning {
            // Check if paused
            emulationLock.lock()
            let paused = isPaused
            emulationLock.unlock()

            if paused {
                Thread.sleep(forTimeInterval: 0.01)
                lastFrameTime = CACurrentMediaTime()
                continue
            }

            // Run emulation frame
            emulator.runFrame()

            // Cache frame buffer for Metal renderer (avoid Go bridge during render)
            frameBufferLock.lock()
            cachedFrameBuffer = emulator.getFrameBuffer()
            frameBufferLock.unlock()

            // Queue audio
            if let samples = emulator.getAudioSamples() {
                audioEngine?.queueSamples(samples)
            }

            // Frame timing - sleep until next frame
            let now = CACurrentMediaTime()
            let elapsed = now - lastFrameTime
            let sleepTime = frameTime - elapsed

            if sleepTime > 0 {
                Thread.sleep(forTimeInterval: sleepTime)
            }

            lastFrameTime = CACurrentMediaTime()
        }
    }

    func getFrameBuffer() -> Data? {
        frameBufferLock.lock()
        let buffer = cachedFrameBuffer
        frameBufferLock.unlock()
        return buffer
    }

    func setInput(up: Bool, down: Bool, left: Bool, right: Bool, button1: Bool, button2: Bool) {
        emulator.setInput(up: up, down: down, left: left, right: right, button1: button1, button2: button2)
    }

    func triggerPause() {
        emulator.triggerPause()
    }

    // MARK: - Save State

    func saveResumeState(gameCRC: String) {
        saveStateManager.setGame(crc: gameCRC)
        guard let data = emulator.serialize() else {
            return
        }
        do {
            try saveStateManager.saveResumeState(data: data)
        } catch {
            Log.storage.error("Failed to save resume state: \(error.localizedDescription)")
        }
    }

    func loadResumeState(gameCRC: String) {
        saveStateManager.setGame(crc: gameCRC)
        do {
            let data = try saveStateManager.loadResumeState()
            _ = emulator.deserialize(data: data)
        } catch {
            // Resume state not found is normal for first launch
            Log.storage.debug("Resume state not loaded: \(error.localizedDescription)")
        }
    }

    func saveSRAM(gameCRC: String) {
        saveStateManager.setGame(crc: gameCRC)
        guard let data = emulator.getCartRAM() else { return }
        do {
            try saveStateManager.saveSRAM(data: data)
        } catch {
            Log.storage.error("Failed to save SRAM: \(error.localizedDescription)")
        }
    }

    func loadSRAM(gameCRC: String) {
        saveStateManager.setGame(crc: gameCRC)
        do {
            let data = try saveStateManager.loadSRAM()
            emulator.setCartRAM(data: data)
        } catch {
            // SRAM not found is normal for games without battery save
            Log.storage.debug("SRAM not loaded: \(error.localizedDescription)")
        }
    }

    func setupRenderer(for mtkView: MTKView) {
        renderer = MetalRenderer(mtkView: mtkView)
        mtkView.delegate = renderer
    }
}

#Preview {
    GameplayView(gameCRC: "12345678", resume: false)
        .environmentObject(AppState())
}
