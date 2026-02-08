import Foundation
import os.log

/// Centralized logging for the app using os.log
/// Logs can be viewed in Console.app or with `log stream --predicate 'subsystem == "com.emkiii"'`
enum Log {
    private static let subsystem = "com.emkiii"

    /// Storage operations (library, config, save states)
    static let storage = Logger(subsystem: subsystem, category: "storage")

    /// Emulator operations
    static let emulator = Logger(subsystem: subsystem, category: "emulator")

    /// Network operations (RDB download, artwork)
    static let network = Logger(subsystem: subsystem, category: "network")

    /// ROM import operations
    static let romImport = Logger(subsystem: subsystem, category: "import")
}
