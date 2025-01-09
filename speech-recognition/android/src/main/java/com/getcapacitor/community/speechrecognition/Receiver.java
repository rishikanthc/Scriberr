package com.getcapacitor.community.speechrecognition;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.os.Bundle;
import android.speech.RecognizerIntent;
import com.getcapacitor.JSArray;
import com.getcapacitor.JSObject;
import com.getcapacitor.PluginCall;
import java.util.List;

public class Receiver extends BroadcastReceiver implements Constants {

    public static final String TAG = "Receiver";

    private List<String> supportedLanguagesList;
    private String languagePref;
    private PluginCall call;

    public Receiver(PluginCall call) {
        super();
        this.call = call;
    }

    @Override
    public void onReceive(Context context, Intent intent) {
        Bundle extras = getResultExtras(true);

        if (extras.containsKey(RecognizerIntent.EXTRA_LANGUAGE_PREFERENCE)) {
            languagePref = extras.getString(RecognizerIntent.EXTRA_LANGUAGE_PREFERENCE);
        }

        if (extras.containsKey(RecognizerIntent.EXTRA_SUPPORTED_LANGUAGES)) {
            supportedLanguagesList = extras.getStringArrayList(RecognizerIntent.EXTRA_SUPPORTED_LANGUAGES);

            JSArray languagesList = new JSArray(supportedLanguagesList);
            call.resolve(new JSObject().put("languages", languagesList));
            return;
        }

        call.reject(ERROR);
    }

    public List<String> getSupportedLanguages() {
        return supportedLanguagesList;
    }

    public String getLanguagePreference() {
        return languagePref;
    }
}
