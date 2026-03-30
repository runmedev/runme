import AppKit
import SwiftUI

final class AppDelegate: NSObject, NSApplicationDelegate {
    func applicationDidFinishLaunching(_ notification: Notification) {
        NSApp.setActivationPolicy(.accessory)
    }
}

@main
struct RunmeMenuBarApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate
    @StateObject private var supervisor = AgentSupervisor()

    var body: some Scene {
        MenuBarExtra("Runme", systemImage: supervisor.state.iconName) {
            MenuView()
                .environmentObject(supervisor)
        }
        .menuBarExtraStyle(.window)

        Settings {
            EmptyView()
        }
    }
}
