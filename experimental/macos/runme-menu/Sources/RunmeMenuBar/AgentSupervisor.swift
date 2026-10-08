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

struct ConfigStatus {
    var state: AgentState
    var message: String
}

struct AgentConfig: Identifiable, Hashable {
    let path: String
    let endpointURL: URL
    let logURL: URL

    var id: String { path }
    var fileName: String { URL(fileURLWithPath: path).lastPathComponent }
}

private final class RunningProcess {
    let process: Process
    let logHandle: FileHandle

    init(process: Process, logHandle: FileHandle) {
        self.process = process
        self.logHandle = logHandle
    }
}

@MainActor
final class AgentSupervisor: ObservableObject {
    @Published private(set) var configs: [AgentConfig] = []
    @Published private(set) var summaryState: AgentState = .stopped
    @Published private(set) var summaryMessage = "No configs found"

    private var running: [String: RunningProcess] = [:]
    private var statusByConfigPath: [String: ConfigStatus] = [:]
    private var stoppingConfigPaths: Set<String> = []
    private var healthTimer: Timer?

    init() {
        refreshConfigs()
        startHealthChecks()
    }

    deinit {
        healthTimer?.invalidate()
    }

    func refreshConfigs() {
        let discovered = discoverConfigPaths()
        configs = discovered.map { configPath in
            let endpoint = endpointForConfig(at: configPath)
            let logURL = logURL(forConfigPath: configPath)
            return AgentConfig(path: configPath, endpointURL: endpoint, logURL: logURL)
        }

        let known = Set(configs.map(\.path))
        statusByConfigPath = statusByConfigPath.filter { known.contains($0.key) }
        for cfg in configs where statusByConfigPath[cfg.path] == nil {
            statusByConfigPath[cfg.path] = ConfigStatus(state: .stopped, message: "Not running")
        }
        recalculateSummary()
    }

    func status(for config: AgentConfig) -> ConfigStatus {
        statusByConfigPath[config.path] ?? ConfigStatus(state: .stopped, message: "Not running")
    }

    func isRunning(_ config: AgentConfig) -> Bool {
        running[config.path]?.process.isRunning == true
    }

    func start(_ config: AgentConfig) {
        guard running[config.path] == nil else {
            statusByConfigPath[config.path] = ConfigStatus(state: .running, message: "Already running")
            recalculateSummary()
            return
        }

        // Prototype policy: single active runner to avoid conflicts on shared ports.
        for path in running.keys where path != config.path {
            stopByPath(path)
        }

        do {
            let runmeURL = try resolveRunmeBinary()
            let logURL = try ensureLogFile(at: config.logURL)
            let handle = try FileHandle(forWritingTo: logURL)
            try handle.seekToEnd()

            let proc = Process()
            proc.executableURL = runmeURL
            proc.arguments = ["agent", "serve", "--config", config.path]
            proc.standardOutput = handle
            proc.standardError = handle
            proc.terminationHandler = { [weak self] terminated in
                Task { @MainActor in
                    self?.handleTermination(configPath: config.path, process: terminated)
                }
            }

            statusByConfigPath[config.path] = ConfigStatus(state: .starting, message: "Starting server")
            try proc.run()
            running[config.path] = RunningProcess(process: proc, logHandle: handle)
            recalculateSummary()
        } catch {
            statusByConfigPath[config.path] = ConfigStatus(state: .error, message: "Start failed: \(error.localizedDescription)")
            recalculateSummary()
        }
    }

    func stop(_ config: AgentConfig) {
        stopByPath(config.path)
    }

    func openWebUI(_ config: AgentConfig) {
        NSWorkspace.shared.open(config.endpointURL)
    }

    func openLogFile(_ config: AgentConfig) {
        do {
            let logURL = try ensureLogFile(at: config.logURL)
            NSWorkspace.shared.open(logURL)
        } catch {
            statusByConfigPath[config.path] = ConfigStatus(state: .error, message: "Failed to open log file: \(error.localizedDescription)")
            recalculateSummary()
        }
    }

