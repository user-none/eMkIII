import SwiftUI
import UIKit

/// Touch control state
struct ControlState: Equatable {
    var up: Bool = false
    var down: Bool = false
    var left: Bool = false
    var right: Bool = false
    var buttonA: Bool = false
    var buttonB: Bool = false
    var pause: Bool = false
    var menu: Bool = false
}

/// Virtual touch controls overlay for gameplay
struct TouchControlsView: View {
    @Binding var controlState: ControlState
    var onMenuTap: () -> Void

    // Haptic feedback generator
    private let impactGenerator = UIImpactFeedbackGenerator(style: .light)

    var body: some View {
        GeometryReader { geometry in
            let isLandscape = geometry.size.width > geometry.size.height

            if isLandscape {
                landscapeLayout(size: geometry.size)
            } else {
                portraitLayout(size: geometry.size)
            }
        }
    }

    // MARK: - Layouts

    @ViewBuilder
    private func landscapeLayout(size: CGSize) -> some View {
        ZStack {
            // Left side - D-Pad centered vertically with Menu/Pause above
            HStack {
                VStack(spacing: 10) {
                    // Menu and Pause at top left
                    HStack(spacing: 10) {
                        CircleButton(label: "MENU", isPressed: .constant(false)) {
                            onMenuTap()
                        }
                        .frame(width: 50, height: 50)

                        CircleButton(label: "PAUSE", isPressed: $controlState.pause) {
                            triggerHaptic()
                        }
                        .frame(width: 50, height: 50)
                    }

                    Spacer()

                    // D-Pad centered vertically (same size as portrait)
                    DPadView(
                        up: $controlState.up,
                        down: $controlState.down,
                        left: $controlState.left,
                        right: $controlState.right,
                        onStateChange: { triggerHaptic() }
                    )
                    .frame(width: 150, height: 150)

                    Spacer()
                }
                .padding(.leading, 60)  // More padding to avoid notch
                .padding(.top, 10)

                Spacer()
            }

            // Right side - Action buttons aligned with D-Pad
            HStack {
                Spacer()

                VStack {
                    Spacer()

                    // Action buttons side by side (same size as portrait)
                    HStack(spacing: 10) {
                        CircleButton(label: "1", isPressed: $controlState.buttonA) {
                            triggerHaptic()
                        }
                        .frame(width: 70, height: 70)

                        CircleButton(label: "2", isPressed: $controlState.buttonB) {
                            triggerHaptic()
                        }
                        .frame(width: 70, height: 70)
                    }
                    .offset(y: 35)

                    Spacer()
                }
                .padding(.trailing, 20)
            }
        }
    }

    @ViewBuilder
    private func portraitLayout(size: CGSize) -> some View {
        // Calculate where gameplay ends
        // Game aspect ratio is ~256:224, gameplay is shifted up by 40%
        let gameAspect: CGFloat = 256.0 / 224.0
        let viewAspect = size.width / size.height
        let scaleY: CGFloat = viewAspect > gameAspect ? 1.0 : viewAspect / gameAspect
        let gameHeight = size.height * scaleY
        let topBlackBar = (size.height - gameHeight) / 2
        let gameShiftUp = (size.height - gameHeight) * 0.4 / 2  // Match Metal renderer offset
        let gameBottom = topBlackBar + gameHeight - gameShiftUp

        ZStack(alignment: .topLeading) {
            // Menu and Pause buttons below gameplay on the left
            HStack(spacing: 15) {
                CircleButton(label: "MENU", isPressed: .constant(false)) {
                    onMenuTap()
                }
                .frame(width: 50, height: 50)

                CircleButton(label: "PAUSE", isPressed: $controlState.pause) {
                    triggerHaptic()
                }
                .frame(width: 50, height: 50)
            }
            .padding(.leading, 20)
            .padding(.top, gameBottom + 15)

            // Controls at bottom
            VStack {
                Spacer()

                HStack(alignment: .center) {
                    // D-Pad
                    DPadView(
                        up: $controlState.up,
                        down: $controlState.down,
                        left: $controlState.left,
                        right: $controlState.right,
                        onStateChange: { triggerHaptic() }
                    )
                    .frame(width: 150, height: 150)
                    .padding(.leading, 20)

                    Spacer()

                    // Action buttons side by side
                    HStack(spacing: 10) {
                        CircleButton(label: "1", isPressed: $controlState.buttonA) {
                            triggerHaptic()
                        }
                        .frame(width: 70, height: 70)

                        CircleButton(label: "2", isPressed: $controlState.buttonB) {
                            triggerHaptic()
                        }
                        .frame(width: 70, height: 70)
                    }
                    .padding(.trailing, 20)
                }
                .padding(.bottom, 80)
            }
        }
    }

    private func triggerHaptic() {
        impactGenerator.impactOccurred()
    }
}

/// D-Pad control
struct DPadView: View {
    @Binding var up: Bool
    @Binding var down: Bool
    @Binding var left: Bool
    @Binding var right: Bool
    var onStateChange: () -> Void

    @State private var dragLocation: CGPoint = .zero

