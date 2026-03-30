// swift-tools-version: 5.10
import PackageDescription

let package = Package(
    name: "RunmeMenuBar",
    platforms: [
        .macOS(.v13),
    ],
    products: [
        .executable(name: "RunmeMenuBar", targets: ["RunmeMenuBar"]),
    ],
    targets: [
        .executableTarget(
            name: "RunmeMenuBar"
        ),
    ]
)
