#!/bin/bash

# Scriberr 自動化轉錄腳本
#
# 這個腳本會自動執行登入、上傳音檔、等待轉錄完成，並印出結果的完整流程。
#
# 使用方法:
# ./transcribe.sh <音檔路徑>
#
# 範例:
# ./transcribe.sh /path/to/your/audio.wav
#
# 前置需求:
# - curl: 用於發送 API 請求。
# - jq: 用於解析 JSON 回應。 (在大多數 Linux 發行版中可以透過 `sudo apt-get install jq` 或 `sudo yum install jq` 安裝)

# --- 設定 ---
SCRIBERR_URL="http://localhost:3000"
USERNAME="admin"
PASSWORD="password"
LANGUAGE="en"
MODEL_SIZE="base" # 可選 'tiny', 'base', 'small', 'medium', 'large'
DIARIZATION="false" # 是否啟用說話人分離 (true/false)
POLL_INTERVAL=15 # 輪詢間隔（秒）
# --- 設定結束 ---

# 檢查是否提供了音檔路徑
if [ -z "$1" ]; then
    echo "錯誤: 請提供音檔的路徑作為第一個參數。"
    echo "用法: $0 <音檔路徑>"
    exit 1
fi

AUDIO_FILE=$1

# 檢查檔案是否存在
if [ ! -f "$AUDIO_FILE" ]; then
    echo "錯誤: 檔案 '$AUDIO_FILE' 不存在。"
    exit 1
fi

echo "--- 1. 正在進行身分驗證... ---"
AUTH_RESPONSE=$(curl -s -X POST "$SCRIBERR_URL/api/auth" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"$USERNAME\",\"password\":\"$PASSWORD\"}")

TOKEN=$(echo "$AUTH_RESPONSE" | jq -r .accessToken)

if [ "$TOKEN" == "null" ] || [ -z "$TOKEN" ]; then
    echo "錯誤: 身分驗證失敗。請檢查您的使用者名稱和密碼。"
    echo "伺服器回應: $AUTH_RESPONSE"
    exit 1
fi

echo "成功取得 Access Token。"
echo

echo "--- 2. 正在上傳檔案並啟動轉錄... ---"
# 建立轉錄選項的 JSON 字串
TRANSCRIPTION_OPTIONS=$(jq -n \
    --arg lang "$LANGUAGE" \
    --arg model "$MODEL_SIZE" \
    --argjson diarize "$DIARIZATION" \
    '{language: $lang, modelSize: $model, diarization: $diarize}')

UPLOAD_RESPONSE=$(curl -s -X POST "$SCRIBERR_URL/api/upload" \
    -H "Authorization: Bearer $TOKEN" \
    -F "file=@$AUDIO_FILE" \
    -F "options=$TRANSCRIPTION_OPTIONS")

JOB_ID=$(echo "$UPLOAD_RESPONSE" | jq -r .id)

if [ "$JOB_ID" == "null" ] || [ -z "$JOB_ID" ]; then
    echo "錯誤: 檔案上傳失敗。"
    echo "伺服器回應: $UPLOAD_RESPONSE"
    exit 1
fi

echo "檔案上傳成功。轉錄工作 ID: $JOB_ID"
echo

echo "--- 3. 正在等待轉錄結果 (每 $POLL_INTERVAL 秒查詢一次)... ---"
while true; do
    STATUS_RESPONSE=$(curl -s -X GET "$SCRIBERR_URL/api/transcription/$JOB_ID" \
        -H "Authorization: Bearer $TOKEN")

    STATUS=$(echo "$STATUS_RESPONSE" | jq -r .status)
    echo "目前狀態: $STATUS..."

    if [ "$STATUS" == "completed" ]; then
        echo
        echo "--- 轉錄完成！ ---"
        TRANSCRIPT=$(echo "$STATUS_RESPONSE" | jq -r .transcript.text)
        echo "$TRANSCRIPT"
        exit 0
    elif [ "$STATUS" == "failed" ]; then
        echo
        echo "--- 轉錄失敗！ ---"
        ERROR_MESSAGE=$(echo "$STATUS_RESPONSE" | jq -r .error)
        echo "錯誤訊息: $ERROR_MESSAGE"
        exit 1
    fi

    sleep $POLL_INTERVAL
done
