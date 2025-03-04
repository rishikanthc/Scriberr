package com.getcapacitor.community.speechrecognition;

import android.Manifest;

public interface Constants {
    int REQUEST_CODE_PERMISSION = 2001;
    int REQUEST_CODE_SPEECH = 2002;
    String IS_RECOGNITION_AVAILABLE = "isRecognitionAvailable";
    String START_LISTENING = "startListening";
    String STOP_LISTENING = "stopListening";
    String GET_SUPPORTED_LANGUAGES = "getSupportedLanguages";
    String HAS_PERMISSION = "hasPermission";
    String REQUEST_PERMISSION = "requestPermission";
    int MAX_RESULTS = 5;
    String NOT_AVAILABLE = "Speech recognition service is not available.";
    String MISSING_PERMISSION = "Missing permission";
    String RECORD_AUDIO_PERMISSION = Manifest.permission.RECORD_AUDIO;
    String ERROR = "Could not get list of languages";
}
