import AppKit
import Foundation

enum AgentState: String {
    case stopped = "Stopped"
    case starting = "Starting"
    case running = "Running"
    case error = "Error"

    var iconName: String {
        switch self {
        case .stopped:
            return "pause.circle"
        case .starting:
            return "hourglass.circle"
        case .running:
            return "play.circle"
        case .error:
            return "exclamationmark.triangle"
        }
    }
}

@MainActor
final class AgentSupervisor: ObservableObject {
    @Published private(set) var state: AgentState = .stopped
    @Published private(set) var statusMessage = "Not running"

    private var process: Process?
    private var healthTimer: Timer?

    var endpointURL: URL {
        let raw = ProcessInfo.processInfo.environment["RUNME_ENDPOINT"] ?? "http://127.0.0.1:8080"
        return URL(string: raw) ?? URL(string: "http://127.0.0.1:8080")!
    }

    var isRunning: Bool {
        process?.isRunning == true
    }

    func start() {
        guard process == nil else {
            statusMessage = "Agent is already running"
            return
        }

        do {
            let runmeURL = try resolveRunmeBinary()
            let configPath = resolveConfigPath()
            let logURL = try ensureLogFile()
            let handle = try FileHandle(forWritingTo: logURL)
            try handle.seekToEnd()

            let proc = Process()
            proc.executableURL = runmeURL
            proc.arguments = [
                "agent",
                "serve",
                "--config",
                configPath,
            ]
            proc.standardOutput = handle
            proc.standardError = handle
            proc.terminationHandler = { [weak self] terminated in
                Task { @MainActor in
                    self?.process = nil
                    self?.stopHealthChecks()
                    if terminated.terminationStatus == 0 {
                        self?.state = .stopped
                        self?.statusMessage = "Agent stopped"
                    } else {
                        self?.state = .error
                        self?.statusMessage = "Agent exited with status \(terminated.terminationStatus)"
                    }
                }
            }

            state = .starting
            statusMessage = "Starting runme agent serve"
            try proc.run()
            process = proc
            startHealthChecks()
        } catch {
            state = .error
            statusMessage = "Start failed: \(error.localizedDescription)"
        }
    }

    func stop() {
        guard let proc = process else {
            state = .stopped
            statusMessage = "Agent is not running"
            return
        }

        proc.interrupt()
        state = .stopped
        statusMessage = "Stopping agent"
    }

    func openWebUI() {
        NSWorkspace.shared.open(endpointURL)
    }

    func openLogFile() {
        do {
            let logURL = try ensureLogFile()
            NSWorkspace.shared.open(logURL)
        } catch {
            state = .error
            statusMessage = "Failed to open log file: \(error.localizedDescription)"
        }
    }

    private func startHealthChecks() {
        stopHealthChecks()
        healthTimer = Timer.scheduledTimer(withTimeInterval: 2.0, repeats: true) { [weak self] _ in
            Task { @MainActor in
                self?.checkHealth()
            }
        }
    }

    private func stopHealthChecks() {
        healthTimer?.invalidate()
        healthTimer = nil
    }

    private func checkHealth() {
        guard process?.isRunning == true else {
            return
        }

        var request = URLRequest(url: endpointURL.appendingPathComponent("metrics"))
        request.timeoutInterval = 1.0

        URLSession.shared.dataTask(with: request) { [weak self] _, response, error in
            Task { @MainActor in
                guard let self else {
                    return
                }
                if error != nil {
                    self.state = .starting
                    self.statusMessage = "Agent is starting"
                    return
                }

                if let http = response as? HTTPURLResponse, (200..<500).contains(http.statusCode) {
                    self.state = .running
                    self.statusMessage = "Agent running at \(self.endpointURL.absoluteString)"
                } else {
                    self.state = .starting
                    self.statusMessage = "Agent is starting"
                }
            }
        }.resume()
    }

    private func resolveRunmeBinary() throws -> URL {
        if let override = ProcessInfo.processInfo.environment["RUNME_BIN"], !override.isEmpty {
            return URL(fileURLWithPath: NSString(string: override).expandingTildeInPath)
        }

        if let bundled = Bundle.module.url(forResource: "runme", withExtension: nil, subdirectory: "Resources")
            ?? Bundle.module.url(forResource: "runme", withExtension: nil),
            FileManager.default.isExecutableFile(atPath: bundled.path)
        {
            return bundled
        }

        if let bundled = Bundle.main.url(forResource: "runme", withExtension: nil),
            FileManager.default.isExecutableFile(atPath: bundled.path)
        {
            return bundled
        }

        let candidates = [
            "/opt/homebrew/bin/runme",
            "/usr/local/bin/runme",
            "/usr/bin/runme",
        ]

        for candidate in candidates {
            if FileManager.default.isExecutableFile(atPath: candidate) {
                return URL(fileURLWithPath: candidate)
            }
        }

        throw NSError(
            domain: "RunmeMenuBar",
            code: 1,
            userInfo: [NSLocalizedDescriptionKey: "Could not find runme binary. Set RUNME_BIN or install runme."]
        )
    }

    private func resolveConfigPath() -> String {
        if let override = ProcessInfo.processInfo.environment["RUNME_CONFIG"], !override.isEmpty {
            return NSString(string: override).expandingTildeInPath
        }
        return NSString(string: "~/.runme-agent/config.yaml").expandingTildeInPath
    }

    private func ensureLogFile() throws -> URL {
        if let override = ProcessInfo.processInfo.environment["RUNME_LOG_PATH"], !override.isEmpty {
            let path = NSString(string: override).expandingTildeInPath
            let url = URL(fileURLWithPath: path)
            let parent = url.deletingLastPathComponent()
            try FileManager.default.createDirectory(at: parent, withIntermediateDirectories: true)
            if !FileManager.default.fileExists(atPath: path) {
                FileManager.default.createFile(atPath: path, contents: nil)
            }
            return url
        }

        let base = FileManager.default.urls(for: .libraryDirectory, in: .userDomainMask).first!
        let dir = base.appendingPathComponent("Logs/Runme", isDirectory: true)
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        let file = dir.appendingPathComponent("agent.log")
        if !FileManager.default.fileExists(atPath: file.path) {
            FileManager.default.createFile(atPath: file.path, contents: nil)
        }
        return file
    }
}
