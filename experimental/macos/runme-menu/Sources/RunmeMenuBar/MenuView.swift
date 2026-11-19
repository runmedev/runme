import SwiftUI

struct MenuView: View {
    @EnvironmentObject var supervisor: AgentSupervisor

    var body: some View {
        Text("Runme: \(supervisor.summaryState.rawValue)")
        Text(supervisor.summaryMessage)

        Divider()

        if supervisor.configs.isEmpty {
            Text("No config*.yaml found")
        } else {
            ForEach(supervisor.configs) { cfg in
                let status = supervisor.status(for: cfg)
                Menu("\(cfg.fileName) (\(status.state.rawValue))") {
                    Text(status.message)
                    Divider()

                    if supervisor.isRunning(cfg) {
                        Button("Stop") {
                            supervisor.stop(cfg)
                        }
                    } else {
                        Button("Start") {
                            supervisor.start(cfg)
                        }
                    }

                    Button("Open UI") {
                        supervisor.openWebUI(cfg)
                    }

                    Button("Open Log") {
                        supervisor.openLogFile(cfg)
                    }
                }
            }
        }

        Divider()

        Button("Refresh Configs") {
            supervisor.refreshConfigs()
        }

        Button("Quit") {
            NSApplication.shared.terminate(nil)
        }
    }
}