    var body: some View {
        GeometryReader { geometry in
            let size = min(geometry.size.width, geometry.size.height)
            let center = CGPoint(x: size / 2, y: size / 2)
            let buttonSize = size / 3

            ZStack {
                // Background circle
                Circle()
                    .fill(Color.black.opacity(0.4))

                // D-pad shape
                DPadShape()
                    .fill(Color.gray.opacity(0.6))
                    .frame(width: size * 0.9, height: size * 0.9)

                // Direction indicators
                VStack(spacing: buttonSize * 0.8) {
                    DirectionIndicator(direction: "U", isPressed: up)
                        .frame(width: buttonSize * 0.5, height: buttonSize * 0.5)
                    Spacer()
                    DirectionIndicator(direction: "D", isPressed: down)
                        .frame(width: buttonSize * 0.5, height: buttonSize * 0.5)
                }
                .frame(height: size * 0.9)

                HStack(spacing: buttonSize * 0.8) {
                    DirectionIndicator(direction: "L", isPressed: left)
                        .frame(width: buttonSize * 0.5, height: buttonSize * 0.5)
                    Spacer()
                    DirectionIndicator(direction: "R", isPressed: right)
                        .frame(width: buttonSize * 0.5, height: buttonSize * 0.5)
                }
                .frame(width: size * 0.9)
            }
            .gesture(
                DragGesture(minimumDistance: 0)
                    .onChanged { value in
                        updateDirection(location: value.location, center: center, size: size)
                    }
                    .onEnded { _ in
                        clearAll()
                    }
            )
        }
    }

    private func updateDirection(location: CGPoint, center: CGPoint, size: CGFloat) {
        let deadzone = size * 0.15
        let dx = location.x - center.x
        let dy = location.y - center.y

        let wasPressed = up || down || left || right

        // Reset all directions
        up = false
        down = false
        left = false
        right = false

        // Check if within the d-pad bounds
        let distance = sqrt(dx * dx + dy * dy)
        if distance < deadzone {
            if wasPressed { onStateChange() }
            return
        }

        // Determine direction based on angle
        let angle = atan2(dy, dx)

        // 8-way with 45-degree zones
        // Right: -22.5 to 22.5 degrees
        // Down-Right: 22.5 to 67.5 degrees
        // etc.

        let degrees = angle * 180 / .pi

        if degrees >= -22.5 && degrees < 22.5 {
            right = true
        } else if degrees >= 22.5 && degrees < 67.5 {
            right = true
            down = true
        } else if degrees >= 67.5 && degrees < 112.5 {
            down = true
        } else if degrees >= 112.5 && degrees < 157.5 {
            left = true
            down = true
        } else if degrees >= 157.5 || degrees < -157.5 {
            left = true
        } else if degrees >= -157.5 && degrees < -112.5 {
            left = true
            up = true
        } else if degrees >= -112.5 && degrees < -67.5 {
            up = true
        } else if degrees >= -67.5 && degrees < -22.5 {
            right = true
            up = true
        }

        let isPressed = up || down || left || right
        if isPressed != wasPressed {
            onStateChange()
        }
    }

    private func clearAll() {
        if up || down || left || right {
            up = false
            down = false
            left = false
            right = false
            onStateChange()
        }
    }
}

/// D-Pad cross shape
struct DPadShape: Shape {
    func path(in rect: CGRect) -> Path {
        var path = Path()

        let third = rect.width / 3

        // Top arm
        path.move(to: CGPoint(x: third, y: 0))
        path.addLine(to: CGPoint(x: third * 2, y: 0))
        path.addLine(to: CGPoint(x: third * 2, y: third))

        // Right arm
        path.addLine(to: CGPoint(x: rect.width, y: third))
        path.addLine(to: CGPoint(x: rect.width, y: third * 2))
        path.addLine(to: CGPoint(x: third * 2, y: third * 2))

        // Bottom arm
        path.addLine(to: CGPoint(x: third * 2, y: rect.height))
        path.addLine(to: CGPoint(x: third, y: rect.height))
        path.addLine(to: CGPoint(x: third, y: third * 2))

        // Left arm
        path.addLine(to: CGPoint(x: 0, y: third * 2))
        path.addLine(to: CGPoint(x: 0, y: third))
        path.addLine(to: CGPoint(x: third, y: third))

        path.closeSubpath()

        return path
    }
}

/// Direction indicator arrow
struct DirectionIndicator: View {
    let direction: String
    let isPressed: Bool

    var body: some View {
        Text(direction)
            .font(.system(size: 12, weight: .bold))
            .foregroundColor(isPressed ? .white : .gray)
    }
}

/// Circular button for A, B, Pause, Menu
struct CircleButton: View {
    let label: String
    @Binding var isPressed: Bool
    var onTap: (() -> Void)?

    var body: some View {
        ZStack {
            Circle()
                .fill(isPressed ? Color.blue.opacity(0.8) : Color.gray.opacity(0.5))
                .overlay(
                    Circle()
                        .stroke(Color.white.opacity(0.3), lineWidth: 2)
                )

            Text(label)
                .font(.system(size: label.count > 1 ? 10 : 20, weight: .bold))
                .foregroundColor(.white)
        }
        .gesture(
            DragGesture(minimumDistance: 0)
                .onChanged { _ in
                    if !isPressed {
                        isPressed = true
                        onTap?()
                    }
                }
                .onEnded { _ in
                    isPressed = false
                }
        )
    }
}

// Preview
#Preview {
    ZStack {
        Color.black
        TouchControlsView(
            controlState: .constant(ControlState()),
            onMenuTap: {}
        )
    }
}