    private func stopByPath(_ path: String) {
        guard let entry = running[path] else {
            if statusByConfigPath[path] == nil {
                statusByConfigPath[path] = ConfigStatus(state: .stopped, message: "Not running")
            }
            recalculateSummary()
            return
        }

        stoppingConfigPaths.insert(path)
        statusByConfigPath[path] = ConfigStatus(state: .stopped, message: "Stopping server")
        entry.process.interrupt()
        recalculateSummary()
    }

    private func handleTermination(configPath: String, process: Process) {
        let wasStopping = stoppingConfigPaths.remove(configPath) != nil
        if let entry = running.removeValue(forKey: configPath) {
            try? entry.logHandle.close()
        }

        if wasStopping || process.terminationStatus == 0 {
            statusByConfigPath[configPath] = ConfigStatus(state: .stopped, message: "Stopped")
        } else {
            statusByConfigPath[configPath] = ConfigStatus(
                state: .error,
                message: "Exited with status \(process.terminationStatus)"
            )
        }
        recalculateSummary()
    }

    private func startHealthChecks() {
        healthTimer?.invalidate()
        healthTimer = Timer.scheduledTimer(withTimeInterval: 2.0, repeats: true) { [weak self] _ in
            Task { @MainActor in
                self?.checkHealth()
            }
        }
    }

    private func checkHealth() {
        for cfg in configs {
            guard let entry = running[cfg.path], entry.process.isRunning else {
                continue
            }

            var request = URLRequest(url: cfg.endpointURL.appendingPathComponent("metrics"))
            request.timeoutInterval = 1.0

            URLSession.shared.dataTask(with: request) { [weak self] _, response, error in
                Task { @MainActor in
                    guard let self else { return }
                    guard self.running[cfg.path] != nil else { return }

                    if error != nil {
                        self.statusByConfigPath[cfg.path] = ConfigStatus(state: .starting, message: "Starting server")
                        self.recalculateSummary()
                        return
                    }

                    if let http = response as? HTTPURLResponse, (200..<500).contains(http.statusCode) {
                        self.statusByConfigPath[cfg.path] = ConfigStatus(
                            state: .running,
                            message: "Running at \(cfg.endpointURL.absoluteString)"
                        )
                    } else {
                        self.statusByConfigPath[cfg.path] = ConfigStatus(state: .starting, message: "Starting server")
                    }
                    self.recalculateSummary()
                }
            }.resume()
        }
    }

    private func recalculateSummary() {
        if configs.isEmpty {
            summaryState = .stopped
            summaryMessage = "No config*.yaml files found in \(configDirectoryPath())"
            return
        }

        let statuses = configs.map { statusByConfigPath[$0.path]?.state ?? .stopped }

        if statuses.contains(.error) {
            summaryState = .error
            summaryMessage = "One or more configs errored"
            return
        }
        if statuses.contains(.running) {
            summaryState = .running
            summaryMessage = "At least one config running"
            return
        }
        if statuses.contains(.starting) {
            summaryState = .starting
            summaryMessage = "Starting server"
            return
        }

        summaryState = .stopped
        summaryMessage = "All configs stopped"
    }

