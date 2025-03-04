import XCTest
import Capacitor
@testable import Plugin

class PluginTests: XCTestCase {

    func testEcho() {
        // This is an example of a functional test case for a plugin.
        // Use XCTAssert and related functions to verify your tests produce the correct results.

        let value = "Hello, World!"
        let plugin = MyPlugin()

        let call = CAPPluginCall(callbackId: "test", options: [
            "value": value
        ], resolve: { (result, _) in
            let resultValue = result!.data["value"] as? String
            XCTAssertEqual(value, resultValue)
        }, reject: { (_) in
            XCTFail("Error shouldn't have been called")
        })

        plugin.echo(call!)
    }
}
