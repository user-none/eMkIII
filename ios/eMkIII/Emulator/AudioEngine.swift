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

    // Volume control
    private let volume: Float = 0.2

    // Accumulate samples across frames for smoother playback
    private var pendingSamples: [Int16] = []
    private let samplesLock = NSLock()
    private let samplesPerBuffer = 1600  // ~16ms worth at 48kHz stereo
    private let maxPendingSamples = 9600  // ~100ms max buffer
    private let preBufferThreshold = 3200  // Wait for ~33ms before starting to schedule
    private var isPreBuffering = true

    var isRunning: Bool {
        audioEngine?.isRunning ?? false
    }

    init() {
        pendingSamples.reserveCapacity(samplesPerBuffer * 2)
    }

    /// Start the audio engine
    func start() throws {
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

        // Use vDSP for fast int16 to float conversion with de-interleaving
        let scale = volume / 32768.0
        samples.withUnsafeBufferPointer { samplesPtr in
            for i in 0..<frameCount {
                leftChannel[i] = Float(samplesPtr[i * 2]) * scale
                rightChannel[i] = Float(samplesPtr[i * 2 + 1]) * scale
            }
        }

        player.scheduleBuffer(buffer, completionHandler: nil)
    }

    /// Clear the audio buffer
    func clearBuffer() {
        samplesLock.lock()
        pendingSamples.removeAll(keepingCapacity: true)
        isPreBuffering = true
        samplesLock.unlock()
        playerNode?.stop()
        playerNode?.play()
    }
}

enum AudioError: Error {
    case formatCreationFailed
}
