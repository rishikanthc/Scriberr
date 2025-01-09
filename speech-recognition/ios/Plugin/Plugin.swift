import Capacitor
import Foundation
import Speech

extension AVAudioPCMBuffer {
    func toArray() -> [Float] {
        let ptr = self.floatChannelData?[0]
        let buf = UnsafeBufferPointer(start: ptr, count: Int(self.frameLength))
        return Array(buf)
    }
}

@objc(SpeechRecognition)
public class SpeechRecognition: CAPPlugin {

    let defaultMatches = 5
    let messageMissingPermission = "Missing permission"
    let messageAccessDenied = "User denied access to speech recognition"
    let messageRestricted = "Speech recognition restricted on this device"
    let messageNotDetermined = "Speech recognition not determined on this device"
    let messageAccessDeniedMicrophone = "User denied access to microphone"
    let messageOngoing = "Ongoing speech recognition"
    let messageUnknown = "Unknown error occured"

    private var speechRecognizer: SFSpeechRecognizer?
    private var audioEngine: AVAudioEngine?
    private var recognitionRequest: SFSpeechAudioBufferRecognitionRequest?
    private var recognitionTask: SFSpeechRecognitionTask?
    private var audioRecorder: AVAudioRecorder?

    private var microphoneNode: AVAudioInputNode?

    @objc func startMicrophoneStream(_ call: CAPPluginCall) {
        let audioSession = AVAudioSession.sharedInstance()

        do {
            try audioSession.setCategory(.playAndRecord, options: .defaultToSpeaker)
            try audioSession.setActive(true)

            if audioEngine == nil {
                audioEngine = AVAudioEngine()
            }

            guard let audioEngine = audioEngine else {
                call.reject("Failed to create audio engine")
                return
            }

            microphoneNode = audioEngine.inputNode
            let format = microphoneNode?.outputFormat(forBus: 0)

            // Install tap on microphone node to get real-time audio data
            microphoneNode?.installTap(onBus: 0, bufferSize: 1024, format: format) {
                (buffer, time) in
                // Send audio buffer data to JavaScript
                let data = ["buffer": buffer.toArray()]
                self.notifyListeners("audioData", data: data)
            }

            try audioEngine.start()
            call.resolve()

        } catch {
            call.reject("Failed to start microphone: \(error.localizedDescription)")
        }
    }

    @objc func stopMicrophoneStream(_ call: CAPPluginCall) {
        if let node = microphoneNode {
            node.removeTap(onBus: 0)
        }

        audioEngine?.stop()
        microphoneNode = nil

        call.resolve()
    }

    @objc func record(_ call: CAPPluginCall) {
        let audioSession = AVAudioSession.sharedInstance()

        do {
            try audioSession.setCategory(.playAndRecord, options: .defaultToSpeaker)
            try audioSession.setActive(true)

            // Use Documents directory
            let documentsPath = FileManager.default.urls(
                for: .documentDirectory, in: .userDomainMask)[0]
            let fileName = call.getString("fileName") ?? "recording.wav"
            let audioFilename = documentsPath.appendingPathComponent(fileName)

            // Create parent directory if it doesn't exist
            try FileManager.default.createDirectory(
                at: documentsPath, withIntermediateDirectories: true)

            let settings: [String: Any] = [
                AVFormatIDKey: Int(kAudioFormatLinearPCM),
                AVSampleRateKey: 44100.0,
                AVNumberOfChannelsKey: 1,
                AVLinearPCMBitDepthKey: 16,
                AVLinearPCMIsFloatKey: false,
                AVLinearPCMIsBigEndianKey: false,
                AVEncoderAudioQualityKey: AVAudioQuality.high.rawValue,
            ]

            // Log the file path for debugging
            print("Recording to path: \(audioFilename.path)")

            // Create and start the audio recorder
            audioRecorder = try AVAudioRecorder(url: audioFilename, settings: settings)
            audioRecorder?.record()

            // Return the file path to the caller
            call.resolve([
                "path": audioFilename.path
            ])

        } catch {
            call.reject("Failed to start recording: \(error.localizedDescription)")
        }
    }

    @objc func stopRecording(_ call: CAPPluginCall) {
        guard let recorder = audioRecorder, recorder.isRecording else {
            call.reject("No active recording")
            return
        }

        recorder.stop()
        audioRecorder = nil
        call.resolve()
    }

    @objc func available(_ call: CAPPluginCall) {
        guard let recognizer = SFSpeechRecognizer() else {
            call.resolve([
                "available": false
            ])
            return
        }
        call.resolve([
            "available": recognizer.isAvailable
        ])
    }

