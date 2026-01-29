const startBtn = document.getElementById('startBtn');
const stopBtn = document.getElementById('stopBtn');
const statusEl = document.getElementById('status');
const transcriptEl = document.getElementById('transcript');

let ws;
let audioContext;
let mediaStream;
let processor;
let isRecording = false;

function setStatus(text) {
  statusEl.textContent = text;
}

function appendTranscript(text) {
  if (!text) return;
  transcriptEl.textContent += text.trim() + '\n';
}

function downsampleBuffer(buffer, inputRate, outputRate) {
  if (outputRate === inputRate) {
    return new Float32Array(buffer);
  }
  const sampleRateRatio = inputRate / outputRate;
  const newLength = Math.round(buffer.length / sampleRateRatio);
  const result = new Float32Array(newLength);
  let offsetResult = 0;
  let offsetBuffer = 0;

  while (offsetResult < result.length) {
    const nextOffsetBuffer = Math.round((offsetResult + 1) * sampleRateRatio);
    let accum = 0;
    let count = 0;
    for (let i = offsetBuffer; i < nextOffsetBuffer && i < buffer.length; i++) {
      accum += buffer[i];
      count++;
    }
    result[offsetResult] = count > 0 ? accum / count : 0;
    offsetResult++;
    offsetBuffer = nextOffsetBuffer;
  }
  return result;
}

async function startRecording() {
  if (isRecording) return;
  isRecording = true;
  transcriptEl.textContent = '';

  const wsUrl = `wss://${location.host}/ws`;
  ws = new WebSocket(wsUrl);
  ws.binaryType = 'arraybuffer';

  ws.onopen = async () => {
    ws.send(JSON.stringify({ type: 'config', sample_rate: 16000 }));
    setStatus('Connecting…');
  };

  ws.onmessage = (event) => {
    const msg = JSON.parse(event.data);
    if (msg.type === 'ready') {
      setStatus('Recording');
    }
    if (msg.type === 'result') {
      appendTranscript(msg.text);
    }
    if (msg.type === 'error') {
      setStatus(msg.message || 'Error');
    }
  };

  ws.onclose = () => {
    setStatus('Disconnected');
  };

  mediaStream = await navigator.mediaDevices.getUserMedia({ audio: true });
  audioContext = new (window.AudioContext || window.webkitAudioContext)({
    sampleRate: 16000,
  });
  const source = audioContext.createMediaStreamSource(mediaStream);
  processor = audioContext.createScriptProcessor(4096, 1, 1);

  processor.onaudioprocess = (event) => {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    const input = event.inputBuffer.getChannelData(0);
    const downsampled = downsampleBuffer(input, audioContext.sampleRate, 16000);
    ws.send(downsampled.buffer);
  };

  source.connect(processor);
  processor.connect(audioContext.destination);

  startBtn.disabled = true;
  stopBtn.disabled = false;
}

async function stopRecording() {
  if (!isRecording) return;
  isRecording = false;

  if (ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'stop' }));
  }

  if (processor) {
    processor.disconnect();
    processor.onaudioprocess = null;
  }
  if (audioContext) {
    await audioContext.close();
  }
  if (mediaStream) {
    mediaStream.getTracks().forEach((track) => track.stop());
  }

  if (ws) {
    ws.close();
  }

  startBtn.disabled = false;
  stopBtn.disabled = true;
  setStatus('Idle');
}

startBtn.addEventListener('click', () => {
  startRecording().catch((err) => {
    console.error(err);
    setStatus('Mic error');
  });
});

stopBtn.addEventListener('click', () => {
  stopRecording().catch(console.error);
});
