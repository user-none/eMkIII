import Foundation
import Metal
import MetalKit
import QuartzCore

/// Metal-based renderer for emulator framebuffer
class MetalRenderer: NSObject, MTKViewDelegate {
    // Metal objects
    private let device: MTLDevice
    private let commandQueue: MTLCommandQueue
    private let pipelineState: MTLRenderPipelineState
    private var texture: MTLTexture?
    private let sampler: MTLSamplerState

    // Vertex buffer for fullscreen quad
    private let vertexBuffer: MTLBuffer

    // Current frame dimensions
    private var currentWidth: Int = 256
    private var currentHeight: Int = 192

    // Callback for frame requests
    var onFrameRequest: (() -> Data?)?

    // Border cropping
    var cropLeftBorder: Bool = false

    // View size (set from SwiftUI to ensure correct orientation)
    var viewSize: CGSize = .zero

    init?(mtkView: MTKView) {
        guard let device = MTLCreateSystemDefaultDevice(),
              let commandQueue = device.makeCommandQueue() else {
            return nil
        }

        self.device = device
        self.commandQueue = commandQueue

        // Configure view
        mtkView.device = device
        mtkView.colorPixelFormat = .bgra8Unorm
        mtkView.clearColor = MTLClearColor(red: 0, green: 0, blue: 0, alpha: 1)

        // Create shader library and pipeline
        guard let library = device.makeDefaultLibrary(),
              let vertexFunction = library.makeFunction(name: "vertexShader"),
              let fragmentFunction = library.makeFunction(name: "fragmentShader") else {

            // If shaders aren't compiled, use simple passthrough
            // This allows the app to work even without compiled shaders
            guard let pipelineState = Self.createSimplePipeline(device: device) else {
                return nil
            }
            self.pipelineState = pipelineState
            self.sampler = Self.createSampler(device: device)!
            self.vertexBuffer = Self.createVertexBuffer(device: device)!
            super.init()
            return
        }

        let pipelineDescriptor = MTLRenderPipelineDescriptor()
        pipelineDescriptor.vertexFunction = vertexFunction
        pipelineDescriptor.fragmentFunction = fragmentFunction
        pipelineDescriptor.colorAttachments[0].pixelFormat = mtkView.colorPixelFormat

        guard let pipelineState = try? device.makeRenderPipelineState(descriptor: pipelineDescriptor) else {
            return nil
        }
        self.pipelineState = pipelineState

        // Create sampler for nearest-neighbor filtering (crisp pixels)
        guard let sampler = Self.createSampler(device: device) else {
            return nil
        }
        self.sampler = sampler

        // Create vertex buffer
        guard let vertexBuffer = Self.createVertexBuffer(device: device) else {
            return nil
        }
        self.vertexBuffer = vertexBuffer

        super.init()
    }

    private static func createSampler(device: MTLDevice) -> MTLSamplerState? {
        let samplerDescriptor = MTLSamplerDescriptor()
        samplerDescriptor.minFilter = .nearest
        samplerDescriptor.magFilter = .nearest
        samplerDescriptor.sAddressMode = .clampToEdge
        samplerDescriptor.tAddressMode = .clampToEdge
        return device.makeSamplerState(descriptor: samplerDescriptor)
    }

    private static func createVertexBuffer(device: MTLDevice) -> MTLBuffer? {
        // Fullscreen quad vertices (position + texcoord)
        let vertices: [Float] = [
            // Position (x, y), Texcoord (u, v)
            -1.0, -1.0, 0.0, 1.0,  // Bottom-left
             1.0, -1.0, 1.0, 1.0,  // Bottom-right
            -1.0,  1.0, 0.0, 0.0,  // Top-left
             1.0,  1.0, 1.0, 0.0,  // Top-right
        ]
        return device.makeBuffer(bytes: vertices, length: vertices.count * MemoryLayout<Float>.size, options: .storageModeShared)
    }

    private static func createSimplePipeline(device: MTLDevice) -> MTLRenderPipelineState? {
        // Create a simple pipeline without custom shaders
        let shaderSource = """
        #include <metal_stdlib>
        using namespace metal;

        struct VertexOut {
            float4 position [[position]];
            float2 texCoord;
        };

        vertex VertexOut vertexShader(uint vertexID [[vertex_id]],
                                      constant float4 *vertices [[buffer(0)]]) {
            float4 v = vertices[vertexID];
            VertexOut out;
            out.position = float4(v.xy, 0.0, 1.0);
            out.texCoord = v.zw;
            return out;
        }

        fragment float4 fragmentShader(VertexOut in [[stage_in]],
                                       texture2d<float> texture [[texture(0)]],
                                       sampler textureSampler [[sampler(0)]]) {
            return texture.sample(textureSampler, in.texCoord);
        }
        """

        do {
            let library = try device.makeLibrary(source: shaderSource, options: nil)
            let vertexFunction = library.makeFunction(name: "vertexShader")
            let fragmentFunction = library.makeFunction(name: "fragmentShader")

            let pipelineDescriptor = MTLRenderPipelineDescriptor()
            pipelineDescriptor.vertexFunction = vertexFunction
            pipelineDescriptor.fragmentFunction = fragmentFunction
            pipelineDescriptor.colorAttachments[0].pixelFormat = .bgra8Unorm

            return try device.makeRenderPipelineState(descriptor: pipelineDescriptor)
        } catch {
            return nil
        }
    }