    @objc func start(_ call: CAPPluginCall) {
        if self.audioEngine != nil {
            if self.audioEngine!.isRunning {
                call.reject(self.messageOngoing)
                return
            }
        }

        let status: SFSpeechRecognizerAuthorizationStatus = SFSpeechRecognizer.authorizationStatus()
        if status != SFSpeechRecognizerAuthorizationStatus.authorized {
            call.reject(self.messageMissingPermission)
            return
        }

        AVAudioSession.sharedInstance().requestRecordPermission { (granted) in
            if !granted {
                call.reject(self.messageAccessDeniedMicrophone)
                return
            }

            let language: String = call.getString("language") ?? "en-US"
            let maxResults: Int = call.getInt("maxResults") ?? self.defaultMatches
            let partialResults: Bool = call.getBool("partialResults") ?? false

            if self.recognitionTask != nil {
                self.recognitionTask?.cancel()
                self.recognitionTask = nil
            }

            self.audioEngine = AVAudioEngine.init()
            self.speechRecognizer = SFSpeechRecognizer.init(locale: Locale(identifier: language))

            let audioSession: AVAudioSession = AVAudioSession.sharedInstance()
            do {
                try audioSession.setCategory(
                    AVAudioSession.Category.playAndRecord,
                    options: AVAudioSession.CategoryOptions.defaultToSpeaker)
                try audioSession.setMode(AVAudioSession.Mode.default)
                try audioSession.setActive(
                    true, options: AVAudioSession.SetActiveOptions.notifyOthersOnDeactivation)
            } catch {

            }

            self.recognitionRequest = SFSpeechAudioBufferRecognitionRequest()
            self.recognitionRequest?.shouldReportPartialResults = partialResults

            let inputNode: AVAudioInputNode = self.audioEngine!.inputNode
            let format: AVAudioFormat = inputNode.outputFormat(forBus: 0)

            self.recognitionTask = self.speechRecognizer?.recognitionTask(
                with: self.recognitionRequest!,
                resultHandler: { (result, error) in
                    if result != nil {
                        let resultArray: NSMutableArray = NSMutableArray()
                        var counter: Int = 0

                        for transcription: SFTranscription in result!.transcriptions {
                            if maxResults > 0 && counter < maxResults {
                                resultArray.add(transcription.formattedString)
                            }
                            counter += 1
                        }

                        if partialResults {
                            self.notifyListeners("partialResults", data: ["matches": resultArray])
                        } else {
                            call.resolve([
                                "matches": resultArray
                            ])
                        }

                        if result!.isFinal {
                            self.audioEngine!.stop()
                            self.audioEngine?.inputNode.removeTap(onBus: 0)
                            self.notifyListeners("listeningState", data: ["status": "stopped"])
                            self.recognitionTask = nil
                            self.recognitionRequest = nil
                        }
                    }

                    if error != nil {
                        self.audioEngine!.stop()
                        self.audioEngine?.inputNode.removeTap(onBus: 0)
                        self.recognitionRequest = nil
                        self.recognitionTask = nil
                        self.notifyListeners("listeningState", data: ["status": "stopped"])
                        call.reject(error!.localizedDescription)
                    }
                })

            inputNode.installTap(onBus: 0, bufferSize: 1024, format: format) {
                (buffer: AVAudioPCMBuffer, _: AVAudioTime) in
                self.recognitionRequest?.append(buffer)
            }

            self.audioEngine?.prepare()
            do {
                try self.audioEngine?.start()
                self.notifyListeners("listeningState", data: ["status": "started"])
                if partialResults {
                    call.resolve()
                }
            } catch {
                call.reject(self.messageUnknown)
            }
        }
    }

    @objc func stop(_ call: CAPPluginCall) {
        DispatchQueue.global(qos: DispatchQoS.QoSClass.default).async {
            if let engine = self.audioEngine, engine.isRunning {
                engine.stop()
                self.recognitionRequest?.endAudio()
                self.notifyListeners("listeningState", data: ["status": "stopped"])
            }
            call.resolve()
        }
    }

    @objc func isListening(_ call: CAPPluginCall) {
        let isListening = self.audioEngine?.isRunning ?? false
        call.resolve([
            "listening": isListening
        ])
    }

    @objc func getSupportedLanguages(_ call: CAPPluginCall) {
        let supportedLanguages: Set<Locale>! = SFSpeechRecognizer.supportedLocales() as Set<Locale>
        let languagesArr: NSMutableArray = NSMutableArray()

        for lang: Locale in supportedLanguages {
            languagesArr.add(lang.identifier)
        }

        call.resolve([
            "languages": languagesArr
        ])
    }

    @objc override public func checkPermissions(_ call: CAPPluginCall) {
        let status: SFSpeechRecognizerAuthorizationStatus = SFSpeechRecognizer.authorizationStatus()
        let permission: String
        switch status {
        case .authorized:
            permission = "granted"
        case .denied, .restricted:
            permission = "denied"
        case .notDetermined:
            permission = "prompt"
        @unknown default:
            permission = "prompt"
        }
        call.resolve(["speechRecognition": permission])
    }

    @objc override public func requestPermissions(_ call: CAPPluginCall) {
        SFSpeechRecognizer.requestAuthorization { (status: SFSpeechRecognizerAuthorizationStatus) in
            DispatchQueue.main.async {
                switch status {
                case .authorized:
                    AVAudioSession.sharedInstance().requestRecordPermission { (granted: Bool) in
                        if granted {
                            call.resolve(["speechRecognition": "granted"])
                        } else {
                            call.resolve(["speechRecognition": "denied"])
                        }
                    }
                    break
                case .denied, .restricted, .notDetermined:
                    self.checkPermissions(call)
                    break
                @unknown default:
                    self.checkPermissions(call)
                }
            }
        }
    }
}