    private func discoverConfigPaths() -> [String] {
        let fm = FileManager.default
        let base = configDirectoryPath()
        let baseURL = URL(fileURLWithPath: base, isDirectory: true)

        let filenames: [String]
        do {
            filenames = try fm.contentsOfDirectory(atPath: base)
        } catch {
            return []
        }

        let pattern = try? NSRegularExpression(pattern: #"^config.*\.ya?ml$"#, options: [.caseInsensitive])
        var paths: [String] = []

        for file in filenames {
            let range = NSRange(location: 0, length: file.utf16.count)
            guard pattern?.firstMatch(in: file, options: [], range: range) != nil else {
                continue
            }
            let full = baseURL.appendingPathComponent(file).path
            var isDir: ObjCBool = false
            guard fm.fileExists(atPath: full, isDirectory: &isDir), !isDir.boolValue else {
                continue
            }
            paths.append(full)
        }

        if let explicit = ProcessInfo.processInfo.environment["RUNME_CONFIG"], !explicit.isEmpty {
            let expanded = NSString(string: explicit).expandingTildeInPath
            if fm.fileExists(atPath: expanded), !paths.contains(expanded) {
                paths.append(expanded)
            }
        }

        return paths.sorted()
    }

    private func configDirectoryPath() -> String {
        if let override = ProcessInfo.processInfo.environment["RUNME_CONFIG_DIR"], !override.isEmpty {
            return NSString(string: override).expandingTildeInPath
        }
        return NSString(string: "~/.runme-agent").expandingTildeInPath
    }

    private func endpointForConfig(at path: String) -> URL {
        if let override = ProcessInfo.processInfo.environment["RUNME_ENDPOINT"], !override.isEmpty,
            let url = URL(string: override)
        {
            return url
        }

        let defaultURL = URL(string: "http://127.0.0.1:8080")!
        guard let data = try? String(contentsOfFile: path, encoding: .utf8) else {
            return defaultURL
        }

        let settings = parseAssistantServerSettings(from: data)
        let host = normalizeHost(settings.bindAddress)
        let port = settings.port ?? 8080
        return URL(string: "http://\(host):\(port)") ?? defaultURL
    }

    private func parseAssistantServerSettings(from data: String) -> (bindAddress: String?, port: Int?) {
        var bind: String?
        var port: Int?
        var inAssistantServer = false
        var assistantIndent = 0

        let lines = data.components(separatedBy: .newlines)
        for raw in lines {
            let noComment = raw.split(separator: "#", maxSplits: 1, omittingEmptySubsequences: false).first.map(String.init) ?? raw
            if noComment.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                continue
            }

            let indent = noComment.prefix { $0 == " " || $0 == "\t" }.count
            let trimmed = noComment.trimmingCharacters(in: .whitespacesAndNewlines)

            if !inAssistantServer {
                if trimmed.hasPrefix("assistantServer:") {
                    inAssistantServer = true
                    assistantIndent = indent
                    continue
                }
                continue
            }

            if indent <= assistantIndent {
                inAssistantServer = false
                continue
            }

            guard let colon = trimmed.firstIndex(of: ":") else {
                continue
            }

            let key = String(trimmed[..<colon]).trimmingCharacters(in: .whitespacesAndNewlines)
            let rawValue = String(trimmed[trimmed.index(after: colon)...]).trimmingCharacters(in: .whitespacesAndNewlines)
            let value = rawValue
                .trimmingCharacters(in: .whitespacesAndNewlines)
                .trimmingCharacters(in: CharacterSet(charactersIn: "\"'"))

            if key == "bindAddress", !value.isEmpty {
                bind = value
            } else if key == "port", let parsed = Int(value) {
                port = parsed
            }
        }

        return (bindAddress: bind, port: port)
    }

    private func normalizeHost(_ host: String?) -> String {
        guard let host, !host.isEmpty else {
            return "127.0.0.1"
        }

        switch host {
        case "0.0.0.0":
            return "127.0.0.1"
        case "::":
            return "localhost"
        default:
            return host
        }
    }

    private func logURL(forConfigPath configPath: String) -> URL {
        if let override = ProcessInfo.processInfo.environment["RUNME_LOG_PATH"], !override.isEmpty {
            return URL(fileURLWithPath: NSString(string: override).expandingTildeInPath)
        }

        let fileName = URL(fileURLWithPath: configPath).lastPathComponent
            .replacingOccurrences(of: "/", with: "_")
        let base = FileManager.default.urls(for: .libraryDirectory, in: .userDomainMask).first!
        return base
            .appendingPathComponent("Logs/Runme", isDirectory: true)
            .appendingPathComponent("\(fileName).log")
    }

    private func ensureLogFile(at url: URL) throws -> URL {
        let parent = url.deletingLastPathComponent()
        try FileManager.default.createDirectory(at: parent, withIntermediateDirectories: true)
        if !FileManager.default.fileExists(atPath: url.path) {
            FileManager.default.createFile(atPath: url.path, contents: nil)
        }
        return url
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

        for candidate in candidates where FileManager.default.isExecutableFile(atPath: candidate) {
            return URL(fileURLWithPath: candidate)
        }

        throw NSError(
            domain: "RunmeMenuBar",
            code: 1,
            userInfo: [NSLocalizedDescriptionKey: "Could not find runme binary. Set RUNME_BIN or run ./scripts/bundle-runme.sh."]
        )
    }
}
