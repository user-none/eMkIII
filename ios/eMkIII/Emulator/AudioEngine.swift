import Foundation
import AVFoundation
import Accelerate

/// Audio playback engine using AVAudioEngine with scheduled buffers
class AudioEngine {
    private var audioEngine: AVAudioEngine?
    private var playerNode: AVAudioPlayerNode?
    private var audioFormat: AVAudioFormat?

    // Audio format - must match emulator output
    static let sampleRate: Double = 48000
    static let channelCount: AVAudioChannelCount = 2

    // Buffer level thresholds (in stereo sample pairs = frames)
    static let targetBufferLevel = 3200   // ~67ms target at 48kHz (4 frames at 60fps)
    static let minBufferLevel = 2400      // ~50ms - speed up below this
    static let maxBufferLevel = 4800      // ~100ms - slow down above this

    // Accumulate samples across frames for smoother playback
    private var pendingSamples: [Int16] = []
    private let samplesLock = NSLock()
    private let samplesPerBuffer = 1600  // ~16ms worth at 48kHz stereo
    private let maxPendingSamples = 9600  // ~100ms max buffer
    private let preBufferThreshold = 3200  // Wait for ~33ms before starting to schedule
    private var isPreBuffering = true

    // Buffer level tracking for audio-driven timing
    private var samplesScheduled: Int64 = 0
    private let bufferTrackingLock = NSLock()
    private var playbackStartTime: AVAudioTime?

    var isRunning: Bool {
        audioEngine?.isRunning ?? false
    }

    init() {
        pendingSamples.reserveCapacity(samplesPerBuffer * 2)
    }

    /// Start the audio engine
    func start(muted: Bool = false) throws {
        // Configure audio session
        let session = AVAudioSession.sharedInstance()
        try session.setCategory(.playback, mode: .default, options: [.mixWithOthers])
        try session.setPreferredSampleRate(Self.sampleRate)
        try session.setActive(true)

        let engine = AVAudioEngine()
        let player = AVAudioPlayerNode()

        // Create format: 48kHz stereo float32
        guard let format = AVAudioFormat(
            commonFormat: .pcmFormatFloat32,
            sampleRate: Self.sampleRate,
            channels: Self.channelCount,
            interleaved: false
        ) else {
            throw AudioError.formatCreationFailed
        }

        engine.attach(player)
        engine.connect(player, to: engine.mainMixerNode, format: format)

        // Set volume before starting to prevent pop when muted
        engine.mainMixerNode.outputVolume = muted ? 0.0 : 1.0

        try engine.start()
        player.play()

        self.audioEngine = engine
        self.playerNode = player
        self.audioFormat = format
    }

    /// Stop the audio engine
    func stop() {
        playerNode?.stop()
        audioEngine?.stop()
        audioEngine = nil
        playerNode = nil
    }

    /// Queue audio samples for playback
    /// Data format: little-endian int16, interleaved stereo (L, R, L, R, ...)
    func queueSamples(_ data: Data) {
        guard data.count > 0 else { return }

        samplesLock.lock()
        defer { samplesLock.unlock() }

        let incomingSampleCount = data.count / 2

        // Check if buffer would overflow
        if pendingSamples.count + incomingSampleCount > maxPendingSamples {
            let toDrop = pendingSamples.count + incomingSampleCount - maxPendingSamples
            // Drop oldest samples
            if toDrop <= pendingSamples.count {
                pendingSamples.removeFirst(toDrop)
            } else {
                pendingSamples.removeAll()
            }
        }

        // Accumulate samples using bulk append (much faster than one-by-one)
        data.withUnsafeBytes { ptr in
            guard let basePtr = ptr.baseAddress else { return }
            let int16Ptr = basePtr.assumingMemoryBound(to: Int16.self)
            let count = data.count / 2
            let buffer = UnsafeBufferPointer(start: int16Ptr, count: count)
            pendingSamples.append(contentsOf: buffer)
        }

        // Pre-buffer before starting to schedule
        if isPreBuffering {
            if pendingSamples.count >= preBufferThreshold {
                isPreBuffering = false
            } else {
                return
            }
        }

        // Schedule buffer when we have enough samples
        while pendingSamples.count >= samplesPerBuffer {
            scheduleBuffer(samples: Array(pendingSamples.prefix(samplesPerBuffer)))
            pendingSamples.removeFirst(samplesPerBuffer)
        }
    }

    private func scheduleBuffer(samples: [Int16]) {
        guard let player = playerNode,
              let format = audioFormat else { return }

        let frameCount = samples.count / 2  // stereo = 2 samples per frame

        guard let buffer = AVAudioPCMBuffer(pcmFormat: format, frameCapacity: AVAudioFrameCount(frameCount)) else {
            return
        }
        buffer.frameLength = AVAudioFrameCount(frameCount)

        guard let leftChannel = buffer.floatChannelData?[0],
              let rightChannel = buffer.floatChannelData?[1] else {
            return
        }

        // Convert int16 to float with de-interleaving
        let scale: Float = 1.0 / 32768.0
        samples.withUnsafeBufferPointer { samplesPtr in
            for i in 0..<frameCount {
                leftChannel[i] = Float(samplesPtr[i * 2]) * scale
                rightChannel[i] = Float(samplesPtr[i * 2 + 1]) * scale
            }
        }

        // Track scheduled samples and capture start time
        bufferTrackingLock.lock()
        if playbackStartTime == nil, let nodeTime = player.lastRenderTime, nodeTime.isSampleTimeValid {
            playbackStartTime = nodeTime
        }
        samplesScheduled += Int64(frameCount)
        bufferTrackingLock.unlock()

        player.scheduleBuffer(buffer, completionHandler: nil)
    }

    /// Clear the audio buffer
    func clearBuffer() {
        samplesLock.lock()
        pendingSamples.removeAll(keepingCapacity: true)
        isPreBuffering = true
        samplesLock.unlock()

        bufferTrackingLock.lock()
        samplesScheduled = 0
        playbackStartTime = nil
        bufferTrackingLock.unlock()

        playerNode?.stop()
        playerNode?.play()
    }

    /// Get the current buffer level (pending + scheduled - played)
    /// Returns the number of stereo sample frames waiting to be played
    func getBufferLevel() -> Int {
        samplesLock.lock()
        let pending = pendingSamples.count / 2  // Convert to frames
        samplesLock.unlock()

        bufferTrackingLock.lock()
        let scheduled = samplesScheduled
        let startTime = playbackStartTime
        bufferTrackingLock.unlock()

        // Estimate played samples from player's current position
        var played: Int64 = 0
        if let player = playerNode,
           let start = startTime,
           let currentTime = player.lastRenderTime,
           currentTime.isSampleTimeValid && start.isSampleTimeValid {
            played = max(0, currentTime.sampleTime - start.sampleTime)
        }

        let inFlight = Int(max(0, scheduled - played))
        return pending + inFlight
    }

    /// Set the audio volume (0.0 = muted, 1.0 = full volume)
    func setVolume(_ volume: Float) {
        audioEngine?.mainMixerNode.outputVolume = max(0.0, min(1.0, volume))
    }
}

enum AudioError: Error {
    case formatCreationFailed
}
