import SwiftUI

struct MenuView: View {
    @EnvironmentObject var supervisor: AgentSupervisor

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            Text("Runme Agent")
                .font(.headline)
            Text(supervisor.state.rawValue)
                .font(.subheadline)
            Text(supervisor.statusMessage)
                .font(.caption)
                .foregroundStyle(.secondary)
                .fixedSize(horizontal: false, vertical: true)

            Divider()

            Button("Open Runme UI") {
                supervisor.openWebUI()
            }

            if supervisor.isRunning {
                Button("Stop Server") {
                    supervisor.stop()
                }
            } else {
                Button("Start Server") {
                    supervisor.start()
                }
            }

            Button("Open Log File") {
                supervisor.openLogFile()
            }

            Divider()

            Button("Quit") {
                NSApplication.shared.terminate(nil)
            }
        }
        .frame(width: 300)
        .padding()
    }
}
