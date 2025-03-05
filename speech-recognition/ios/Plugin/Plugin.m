#import <Foundation/Foundation.h>
#import <Capacitor/Capacitor.h>

// Define the plugin using the CAP_PLUGIN Macro, and
// each method the plugin supports using the CAP_PLUGIN_METHOD macro.
CAP_PLUGIN(SpeechRecognition, "SpeechRecognition",
        CAP_PLUGIN_METHOD(available, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(record, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(stopRecording, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(startMicrophoneStream, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(stopMicrophoneStream, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(start, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(stop, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(getSupportedLanguages, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(hasPermission, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(isListening, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(requestPermission, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(checkPermissions, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(requestPermissions, CAPPluginReturnPromise);
        CAP_PLUGIN_METHOD(removeAllListeners, CAPPluginReturnPromise);
)
