# Capacitor Speech Recognition Plugin

Capacitor community plugin for speech recognition.

## Maintainers

| Maintainer      | GitHub                                      | Social                                           |
| --------------- | ------------------------------------------- | ------------------------------------------------ |
| Priyank Patel   | [priyankpat](https://github.com/priyankpat) | [@priyankpat\_](https://twitter.com/priyankpat_) |
| Matteo Padovano | [mrbatista](https://github.com/mrbatista)   | [@mrba7ista](https://twitter.com/mrba7ista)      |

Maintenance Status: Actively Maintained

## Installation

To use npm

```bash
npm install @capacitor-community/speech-recognition
```

To use yarn

```bash
yarn add @capacitor-community/speech-recognition
```

Sync native files

```bash
npx cap sync
```

## iOS

iOS requires the following usage descriptions be added and filled out for your app in `Info.plist`:

- `NSSpeechRecognitionUsageDescription` (`Privacy - Speech Recognition Usage Description`)
- `NSMicrophoneUsageDescription` (`Privacy - Microphone Usage Description`)

## Android

No further action required.

## Supported methods

<docgen-index>

* [`available()`](#available)
* [`record(...)`](#record)
* [`startMicrophoneStream()`](#startmicrophonestream)
* [`stopMicrophoneStream()`](#stopmicrophonestream)
* [`stopRecording()`](#stoprecording)
* [`start(...)`](#start)
* [`stop()`](#stop)
* [`getSupportedLanguages()`](#getsupportedlanguages)
* [`isListening()`](#islistening)
* [`checkPermissions()`](#checkpermissions)
* [`requestPermissions()`](#requestpermissions)
* [`addListener('partialResults', ...)`](#addlistenerpartialresults-)
* [`addListener('listeningState', ...)`](#addlistenerlisteningstate-)
* [`removeAllListeners()`](#removealllisteners)
* [Interfaces](#interfaces)
* [Type Aliases](#type-aliases)

</docgen-index>

## Example

```typescript
import { SpeechRecognition } from "@capacitor-community/speech-recognition";

SpeechRecognition.available();

SpeechRecognition.start({
  language: "en-US",
  maxResults: 2,
  prompt: "Say something",
  partialResults: true,
  popup: true,
});
// listen to partial results
SpeechRecognition.addListener("partialResults", (data: any) => {
  console.log("partialResults was fired", data.matches);
});

// stop listening partial results
SpeechRecognition.removeAllListeners();

SpeechRecognition.stop();

SpeechRecognition.getSupportedLanguages();

SpeechRecognition.checkPermissions();

SpeechRecognition.requestPermissions();

SpeechRecognition.hasPermission();

SpeechRecognition.requestPermission();
```

<docgen-api>
<!--Update the source file JSDoc comments and rerun docgen to update the docs below-->

### available()

```typescript
available() => Promise<{ available: boolean; }>
```

This method will check if speech recognition feature is available on the device.

**Returns:** <code>Promise&lt;{ available: boolean; }&gt;</code>

--------------------


### record(...)

```typescript
record(options?: RecordOptions | undefined) => Promise<RecordingResult>
```

Start recording audio to a file

| Param         | Type                                                    |
| ------------- | ------------------------------------------------------- |
| **`options`** | <code><a href="#recordoptions">RecordOptions</a></code> |

**Returns:** <code>Promise&lt;<a href="#recordingresult">RecordingResult</a>&gt;</code>

--------------------


### startMicrophoneStream()

```typescript
startMicrophoneStream() => Promise<RecordingResult>
```

Start streaming microphone data for visualization

**Returns:** <code>Promise&lt;<a href="#recordingresult">RecordingResult</a>&gt;</code>

--------------------


### stopMicrophoneStream()

```typescript
stopMicrophoneStream() => Promise<RecordingResult>
```

Stop streaming microphone data

**Returns:** <code>Promise&lt;<a href="#recordingresult">RecordingResult</a>&gt;</code>

--------------------


### stopRecording()

```typescript
stopRecording() => Promise<void>
```

--------------------


### start(...)

```typescript
start(options?: UtteranceOptions | undefined) => Promise<{ matches?: string[]; }>
```

This method will start to listen for utterance.

if `partialResults` is `true`, the function respond directly without result and
event `partialResults` will be emit for each partial result, until stopped.

| Param         | Type                                                          |
| ------------- | ------------------------------------------------------------- |
| **`options`** | <code><a href="#utteranceoptions">UtteranceOptions</a></code> |

**Returns:** <code>Promise&lt;{ matches?: string[]; }&gt;</code>

--------------------


### stop()

```typescript
stop() => Promise<void>
```

This method will stop listening for utterance

--------------------


### getSupportedLanguages()

```typescript
getSupportedLanguages() => Promise<{ languages: any[]; }>
```

This method will return list of languages supported by the speech recognizer.

It's not available on Android 13 and newer.

**Returns:** <code>Promise&lt;{ languages: any[]; }&gt;</code>

--------------------


### isListening()

```typescript
isListening() => Promise<{ listening: boolean; }>
```

This method will check if speech recognition is listening.

**Returns:** <code>Promise&lt;{ listening: boolean; }&gt;</code>

**Since:** 5.1.0

--------------------


### checkPermissions()

```typescript
checkPermissions() => Promise<PermissionStatus>
```

Check the speech recognition permission.

**Returns:** <code>Promise&lt;<a href="#permissionstatus">PermissionStatus</a>&gt;</code>

**Since:** 5.0.0

--------------------


### requestPermissions()

```typescript
requestPermissions() => Promise<PermissionStatus>
```

Request the speech recognition permission.

**Returns:** <code>Promise&lt;<a href="#permissionstatus">PermissionStatus</a>&gt;</code>

**Since:** 5.0.0

--------------------


### addListener('partialResults', ...)

```typescript
addListener(eventName: 'partialResults', listenerFunc: (data: { matches: string[]; }) => void) => Promise<PluginListenerHandle>
```

Called when partialResults set to true and result received.

On Android it doesn't work if popup is true.

Provides partial result.

| Param              | Type                                                   |
| ------------------ | ------------------------------------------------------ |
| **`eventName`**    | <code>'partialResults'</code>                          |
| **`listenerFunc`** | <code>(data: { matches: string[]; }) =&gt; void</code> |

**Returns:** <code>Promise&lt;<a href="#pluginlistenerhandle">PluginListenerHandle</a>&gt;</code>

**Since:** 2.0.2

--------------------


### addListener('listeningState', ...)

```typescript
addListener(eventName: 'listeningState', listenerFunc: (data: { status: 'started' | 'stopped'; }) => void) => Promise<PluginListenerHandle>
```

Called when listening state changed.

| Param              | Type                                                                |
| ------------------ | ------------------------------------------------------------------- |
| **`eventName`**    | <code>'listeningState'</code>                                       |
| **`listenerFunc`** | <code>(data: { status: 'started' \| 'stopped'; }) =&gt; void</code> |

**Returns:** <code>Promise&lt;<a href="#pluginlistenerhandle">PluginListenerHandle</a>&gt;</code>

**Since:** 5.1.0

--------------------


### removeAllListeners()

```typescript
removeAllListeners() => Promise<void>
```

Remove all the listeners that are attached to this plugin.

**Since:** 4.0.0

--------------------


### Interfaces


#### RecordingResult

| Prop         | Type                                | Description                     |
| ------------ | ----------------------------------- | ------------------------------- |
| **`path`**   | <code>string</code>                 | Path to the recorded audio file |
| **`status`** | <code>'started' \| 'stopped'</code> |                                 |


#### RecordOptions

| Prop             | Type                |
| ---------------- | ------------------- |
| **`fileName`**   | <code>string</code> |
| **`outputPath`** | <code>string</code> |


#### UtteranceOptions

| Prop                 | Type                 | Description                                                      |
| -------------------- | -------------------- | ---------------------------------------------------------------- |
| **`language`**       | <code>string</code>  | key returned from `getSupportedLanguages()`                      |
| **`maxResults`**     | <code>number</code>  | maximum number of results to return (5 is max)                   |
| **`prompt`**         | <code>string</code>  | prompt message to display on popup (Android only)                |
| **`popup`**          | <code>boolean</code> | display popup window when listening for utterance (Android only) |
| **`partialResults`** | <code>boolean</code> | return partial results if found                                  |


#### PermissionStatus

| Prop                    | Type                                                        | Description                                                                                                                                                                      | Since |
| ----------------------- | ----------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ----- |
| **`speechRecognition`** | <code><a href="#permissionstate">PermissionState</a></code> | Permission state for speechRecognition alias. On Android it requests/checks RECORD_AUDIO permission On iOS it requests/checks the speech recognition and microphone permissions. | 5.0.0 |


#### PluginListenerHandle

| Prop         | Type                                      |
| ------------ | ----------------------------------------- |
| **`remove`** | <code>() =&gt; Promise&lt;void&gt;</code> |


### Type Aliases


#### PermissionState

<code>'prompt' | 'prompt-with-rationale' | 'granted' | 'denied'</code>

</docgen-api>