    /// Update the texture with new framebuffer data
    func updateTexture(with data: Data, width: Int, height: Int) {
        // Recreate texture if dimensions changed
        if texture == nil || currentWidth != width || currentHeight != height {
            let textureDescriptor = MTLTextureDescriptor.texture2DDescriptor(
                pixelFormat: .rgba8Unorm,
                width: width,
                height: height,
                mipmapped: false
            )
            textureDescriptor.usage = [.shaderRead]
            texture = device.makeTexture(descriptor: textureDescriptor)
            currentWidth = width
            currentHeight = height
        }

        // Copy pixel data to texture
        guard let texture = texture else { return }

        data.withUnsafeBytes { ptr in
            guard let baseAddress = ptr.baseAddress else { return }
            let region = MTLRegion(origin: MTLOrigin(x: 0, y: 0, z: 0),
                                   size: MTLSize(width: width, height: height, depth: 1))
            texture.replace(region: region,
                           mipmapLevel: 0,
                           withBytes: baseAddress,
                           bytesPerRow: width * 4)
        }
    }

    // MARK: - MTKViewDelegate

    func mtkView(_ view: MTKView, drawableSizeWillChange size: CGSize) {
        // Handle resize if needed
    }

    func draw(in view: MTKView) {
        // Force drawable to match view bounds on every frame
        let scale = UIScreen.main.scale
        let expectedDrawableSize = CGSize(width: view.bounds.width * scale, height: view.bounds.height * scale)
        if view.drawableSize != expectedDrawableSize {
            view.drawableSize = expectedDrawableSize
        }

        // Request frame data from emulator
        if let frameData = onFrameRequest?() {
            let height = frameData.count / (256 * 4)
            updateTexture(with: frameData, width: 256, height: height)
        }

        guard let texture = texture,
              let drawable = view.currentDrawable,
              let renderPassDescriptor = view.currentRenderPassDescriptor,
              let commandBuffer = commandQueue.makeCommandBuffer(),
              let renderEncoder = commandBuffer.makeRenderCommandEncoder(descriptor: renderPassDescriptor) else {
            return
        }

        // Calculate aspect-correct scaling using view bounds (always correct)
        let effectiveSize = view.bounds.size
        let viewAspect = Float(effectiveSize.width / effectiveSize.height)

        let srcWidth = cropLeftBorder ? 248 : 256
        let textureAspect = Float(srcWidth) / Float(currentHeight)

        var scaleX: Float = 1.0
        var scaleY: Float = 1.0

        if viewAspect > textureAspect {
            // View is wider than texture - letterbox on sides
            scaleX = textureAspect / viewAspect
        } else {
            // View is taller than texture - letterbox top/bottom
            scaleY = viewAspect / textureAspect
        }

        // Update vertex buffer with proper scaling
        let offsetX: Float = cropLeftBorder ? (8.0 / 256.0) : 0.0
        let texWidth: Float = cropLeftBorder ? (248.0 / 256.0) : 1.0

        // Only apply vertical offset in portrait mode (when view is taller than texture)
        // In landscape, game fills full height and is centered horizontally
        let isPortrait = viewAspect < textureAspect
        let offsetY: Float = isPortrait ? (1.0 - scaleY) * 0.4 : 0.0

        let vertices: [Float] = [
            -scaleX, -scaleY + offsetY, offsetX, 1.0,
             scaleX, -scaleY + offsetY, offsetX + texWidth, 1.0,
            -scaleX,  scaleY + offsetY, offsetX, 0.0,
             scaleX,  scaleY + offsetY, offsetX + texWidth, 0.0,
        ]

        renderEncoder.setRenderPipelineState(pipelineState)
        renderEncoder.setVertexBytes(vertices, length: vertices.count * MemoryLayout<Float>.size, index: 0)
        renderEncoder.setFragmentTexture(texture, index: 0)
        renderEncoder.setFragmentSamplerState(sampler, index: 0)
        renderEncoder.drawPrimitives(type: .triangleStrip, vertexStart: 0, vertexCount: 4)
        renderEncoder.endEncoding()

        commandBuffer.present(drawable)
        commandBuffer.commit()
    }
}
